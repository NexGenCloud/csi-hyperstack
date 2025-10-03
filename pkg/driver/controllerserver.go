package driver

import (
	"strconv"
	"time"

	"github.com/NexGenCloud/hyperstack-sdk-go/lib/volume"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	util "k8s.io/csi-hyperstack/pkg/utils"
	kubernetes "k8s.io/csi-hyperstack/pkg/utils/kubernetes"
	"k8s.io/csi-hyperstack/pkg/utils/metadata"
	"k8s.io/klog/v2"
)

type controllerServer struct {
	driver *Driver
	csi.UnimplementedControllerServer
}

const (
	hyperstackCSIClusterIDKey    = "hyperstack.csi.nexgencloud.com/cluster"
	hyperstackEnvironmentNameKey = "hyperstack.csi.nexgencloud.com/environment"
	hyperstackClusterIdLabelKey  = "hyperstack.cloud/cluster-id"
)

func (cs *controllerServer) CreateVolume(
	ctx context.Context,
	req *csi.CreateVolumeRequest,
) (*csi.CreateVolumeResponse, error) {
	klog.Infof("==============Create volume called ==============")
	klog.Infof("CreateVolume: Driver opts: %#v\n", cs.driver.opts)
	if err := cs.driver.ValidateControllerServiceRequest(
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	); err != nil {
		klog.Errorf("invalid create volume req: %v", req)
		return nil, err
	}
	volName := req.GetName()
	volCapabilities := req.GetVolumeCapabilities()

	if len(volName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume: missing Volume Name")
	}

	if volCapabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume: missing Volume capability")
	}

	volSizeBytes := int64(1 * 1024 * 1024 * 1024)
	if req.GetCapacityRange() != nil {
		volSizeBytes = int64(req.GetCapacityRange().GetRequiredBytes())
	}
	volSizeGB := int(util.RoundUpSize(volSizeBytes, 1024*1024*1024))
	volType := req.GetParameters()["type"]
	cloud := cs.driver.hyperstackClient
	volumes, err := cloud.GetVolumesByName(ctx, volName)
	if err != nil {
		klog.Errorf("CreateVolume: Failed to query for existing Volume during CreateVolume: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to get volumes: %v", err)
	}

	if len(volumes) == 1 {
		if volSizeGB != *volumes[0].Size {
			return nil, status.Error(codes.AlreadyExists, "CreateVolume: Volume Already exists with same name and different capacity")
		}
		klog.Infof("CreateVolume: Volume %d already exists in Environment %s: size %d GiB", *volumes[0].Id, *volumes[0].Environment.Name, *volumes[0].Size)
		return getCreateVolumeResponse(&volumes[0], req.GetAccessibilityRequirements()), nil
	} else if len(volumes) > 1 {
		klog.Infof("CreateVolume: found multiple existing volumes with selected name (%s) during create", volName)
		return nil, status.Error(codes.Internal, "CreateVolume: Multiple volumes reported by Cinder with same name")
	}
	clusterId, err := kubernetes.GetNodeLabel(hyperstackClusterIdLabelKey)
	if err != nil {
		klog.Errorf("failed to get node label: %v", err)
	}
	klog.Infof("CreateVolume: Cluster label -\n%s:%s", hyperstackClusterIdLabelKey, clusterId)

	properties := map[string]string{
		hyperstackCSIClusterIDKey: clusterId,
	}
	for _, mKey := range []string{"csi.storage.k8s.io/pvc/name", "csi.storage.k8s.io/pvc/namespace", "csi.storage.k8s.io/pv/name"} {
		if v, ok := req.Parameters[mKey]; ok {
			properties[mKey] = v
		}
	}

	clusterIdInt, err := strconv.Atoi(clusterId)
	if err != nil {
		klog.Errorf("CreateVolume: Failed to convert cluster ID to int: %v", err)
		return nil, status.Errorf(codes.Internal, "CreateVolume failed with error %v", err)
	}
	clusterDetail, err := cloud.GetClusterDetail(ctx, clusterIdInt)
	if err != nil {
		klog.Errorf("CreateVolume: Failed to GetClusterDetail: %v", err)
		return nil, status.Errorf(codes.Internal, "CreateVolume failed with error %v", err)
	}
	volEnvironment := *clusterDetail.EnvironmentName
	klog.Infof("CreateVolume: Creating volume %s with size %d GiB in Environment: %s", volName, volSizeGB, volEnvironment)
	vol, err := cloud.CreateVolume(ctx, volName, volSizeGB, volType, volEnvironment, properties)
	if err != nil {
		klog.Errorf("CreateVolume: Failed to CreateVolume: %v", err)
		return nil, status.Errorf(codes.Internal, "CreateVolume failed with error %v", err)
	}
	maxAttempts := 15
	klog.Infof("CreateVolume: Polling for volume to be available with max attempts: %d", maxAttempts)
	for i := 0; i < maxAttempts; i++ {
		klog.Infof("CreateVolume: Polling for volume to be available %d/%d", i+1, maxAttempts)
		v, err := cloud.GetVolume(ctx, *vol.Id)
		if err != nil {
			klog.Errorf("CreateVolume: Failed to GetVolume while polling volume availability: %v", err)
			return nil, status.Errorf(codes.Internal, "CreateVolume failed with error %v", err)
		}
		if v == nil {
			klog.Warningf("CreateVolume: GetVolume attempt %d returned nil volume", i+1)
			time.Sleep(2 * time.Second)
			continue
		} else {
			klog.Infof("CreateVolume: GetVolume attempt %d returned volume: %+v", i+1, protosanitizer.StripSecrets(v))
			if *v.Status == "available" {
				klog.Infof("CreateVolume: Volume is now available-\nID: %v\nStatus: %v\nVolume Name: %v", *v.Id, *v.Status, *v.Name)
				vol = v
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	klog.Infof("CreateVolume: Volume Successfully created-Volume Name: %s\nEnvironment: %s\nSize: %d GiB\nStatus: %s", *vol.Name, *vol.Environment.Name, *vol.Size, *vol.Status)
	return getCreateVolumeResponse(vol, req.GetAccessibilityRequirements()), nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.Infof("\n==============DeleteVolume: called================\n")
	klog.Infof("DeleteVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	volumeID := req.GetVolumeId()
	cloud := cs.driver.hyperstackClient
	volumeIDInt, err := strconv.Atoi(volumeID)
	if err != nil {
		klog.Errorf("DeleteVolume: Failed to convert volume ID to int: %v", err)
		return nil, status.Errorf(codes.Internal, "DeleteVolume: Failed to convert volume ID to int: %v", err)
	}
	getVolume, err := cloud.GetVolume(ctx, volumeIDInt)
	if err != nil {
		klog.Errorf("DeleteVolume: Failed to GetVolume from hyperstack: %v", err)
		return nil, status.Errorf(codes.NotFound, "DeleteVolume: Failed to GetVolume from hyperstack: %v", err)
	}
	if getVolume == nil {
		klog.Errorf("DeleteVolume: GetVolume returned nil volume")
		return nil, status.Errorf(codes.NotFound, "DeleteVolume: GetVolume returned nil volume")
	}
	if *getVolume.Status == "in-use" {
		klog.Errorf("DeleteVolume: Volume %s is in use", *getVolume.Name)
		return nil, status.Errorf(codes.FailedPrecondition, "DeleteVolume: Volume %s is in use", *getVolume.Name)
	}
	if *getVolume.Status == "available" {
		klog.Infof("DeleteVolume: Volume %s is available", *getVolume.Name)
		err = cloud.DeleteVolume(ctx, volumeIDInt)
		if err != nil {
			klog.Errorf("DeleteVolume: Failed to DeleteVolume from hyperstack: %v", err)
			return nil, status.Errorf(codes.Internal, "DeleteVolume: Failed to DeleteVolume from hyperstack: %v", err)
		}
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	klog.Infof("\n==============ControllerPublishVolume: called================\n")
	klog.Infof("ControllerPublishVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	searchOrder := "metadataService,configDrive"
	metadataProvider := metadata.GetMetadataProvider(searchOrder)
	metadataProvider.GetHyperstackVMId()
	virtualMachineId := req.NodeId
	vmId, err := strconv.Atoi(virtualMachineId)
	if err != nil {
		klog.Errorf("ControllerPublishVolume: Failed to convert virtual machine ID to int: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to convert virtual machine ID to int: %v", err)
	}
	volumeID := req.GetVolumeId()
	volumeIDInt, err := strconv.Atoi(volumeID)
	if err != nil {
		klog.Errorf("ControllerPublishVolume: Failed to convert volume ID to int: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to convert volume ID to int: %v", err)
	}
	klog.Infof("ControllerPublishVolume: VM and Volume ID while attaching volume to node: %d, %d", vmId, volumeIDInt)
	cloud := cs.driver.hyperstackClient
	getVolume, err := cloud.GetVolume(ctx, volumeIDInt)
	klog.Infof("ControllerPublishVolume: GetVolume returned volume: %+v", getVolume)
	if err != nil {
		klog.Errorf("ControllerPublishVolume: Failed to GetVolume from hyperstack: %v", err)
		return nil, status.Errorf(codes.NotFound, "ControllerPublishVolume: Failed to GetVolume from hyperstack: %v", err)
	}
	if getVolume == nil {
		klog.Errorf("ControllerPublishVolume: GetVolume returned nil volume")
		return nil, status.Errorf(codes.NotFound, "ControllerPublishVolume: GetVolume returned nil volume")
	}
	klog.Infof("ControllerPublishVolume: GetVolume succeeded -\nStatus: %s\nName: %s\nID: %d\nSize:%d", *getVolume.Status, *getVolume.Name, *getVolume.Id, *getVolume.Size)
	if *getVolume.Status == "in-use" {
		klog.Infof("ControllerPublishVolume: Volume %s is already in use", *getVolume.Name)
		if len(*getVolume.Attachments) > 0 {
			attachedInstanceId := *(*getVolume.Attachments)[0].InstanceId
			if err != nil {
				klog.Errorf("ControllerPublishVolume: Failed to convert attached instance ID to int: %v", err)
				return nil, status.Errorf(codes.Internal, "Failed to convert attached instance ID to int: %v", err)
			}
			klog.Infof("ControllerPublishVolume: Volume %s is already attached to node %d", *getVolume.Name, attachedInstanceId)
			if attachedInstanceId == vmId {
				return &csi.ControllerPublishVolumeResponse{
					PublishContext: map[string]string{
						volNameKeyFromControllerPublishVolume: *(*getVolume.Attachments)[0].Device,
					},
				}, nil
			}
			klog.Infof("ControllerPublishVolume: Volume attached to wrong node (attached: %d, requested: %d), detaching", attachedInstanceId, vmId)
			_, err = cloud.DetachVolumeFromNode(ctx, attachedInstanceId, volumeIDInt)
			if err != nil {
				klog.Errorf("ControllerPublishVolume: Failed to detach from wrong node: %v", err)
				return nil, status.Errorf(codes.Internal, "ControllerPublishVolume: Failed to detach from wrong node: %v", err)
			}
			maxAttempts := 30
			for i := 0; i < maxAttempts; i++ {
				v, err := cloud.GetVolume(ctx, volumeIDInt)
				if err != nil || v == nil {
					time.Sleep(2 * time.Second)
					continue
				}
				if *v.Status == "available" {
					klog.Infof("ControllerPublishVolume: Volume detached from wrong node, now available")
					getVolume = v
					break
				}
				time.Sleep(2 * time.Second)
			}
			if *getVolume.Status != "available" {
				return nil, status.Error(codes.DeadlineExceeded, "ControllerPublishVolume: Timeout waiting for detachment from wrong node")
			}
		}
	}
	if *getVolume.Status == "available" {
		attachVolume, err := cloud.AttachVolumeToNode(ctx, vmId, volumeIDInt)
		if err != nil {
			klog.Errorf("ControllerPublishVolume: Failed to AttachVolumeToNode: %v", err)
			return nil, status.Errorf(codes.Internal, "ControllerPublishVolume: Failed to AttachVolumeToNode: %v", err)
		}
		klog.Infof("ControllerPublishVolume: AttachVolumeToNode succeeded -\nID: %v\nInstance id ID: %v\nStatus: %v\nVolume ID: %v", *attachVolume.Id, *attachVolume.InstanceId, *attachVolume.Status, *attachVolume.VolumeId)

		maxAttempts := 30
		klog.Infof("ControllerPublishVolume: Polling for attachment to complete with max attempts: %d", maxAttempts)
		for i := 0; i < maxAttempts; i++ {
			klog.Infof("ControllerPublishVolume: Polling for attachment %d/%d", i+1, maxAttempts)
			v, err := cloud.GetVolume(ctx, volumeIDInt)
			if err != nil {
				klog.Errorf("ControllerPublishVolume: Failed to GetVolume while polling: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
			if v == nil {
				klog.Warningf("ControllerPublishVolume: GetVolume attempt %d returned nil volume", i+1)
				time.Sleep(2 * time.Second)
				continue
			}
			if *v.Status == "in-use" && len(*v.Attachments) > 0 {
				attachment := (*v.Attachments)[0]
				if err == nil && *attachVolume.Id == vmId && attachment.Device != nil {
					klog.Infof("ControllerPublishVolume: Volume attached to correct node with device %s", *attachment.Device)
					return &csi.ControllerPublishVolumeResponse{
						PublishContext: map[string]string{
							volNameKeyFromControllerPublishVolume: *attachment.Device,
						},
					}, nil
				}
			}
			time.Sleep(2 * time.Second)
		}
		return nil, status.Error(codes.DeadlineExceeded, "ControllerPublishVolume: Timeout waiting for attachment to complete")
	}

	klog.Errorf("ControllerPublishVolume: Volume %s in unexpected state: %s", *getVolume.Name, *getVolume.Status)
	return nil, status.Errorf(codes.FailedPrecondition,
		"Volume in unexpected state: %s (expected 'available' or 'in-use')", *getVolume.Status)
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	klog.Infof("ControllerUnpublishVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	virtualMachineId := req.NodeId
	vmId, err := strconv.Atoi(virtualMachineId)
	if err != nil {
		klog.Errorf("ControllerPublishVolume: Failed to convert virtual machine ID to int: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to convert virtual machine ID to int: %v", err)
	}
	volumeID := req.GetVolumeId()
	cloud := cs.driver.hyperstackClient
	volumeIDInt, err := strconv.Atoi(volumeID)
	if err != nil {
		klog.Errorf("ControllerUnpublishVolume: Failed to convert volume ID to int: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to convert volume ID to int: %v", err)
	}
	getVolume, err := cloud.GetVolume(ctx, volumeIDInt)
	klog.Infof("ControllerUnpublishVolume: GetVolume returned volume: %+v", getVolume)
	if err != nil {
		klog.Errorf("ControllerUnpublishVolume: Failed to GetVolume from hyperstack: %v", err)
		return nil, status.Errorf(codes.NotFound, "ControllerUnpublishVolume: Failed to GetVolume from hyperstack: %v", err)
	}
	if getVolume == nil {
		klog.Errorf("ControllerUnpublishVolume: GetVolume returned nil volume")
		return nil, status.Errorf(codes.NotFound, "ControllerUnpublishVolume: GetVolume returned nil volume")
	}
	klog.Infof("ControllerUnpublishVolume: GetVolume succeeded -\nStatus: %s\nName: %s\nID: %d\nSize:%d", *getVolume.Status, *getVolume.Name, *getVolume.Id, *getVolume.Size)
	if *getVolume.Status == "in-use" {
		detachVolume, err := cloud.DetachVolumeFromNode(ctx, vmId, volumeIDInt)
		if err != nil {
			klog.Errorf("ControllerUnpublishVolume: Failed to DetachVolumeFromNode: %v", err)
			return nil, status.Errorf(codes.Internal, "ControllerUnpublishVolume: Failed to DetachVolumeFromNode: %v", err)
		}
		klog.Infof("ControllerUnpublishVolume: DetachVolumeFromNode succeeded -\nMessage: %v\nStatus: %v\nVolume Attachments: %v", *detachVolume.Message, *detachVolume.Status, *detachVolume.VolumeAttachments)

		maxAttempts := 30
		klog.Infof("ControllerUnpublishVolume: Polling for volume to become available with max attempts: %d", maxAttempts)
		for i := 0; i < maxAttempts; i++ {
			klog.Infof("ControllerUnpublishVolume: Polling for volume to become available %d/%d", i+1, maxAttempts)
			v, err := cloud.GetVolume(ctx, volumeIDInt)
			if err != nil {
				klog.Errorf("ControllerUnpublishVolume: Failed to GetVolume while polling: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
			if v == nil {
				klog.Warningf("ControllerUnpublishVolume: GetVolume attempt %d returned nil volume", i+1)
				time.Sleep(2 * time.Second)
				continue
			}
			if *v.Status == "available" {
				klog.Infof("ControllerUnpublishVolume: Volume is now available after detachment")
				return &csi.ControllerUnpublishVolumeResponse{}, nil
			}
			time.Sleep(2 * time.Second)
		}
		return nil, status.Error(codes.DeadlineExceeded, "ControllerUnpublishVolume: Timeout waiting for volume to detach")
	}
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (cs *controllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	klog.Infof("ListVolumes: called with %+#v request", req)
	klog.Warning("ListVolumes: Not implemented yet!")
	return &csi.ListVolumesResponse{}, nil
}

func (cs *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	klog.Infof("CreateSnapshot: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("CreateSnapshot: Not implemented yet!")
	return &csi.CreateSnapshotResponse{}, nil
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	klog.Infof("DeleteSnapshot: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("DeleteSnapshot: Not implemented yet!")
	return &csi.DeleteSnapshotResponse{}, nil
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	// TODO(joseb)
	klog.Warning("ListSnapshots: Not implemented yet!")
	return &csi.ListSnapshotsResponse{}, nil
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities
func (cs *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
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

	volumeIDInt, err := strconv.Atoi(volumeID)
	if err != nil {
		klog.Errorf("ValidateVolumeCapabilities: Failed to convert volume ID to int: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to convert volume ID to int: %v", err)
	}
	_, err = cs.driver.hyperstackClient.GetVolume(ctx, volumeIDInt)
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
	klog.Infof("ControllerGetVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	return &csi.ControllerGetVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	klog.Infof("ControllerExpandVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("ControllerExpandVolume: Not implemented yet!")
	return &csi.ControllerExpandVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerModifyVolume(ctx context.Context, req *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	klog.Infof("ControllerModifyVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	klog.Warning("ControllerModifyVolume: Not implemented yet!")
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
