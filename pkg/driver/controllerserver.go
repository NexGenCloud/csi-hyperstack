package driver

import (
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	// "github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/backups"
	// "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"github.com/NexGenCloud/hyperstack-sdk-go/lib/volume"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// "k8s.io/cloud-provider-openstack/pkg/csi/cinder/openstack"
	"k8s.io/csi-hyperstack/pkg/utils"
	// cpoerrors "k8s.io/cloud-provider-openstack/pkg/util/errors"
	"k8s.io/klog/v2"
)

type controllerServer struct {
	driver *Driver

	csi.UnimplementedControllerServer
}

const (
	hyperstackCSIClusterIDKey = "hyperstack.csi.nexgencloud.com/cluster"
)

func (cs *controllerServer) CreateVolume(
	ctx context.Context,
	req *csi.CreateVolumeRequest,
) (*csi.CreateVolumeResponse, error) {
	if err := cs.driver.ValidateControllerServiceRequest(
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	); err != nil {
		klog.Errorf("invalid create volume req: %v", req)
		return nil, err
	}

	klog.V(4).Infof("CreateVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	// Volume Name
	volName := req.GetName()
	volCapabilities := req.GetVolumeCapabilities()

	if len(volName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "[CreateVolume] missing Volume Name")
	}

	if volCapabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "[CreateVolume] missing Volume capability")
	}

	// Volume Size - Default is 1 GiB
	volSizeBytes := int64(1 * 1024 * 1024 * 1024)
	if req.GetCapacityRange() != nil {
		volSizeBytes = int64(req.GetCapacityRange().GetRequiredBytes())
	}
	volSizeGB := int(util.RoundUpSize(volSizeBytes, 1024*1024*1024))

	// Volume Type
	volType := req.GetParameters()["type"]

	// First check if volEnvironment is already specified, if not get preferred from Topology
	volEnvironment := req.GetParameters()["environment"]
	if volEnvironment == "" {
		// Check from Topology
		if req.GetAccessibilityRequirements() != nil {
			volEnvironment = util.GetEnvFromTopology(topologyKey, req.GetAccessibilityRequirements())
		}
	}

	cloud := cs.driver.hyperstackClient

	// Verify a volume with the provided name doesn't already exist for this tenant
	volumes, err := cloud.GetVolumesByName(ctx, volName)
	if err != nil {
		klog.Errorf("Failed to query for existing Volume during CreateVolume: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to get volumes: %v", err)
	}

	if len(volumes) == 1 {
		if volSizeGB != *volumes[0].Size {
			return nil, status.Error(codes.AlreadyExists, "Volume Already exists with same name and different capacity")
		}
		klog.V(4).Infof("Volume %d already exists in Environment %s: size %d GiB", *volumes[0].Id, *volumes[0].Environment.Name, *volumes[0].Size)
		return getCreateVolumeResponse(&volumes[0], req.GetAccessibilityRequirements()), nil
	} else if len(volumes) > 1 {
		klog.V(3).Infof("found multiple existing volumes with selected name (%s) during create", volName)
		return nil, status.Error(codes.Internal, "Multiple volumes reported by Cinder with same name")
	}

	// Volume Create
	properties := map[string]string{
		hyperstackCSIClusterIDKey: cs.driver.opts.HyperstackClusterId,
	}
	//Tag volume with metadata if present: https://github.com/kubernetes-csi/external-provisioner/pull/399
	for _, mKey := range []string{"csi.storage.k8s.io/pvc/name", "csi.storage.k8s.io/pvc/namespace", "csi.storage.k8s.io/pv/name"} {
		if v, ok := req.Parameters[mKey]; ok {
			properties[mKey] = v
		}
	}

	// TODO(joseb): Once infrahub api supports volume creation from a source volume, volume snapshot and backup functions,
	// 				we need to process the cases for volume creation from another volume, snapshot, and backup when volumeContentSource is not empty.
	// content := req.GetVolumeContentSource()

	vol, err := cloud.CreateVolume(ctx, volName, volSizeGB, volType, volEnvironment, properties)
	// When creating a volume from a backup, the response does not include the backupID.

	if err != nil {
		klog.Errorf("Failed to CreateVolume: %v", err)
		return nil, status.Errorf(codes.Internal, "CreateVolume failed with error %v", err)
	}

	klog.V(4).Infof("CreateVolume: Successfully created volume %d in Environment: %s of size %d GiB", *vol.Id, *vol.Environment.Name, *vol.Size)
	return getCreateVolumeResponse(vol, req.GetAccessibilityRequirements()), nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if err := cs.driver.ValidateControllerServiceRequest(
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	); err != nil {
		klog.Errorf("invalid delete volume req: %v", req)
		return nil, err
	}

	klog.V(4).Infof("DeleteVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("DeleteVolume: Not implemented yet!")
	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	klog.V(4).Infof("ControllerPublishVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("DeleteVolume: Not implemented yet!")
	return &csi.ControllerPublishVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	klog.V(4).Infof("ControllerUnpublishVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("DeleteVolume: Not implemented yet!")
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (cs *controllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	klog.V(4).Infof("ListVolumes: called with %+#v request", req)
	klog.Warning("DeleteVolume: Not implemented yet!")
	return &csi.ListVolumesResponse{}, nil
}

func (cs *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	klog.V(4).Infof("CreateSnapshot: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("DeleteVolume: Not implemented yet!")
	return &csi.CreateSnapshotResponse{}, nil
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	klog.V(4).Infof("DeleteSnapshot: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("DeleteVolume: Not implemented yet!")
	return &csi.DeleteSnapshotResponse{}, nil
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	// TODO(joseb)
	klog.Warning("DeleteVolume: Not implemented yet!")
	return &csi.ListSnapshotsResponse{}, nil
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities
func (cs *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.V(5).Infof("Using default ControllerGetCapabilities")

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.driver.cscap,
	}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {

	reqVolCap := req.GetVolumeCapabilities()

	if len(reqVolCap) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities Volume Capabilities must be provided")
	}
	volumeID := req.GetVolumeId()

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities Volume ID must be provided")
	}

	_, err := cs.driver.hyperstackClient.GetVolume(ctx, volumeID)
	if err != nil {
		// if cpoerrors.IsNotFound(err) {
		// 	return nil, status.Errorf(codes.NotFound, "ValidateVolumeCapabilities Volume %s not found", volumeID)
		// }
		return nil, status.Errorf(codes.Internal, "ValidateVolumeCapabilities %v", err)
	}

	for _, cap := range reqVolCap {
		if cap.GetAccessMode().GetMode() != cs.driver.vcap[0].Mode {
			return &csi.ValidateVolumeCapabilitiesResponse{Message: "Requested Volume Capability not supported"}, nil
		}
	}

	// Cinder CSI driver currently supports one mode only
	resp := &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: []*csi.VolumeCapability{
				{
					AccessMode: cs.driver.vcap[0],
				},
			},
		},
	}

	return resp, nil
}

func (cs *controllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetCapacity is not yet implemented")
}

func (cs *controllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	klog.V(4).Infof("ControllerGetVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	return &csi.ControllerGetVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	klog.V(4).Infof("ControllerExpandVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("DeleteVolume: Not implemented yet!")
	return &csi.ControllerExpandVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerModifyVolume(ctx context.Context, req *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	klog.V(4).Infof("ControllerModifyVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("ModifyVolume: Not implemented yet!")
	return &csi.ControllerModifyVolumeResponse{}, nil
}

func getCreateVolumeResponse(vol *volume.VolumeFields, accessibleTopologyReq *csi.TopologyRequirement) *csi.CreateVolumeResponse {

	var volsrc *csi.VolumeContentSource
	var accessibleTopology []*csi.Topology

	if accessibleTopologyReq != nil {
		accessibleTopology = accessibleTopologyReq.GetPreferred()
	}

	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:           strconv.Itoa(*vol.Id),
			CapacityBytes:      int64(*vol.Size * 1024 * 1024 * 1024),
			AccessibleTopology: accessibleTopology,
			ContentSource:      volsrc,
		},
	}

	return resp
}
