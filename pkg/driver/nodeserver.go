package driver

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"golang.org/x/net/context"
	"k8s.io/klog/v2"

	// cpoerrors "k8s.io/cloud-provider-openstack/pkg/util/errors"
	kubernetes "k8s.io/csi-hyperstack/pkg/utils/kubernetes"
	"k8s.io/csi-hyperstack/pkg/utils/metadata"
	"k8s.io/csi-hyperstack/pkg/utils/mount"
)

const (
	hyperstackInstanceIdLabelKey = "hyperstack.cloud/instance-id"
)

type nodeServer struct {
	driver   *Driver
	mount    mount.IMount
	metadata metadata.IMetadata
	csi.UnimplementedNodeServer
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	klog.Infof("\n==============NodeStageVolume: called================\n")
	klog.Infof("NodeStageVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	devicename := req.PublishContext[volNameKeyFromControllerPublishVolume]
	if devicename == "" {
		return nil, status.Error(codes.InvalidArgument, "Device name not found in publish context. Please wait for volume to be attached.")
	}
	klog.Infof("NodeStageVolume: devicename from publish context: %s", devicename)
	err := formateAndMakeFS(devicename, "ext4")
	if err != nil {
		return nil, err
	}

	target := req.StagingTargetPath
	err = mountDevice(devicename, target, "ext4", []string{})
	if err != nil {
		return nil, err
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

func formateAndMakeFS(device string, fstype string) error {
	klog.Infof("formateAndMakeFS: called with args %s, %s", device, fstype)

	// Wait for device to be ready and check if filesystem exists
	// This prevents race condition where device is attached but not yet accessible
	const maxRetries = 10
	const retryDelay = 500 * time.Millisecond

	var existingFS string
	var deviceReady bool

	for i := 0; i < maxRetries; i++ {
		// Check if device file exists
		if _, err := os.Stat(device); err != nil {
			if i < maxRetries-1 {
				klog.Infof("Device %s not ready (attempt %d/%d), waiting...", device, i+1, maxRetries)
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("device %s not found after %d attempts: %v", device, maxRetries, err)
		}

		// Try to read filesystem type with blkid
		blkidCmd := exec.Command("blkid", "-o", "value", "-s", "TYPE", device)
		output, err := blkidCmd.CombinedOutput()
		existingFS = strings.TrimSpace(string(output))

		if err == nil {
			// blkid succeeded - filesystem exists
			deviceReady = true
			break
		}

		// Check if error indicates device is not ready (vs no filesystem)
		exitErr, ok := err.(*exec.ExitError)
		if ok && exitErr.ExitCode() == 2 {
			// Exit code 2 means no filesystem found - device is ready but empty
			deviceReady = true
			break
		}

		// Other errors might indicate device not ready
		if i < maxRetries-1 {
			klog.Infof("Device %s not ready for blkid (attempt %d/%d): %v, waiting...", device, i+1, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}

		return fmt.Errorf("device %s not ready after %d attempts: %v", device, maxRetries, err)
	}

	if !deviceReady {
		return fmt.Errorf("device %s did not become ready", device)
	}

	if existingFS != "" {
		klog.Infof("Filesystem %s already exists on %s, skipping format", existingFS, device)
		return nil // Don't format if filesystem exists
	}

	klog.Infof("No filesystem detected on %s, creating %s filesystem", device, fstype)

	// Only format if no filesystem exists
	mkfsCmd := fmt.Sprintf("mkfs.%s", fstype)

	_, err := exec.LookPath(mkfsCmd)
	if err != nil {
		return fmt.Errorf("unable to find the mkfs (%s) utiltiy errors is %s", mkfsCmd, err.Error())
	}

	// actually run mkfs.ext4 -F source
	mkfsArgs := []string{"-F", device}

	out, err := exec.Command(mkfsCmd, mkfsArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("create fs command failed output: %s, and err: %s", out, err.Error())
	}
	klog.Infof("formateAndMakeFS: command output: %s", out)
	return nil
}

func mountDevice(source string, target string, fsType string, options []string) error {
	klog.Infof("mountDevice: called with args %s, %s, %s, %v", source, target, fsType, options)
	mountCmd := "mount"

	if fsType == "" {
		return fmt.Errorf("fstype is not provided")
	}

	mountArgs := []string{}
	err := os.MkdirAll(target, 0777)
	if err != nil {
		return fmt.Errorf("error: %s, creating the target dir", err.Error())
	}
	mountArgs = append(mountArgs, "-t", fsType)

	// check of options and then append them at the end of the mount command
	if len(options) > 0 {
		mountArgs = append(mountArgs, "-o", strings.Join(options, ","))
	}

	mountArgs = append(mountArgs, source)
	mountArgs = append(mountArgs, target)

	out, err := exec.Command(mountCmd, mountArgs...).CombinedOutput()
	klog.Infof("mountDevice: command output: %s", out)
	if err != nil {
		return fmt.Errorf("error %s, mounting the source %s to tar %s. Output: %s", err.Error(), source, target, out)
	}
	return nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	klog.Infof("==============NodeUnstageVolume: called================\n")
	klog.Infof("NodeUnstageVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume Id not provided")
	}

	stagingTargetPath := req.GetStagingTargetPath()
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Staging Target Path must be provided")
	}

	err := ns.mount.UnmountPath(stagingTargetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unmount of targetPath %s failed with error %v", stagingTargetPath, err)
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.Infof("==============NodePublishVolume: called================\n")
	klog.Infof("NodePublishVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	options := []string{"bind"}
	if req.Readonly {
		options = append(options, "ro")
	}

	fsType := "ext4"
	if req.VolumeCapability.GetMount().FsType != "" {
		fsType = req.VolumeCapability.GetMount().FsType
	}

	source := req.StagingTargetPath
	target := req.TargetPath

	err := mountDevice(source, target, fsType, options)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error %s, mounting the volume from staging dir to target dir", err.Error()))
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.Infof("NodeUnPublishVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "[NodeUnpublishVolume] Target Path must be provided")
	}
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "[NodeUnpublishVolume] volumeID must be provided")
	}

	if err := ns.mount.UnmountPath(targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "Unmount of targetpath %s failed with error %v", targetPath, err)
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil

}

func (ns *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	nodeID, err := kubernetes.GetNodeLabel(hyperstackInstanceIdLabelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get node UUID: %v", err)
	}
	klog.Infof("NodeGetInfo called with nodeID: %#v\n", nodeID)
	return &csi.NodeGetInfoResponse{
		NodeId:            nodeID,
		MaxVolumesPerNode: 5,
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				"hyperstack.cloud/instance-id": nodeID,
			},
		},
	}, nil
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.Infof("==============NodeGetCapabilities: called================\n")
	klog.Infof("NodeGetCapabilities: called with args %+v", protosanitizer.StripSecrets(*req))
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.driver.nscap,
	}, nil
}

func (ns *nodeServer) NodeGetVolumeStats(_ context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	klog.Infof("NodeGetVolumeStats: called with args %+v", protosanitizer.StripSecrets(*req))
	return &csi.NodeGetVolumeStatsResponse{}, nil
}

func (ns *nodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	klog.Infof("NodeExpandVolume: called with args %+v", protosanitizer.StripSecrets(*req))
	return &csi.NodeExpandVolumeResponse{}, nil
}
