package hyperstack

import (
	"fmt"
	"strings"

	"github.com/NexGenCloud/hyperstack-sdk-go/lib/clusters"
	"github.com/NexGenCloud/hyperstack-sdk-go/lib/volume"
	"github.com/NexGenCloud/hyperstack-sdk-go/lib/volume_attachment"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"

	"golang.org/x/net/context"
	"k8s.io/csi-hyperstack/pkg/metrics"
)

var volumeDescription = "Created by Hyperstack CSI driver"

// GetVolumesByName is a wrapper around ListVolumes that creates a Name filter to act as a GetByName
// Returns a list of Volume references with the specified name
func (hs *Hyperstack) GetVolumesByName(ctx context.Context, n string) ([]volume.VolumeFields, error) {
	if hs.Client == nil {
		return nil, fmt.Errorf("hyperstack client is not initialized")
	}

	client, err := volume.NewClientWithResponses(
		hs.Client.ApiServer,
		volume.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume client: %w", err)
	}

	result, err := client.ListVolumesWithResponse(ctx, &volume.ListVolumesParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("received nil response from volume list API")
	}

	fmt.Printf("ListVolumes status code: %d\n", result.StatusCode())
	fmt.Printf("Response HTTP: %+v\n", result.HTTPResponse)

	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume list result is nil (status code: %d)", result.StatusCode())
	}

	callResult := result.JSON200.Volumes
	if callResult == nil {
		return nil, fmt.Errorf("volume list is nil in the response")
	}

	res := []volume.VolumeFields{}
	for _, row := range *callResult {
		if row.Name != nil && strings.Contains(*row.Name, n) {
			res = append(res, volume.VolumeFields{
				Attachments: row.Attachments,
				Bootable:    row.Bootable,
				CallbackUrl: row.CallbackUrl,
				CreatedAt:   row.CreatedAt,
				Description: row.Description,
				Environment: row.Environment,
				Id:          row.Id,
				ImageId:     row.ImageId,
				Name:        row.Name,
				Size:        row.Size,
				Status:      row.Status,
				UpdatedAt:   row.UpdatedAt,
				VolumeType:  row.VolumeType,
			})
		}
	}
	return res, nil
}

// GetVolume retrieves Volume by its ID.
func (hs *Hyperstack) GetVolume(ctx context.Context, volumeID int) (*volume.VolumeFields, error) {
	client, err := volume.NewClientWithResponses(
		hs.Client.ApiServer,
		volume.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}
	result, err := client.FetchVolumeDetailsWithResponse(ctx, volumeID)
	if err != nil {
		return nil, err
	}
	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume details response is nil")
	}
	if result.JSON400 != nil {
		return nil, fmt.Errorf("volume details failed with 400 error: %v", result.JSON400)
	}
	if result.JSON401 != nil {
		return nil, fmt.Errorf("volume details failed with 401 error: %v", result.JSON401)
	}
	if result.JSON405 != nil {
		return nil, fmt.Errorf("volume details failed with 405 error: %v", result.JSON405)
	}
	attachments := []volume.AttachmentsFieldsForVolume{}
	for _, attachment := range *result.JSON200.Volume.Attachments {
		if *attachment.Status == "ATTACHED" {
			attachments = append(attachments, attachment)
		}
	}
	response := volume.VolumeFields{
		Attachments: &attachments,
		Bootable:    result.JSON200.Volume.Bootable,
		CallbackUrl: result.JSON200.Volume.CallbackUrl,
		CreatedAt:   result.JSON200.Volume.CreatedAt,
		Description: result.JSON200.Volume.Description,
		Environment: result.JSON200.Volume.Environment,
		Id:          result.JSON200.Volume.Id,
		ImageId:     result.JSON200.Volume.ImageId,
		Name:        result.JSON200.Volume.Name,
		Size:        result.JSON200.Volume.Size,
		Status:      result.JSON200.Volume.Status,
		UpdatedAt:   result.JSON200.Volume.UpdatedAt,
		VolumeType:  result.JSON200.Volume.VolumeType,
	}
	return &response, nil
}

// CreateVolume creates a volume of given size
func (hs *Hyperstack) CreateVolume(ctx context.Context, name string, size int, vtype, environment string, tags map[string]string) (*volume.VolumeFields, error) {
	client, err := volume.NewClientWithResponses(
		hs.Client.ApiServer,
		volume.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}
	fmt.Println("Payload for create volume", name, size, vtype, environment, tags)
	mc := metrics.NewMetricContext("volume", "create")
	result, err := client.CreateVolumeWithResponse(
		ctx,
		volume.CreateVolumePayload{
			Name:            name,
			Size:            size,
			VolumeType:      vtype,
			EnvironmentName: environment,
			Description:     &volumeDescription,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create volume %s (size: %d GB, type: %s, env: %s): %w", name, size, vtype, environment, err)
	}

	if result == nil {
		return nil, fmt.Errorf("received nil response from volume creation API for volume %s", name)
	}

	if result.JSON404 != nil {
		return nil, fmt.Errorf("received 404 error for volume %s: %v", name, result.JSON404)
	}

	if mc.ObserveRequest(err) != nil {
		return nil, fmt.Errorf("metrics observation failed for volume %s: %w", name, err)
	}

	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume creation result is nil for volume %s", name)
	}

	vol := result.JSON200.Volume
	if vol == nil {
		return nil, fmt.Errorf("volume creation result includes nil volume object for volume %s", name)
	}

	return vol, nil
}

func (hs *Hyperstack) DeleteVolume(ctx context.Context, volumeID int) error {
	client, err := volume.NewClientWithResponses(
		hs.Client.ApiServer,
		volume.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return err
	}

	result, err := client.DeleteVolumeWithResponse(ctx, volumeID)

	if err != nil {
		return err
	}

	if result.JSON400 != nil {
		return fmt.Errorf("volume deletion failed with 400 error: %v", result.JSON400)
	}
	if result.JSON401 != nil {
		return fmt.Errorf("volume deletion failed with 401 error: %v", result.JSON401)
	}
	if result.JSON404 != nil {
		return fmt.Errorf("volume deletion failed with 404 error: %v", result.JSON404)
	}
	if result.JSON200 == nil {
		return fmt.Errorf("volume deletion response is nil")
	}

	return nil
}

func (hs *Hyperstack) AttachVolumeToNode(ctx context.Context, virtualMachineId int, volumeID int) (*volume_attachment.AttachVolumeFields, error) {
	client, err := volume_attachment.NewClientWithResponses(
		hs.Client.ApiServer,
		volume_attachment.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}
	var protected = true
	result, err := client.AttachVolumesToVirtualMachineWithResponse(
		ctx,
		virtualMachineId,
		volume_attachment.AttachVolumesPayload{
			VolumeIds: &[]int{volumeID},
			Protected: &protected,
		},
	)

	if err != nil {
		return nil, err
	}

	// Check if result is nil
	if result == nil {
		return nil, fmt.Errorf("received nil response from volume attachment API")
	}

	// Check for error status codes
	if result.JSON400 != nil {
		return nil, fmt.Errorf("volume attachment failed with 400 error: %v", result.JSON400)
	}
	if result.JSON401 != nil {
		return nil, fmt.Errorf("volume attachment failed with 401 error: %v", result.JSON401)
	}
	if result.JSON404 != nil {
		return nil, fmt.Errorf("volume attachment failed with 404 error: %v", result.JSON404)
	}
	if result.JSON405 != nil {
		return nil, fmt.Errorf("volume attachment failed with 405 error: %v", result.JSON405)

	}
	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume attachment response is nil")
	}

	attachments := *result.JSON200.VolumeAttachments
	return &attachments[0], nil
}

func (hs *Hyperstack) UpdateVolumeAttachment(ctx context.Context, volumeId int) (*volume_attachment.UpdateAVolumeAttachmentResponse, error) {
	client, err := volume_attachment.NewClientWithResponses(
		hs.Client.ApiServer,
		volume_attachment.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}
	getVolume, err := hs.GetVolume(ctx, volumeId)
	if err != nil {
		return nil, err
	}

	var volumeAttachmentID = *(*getVolume.Attachments)[0].Id

	var protected = false
	result, err := client.UpdateAVolumeAttachmentWithResponse(ctx, volumeAttachmentID, volume_attachment.UpdateVolumeAttachmentPayload{Protected: &protected})
	if result == nil {
		return nil, fmt.Errorf("received nil response from volume attachment API")
	}
	if result.JSON400 != nil {
		return nil, fmt.Errorf("volume attachment update failed with 400 error: %v", result.JSON400)
	}
	if result.JSON401 != nil {
		return nil, fmt.Errorf("volume attachment update failed with 401 error: %v", result.JSON401)
	}
	if result.JSON404 != nil {
		return nil, fmt.Errorf("volume attachment update failed with 404 error: %v", protosanitizer.StripSecrets(result.JSON404))
	}
	if result.JSON405 != nil {
		return nil, fmt.Errorf("volume attachment update failed with 405 error: %v", result.JSON405)
	}
	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume attachment update response is nil")
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (hs *Hyperstack) DetachVolumeFromNode(ctx context.Context, virtualMachineId int, volumeID int) (*volume_attachment.DetachVolumes, error) {
	client, err := volume_attachment.NewClientWithResponses(
		hs.Client.ApiServer,
		volume_attachment.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}

	_, err = hs.UpdateVolumeAttachment(ctx, volumeID)
	if err != nil {
		return nil, err
	}

	result, err := client.DetachVolumesFromVirtualMachineWithResponse(
		ctx,
		virtualMachineId,
		volume_attachment.DetachVolumesPayload{
			VolumeIds: &[]int{volumeID},
		},
	)

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("received nil response from volume detachment API")
	}
	if result.JSON400 != nil {
		return nil, fmt.Errorf("volume detachment failed with 400 error: %v", result.JSON400)
	}
	if result.JSON401 != nil {
		return nil, fmt.Errorf("volume detachment failed with 401 error: %v", result.JSON401)
	}
	if result.JSON404 != nil {
		return nil, fmt.Errorf("volume detachment failed with 404 error: %v", result.JSON404)
	}
	if result.JSON405 != nil {
		return nil, fmt.Errorf("volume detachment failed with 405 error: %v", result.JSON405)
	}
	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume detachment response is nil")
	}
	message := *result.JSON200.Message
	status := *result.JSON200.Status
	attachments := *result.JSON200.VolumeAttachments
	return &volume_attachment.DetachVolumes{
		Message:           &message,
		Status:            &status,
		VolumeAttachments: &attachments,
	}, nil
}

func (hs *Hyperstack) GetClusterDetail(ctx context.Context, clusterID int) (*clusters.ClusterFields, error) {
	fmt.Printf("Getting cluster detail for cluster ID: %d\n", clusterID)
	client, err := clusters.NewClientWithResponses(
		hs.Client.ApiServer,
		clusters.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}

	result, err := client.GettingClusterDetailWithResponse(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("received nil response from volume attachment API")
	}
	if result.JSON400 != nil {
		return nil, fmt.Errorf("volume attachment failed with 400 error: %v", result.JSON400)
	}
	if result.JSON401 != nil {
		return nil, fmt.Errorf("volume attachment failed with 401 error: %v", result.JSON401)
	}
	if result.JSON404 != nil {
		return nil, fmt.Errorf("volume attachment failed with 404 error: %v", result.JSON404)
	}
	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume attachment response is nil")
	}
	return result.JSON200.Cluster, nil
}
