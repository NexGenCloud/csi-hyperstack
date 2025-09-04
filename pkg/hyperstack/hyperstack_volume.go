package hyperstack

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/NexGenCloud/hyperstack-sdk-go/lib/clusters"
	"github.com/NexGenCloud/hyperstack-sdk-go/lib/volume"
	"github.com/NexGenCloud/hyperstack-sdk-go/lib/volume_attachment"

	"golang.org/x/net/context"
	"k8s.io/csi-hyperstack/pkg/metrics"
)

const (
	volumeDescription = "Created by Hyperstack CSI driver"
)

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

	fmt.Printf("Listing volumes with name filter %q\n", n)
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
			res = append(res, row)
		}
	}
	fmt.Printf("Found %d volumes matching name filter %q\n", len(res), n)
	return res, nil
}

// GetVolume retrieves Volume by its ID.
func (hs *Hyperstack) GetVolume(ctx context.Context, volumeID string) (*volume.VolumeFields, error) {
	// TODO(joseb): gosdk doesn't have get operation. Need to list all and search.
	client, err := volume.NewClientWithResponses(
		hs.Client.ApiServer,
		volume.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}
	result, err := client.ListVolumesWithResponse(ctx, &volume.ListVolumesParams{})

	fmt.Println("AttachVolumeToNode status code:", result.StatusCode())
	fmt.Println("Response HTTP:", result.HTTPResponse)

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
	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume attachment response is nil")
	}

	callResult := result.JSON200.Volumes
	if callResult == nil {
		return nil, nil
	}

	for _, row := range *callResult {
		if strconv.Itoa(*row.Id) == volumeID {
			return &row, nil
		}
	}
	return nil, nil
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
			Description:     func() string { s := volumeDescription; return s }(),
		},
	)

	if err != nil {
		fmt.Printf("Error creating volume %q (size: %d GB, type: %s, env: %s): %v\n", name, size, vtype, environment, err)
		return nil, fmt.Errorf("failed to create volume %q (size: %d GB, type: %s, env: %s): %w", name, size, vtype, environment, err)
	}

	if result == nil {
		return nil, fmt.Errorf("received nil response from volume creation API for volume %q", name)
	}

	fmt.Printf("CreateVolume %q status code: %d\n", name, result.StatusCode())
	fmt.Printf("Response HTTP for volume %q: %+v\n", name, result.HTTPResponse)

	if result.JSON404 != nil {
		fmt.Printf("Error response for volume %q: %+v\n", name, result.JSON404)
		return nil, fmt.Errorf("received 404 error for volume %q: %v", name, result.JSON404)
	}

	if mc.ObserveRequest(err) != nil {
		return nil, fmt.Errorf("metrics observation failed for volume %q: %w", name, err)
	}

	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume creation result is nil for volume %q", name)
	}

	vol := result.JSON200.Volume
	if vol == nil {
		return nil, fmt.Errorf("volume creation result includes nil volume object for volume %q", name)
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

func (hs *Hyperstack) AttachVolumeToNode(ctx context.Context, virtualMachineId int, volumeID string) (*volume_attachment.AttachVolumeFields, error) {
	client, err := volume_attachment.NewClientWithResponses(
		hs.Client.ApiServer,
		volume_attachment.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}
	volumeIDInt, err := strconv.Atoi(volumeID)
	if err != nil {
		return nil, err
	}

	result, err := client.AttachVolumesToVirtualMachineWithResponse(
		ctx,
		virtualMachineId,
		volume_attachment.AttachVolumesPayload{
			VolumeIds: &[]int{volumeIDInt},
		},
	)

	fmt.Println("AttachVolumeToNode status code:", result.StatusCode())
	fmt.Println("Response HTTP:", result.HTTPResponse)

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

func (hs *Hyperstack) DetachVolumeFromNode(ctx context.Context, virtualMachineId int, volumeID string) (*volume_attachment.DetachVolumes, error) {
	client, err := volume_attachment.NewClientWithResponses(
		hs.Client.ApiServer,
		volume_attachment.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}
	volumeIDInt, err := strconv.Atoi(volumeID)
	if err != nil {
		return nil, err
	}

	result, err := client.DetachVolumesFromVirtualMachineWithResponse(
		ctx,
		virtualMachineId,
		volume_attachment.DetachVolumesPayload{
			VolumeIds: &[]int{volumeIDInt},
		},
	)

	fmt.Println("AttachVolumeToNode status code:", result.StatusCode())
	fmt.Println("Response HTTP:", result.HTTPResponse)

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
	if result.JSON405 != nil {
		return nil, fmt.Errorf("volume attachment failed with 405 error: %v", result.JSON405)
	}
	if result.JSON200 == nil {
		return nil, fmt.Errorf("volume attachment response is nil")
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
