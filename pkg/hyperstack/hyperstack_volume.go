package hyperstack

import (
	"errors"
	"github.com/NexGenCloud/hyperstack-sdk-go/lib/volume"
	"golang.org/x/net/context"
	"k8s.io/csi-hyperstack/pkg/metrics"
	"strconv"
	"strings"
)

const (
	volumeDescription = "Created by Hyperstack CSI driver"
)

// GetVolumesByName is a wrapper around ListVolumes that creates a Name filter to act as a GetByName
// Returns a list of Volume references with the specified name
func (hs *Hyperstack) GetVolumesByName(ctx context.Context, n string) ([]volume.VolumeFields, error) {
	client, err := volume.NewClientWithResponses(
		hs.Client.ApiServer,
		volume.WithRequestEditorFn(hs.Client.GetAddHeadersFn()),
	)
	if err != nil {
		return nil, err
	}
	result, err := client.ListVolumesWithResponse(ctx, &volume.ListVolumesParams{})
	if err != nil {
		return nil, err
	}

	if result.JSON200 == nil {
		return nil, nil
	}

	callResult := result.JSON200.Volumes
	if callResult == nil {
		return nil, nil
	}

	res := []volume.VolumeFields{}
	for _, row := range *callResult {
		if strings.Contains(*row.Name, n) {
			res = append(res, row)
		}
	}
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
	if err != nil {
		return nil, err
	}

	if result.JSON200 == nil {
		return nil, nil
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
	return &volume.VolumeFields{}, nil
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
	if mc.ObserveRequest(err) != nil {
		return nil, err
	}

	if result.JSON200 == nil {
		return nil, errors.New("Volume creation result is nil")
	}

	vol := result.JSON200.Volume
	if vol == nil {
		return nil, errors.New("Volume creation result includes nil volume object")
	}

	return vol, nil
}
