package hyperstack

import (
	"github.com/NexGenCloud/hyperstack-sdk-go/lib/clusters"
	"github.com/NexGenCloud/hyperstack-sdk-go/lib/volume"
	"github.com/NexGenCloud/hyperstack-sdk-go/lib/volume_attachment"
	"golang.org/x/net/context"
	"k8s.io/csi-hyperstack/pkg/utils/metadata"
)

// https://github.com/kubernetes/cloud-provider-openstack/blob/master/pkg/csi/cinder/openstack/openstack.go#L47
type IHyperstack interface {
	CreateVolume(ctx context.Context, name string, size int, vtype, environment string, tags map[string]string) (*volume.VolumeFields, error)
	GetVolume(ctx context.Context, volumeID string) (*volume.VolumeFields, error)
	GetVolumesByName(ctx context.Context, name string) ([]volume.VolumeFields, error)
	DeleteVolume(ctx context.Context, volumeID int) error
	GetMetadataOpts() metadata.Opts
	AttachVolumeToNode(ctx context.Context, virtualMachineId int, volumeID string) (*volume_attachment.AttachVolumeFields, error)
	DetachVolumeFromNode(ctx context.Context, virtualMachineId int, volumeID string) (*volume_attachment.DetachVolumes, error)
	GetClusterDetail(ctx context.Context, clusterID int) (*clusters.ClusterFields, error)
}

type Hyperstack struct {
	Client       *HyperstackClient
	metadataOpts metadata.Opts
}

func (hs *Hyperstack) GetMetadataOpts() metadata.Opts {
	return hs.metadataOpts
}
