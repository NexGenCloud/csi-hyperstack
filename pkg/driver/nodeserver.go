package driver

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	// check is symlink exists, if not create <hs_name>_<original_name> -> <original_name>
	// hyperstackVolumeName := "pvc-00000000-0000-0000-0000-000000000000"
	hyperstackVolumeName := req.PublishContext[volNameKeyFromControllerPublishVolume]
	klog.Infof("NodeStageVolume: hyperstackVolumeName from publish context: %s", hyperstackVolumeName)
	err := createSymLink(hyperstackVolumeName)
	if err != nil {
		return nil, err
	}
	devicePathToMount, err := getDevicePathToMount(hyperstackVolumeName)
	if err != nil {
		return nil, err
	}
	err = formateAndMakeFS(devicePathToMount, "ext4")
	if err != nil {
		return nil, err
	}

	target := req.StagingTargetPath
	err = mountDevice(devicePathToMount, target, "ext4", []string{})
	if err != nil {
		return nil, err
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

func createSymLink(hyperstackVolumeName string) error {
	klog.Infof("==============createSymLink: Creating symlink for %s================\n", hyperstackVolumeName)
	diskPath := "/dev/disk"
	err := os.MkdirAll(fmt.Sprintf("%s/%s", diskPath, "by-volumeid"), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	byVolumeIdPath := fmt.Sprintf("%s/%s", diskPath, "by-volumeid")
	byVolumeIdSymlinks, err := os.ReadDir(byVolumeIdPath)

	klog.Infof("createSymLink: Files in the by-volumeid directory: %s, %v", byVolumeIdPath, byVolumeIdSymlinks)
	if err != nil {
		return fmt.Errorf("createSymLink: failed to read by-volumeid directory: %v", err)
	}
	byVolumeIdSymlinkString := "_"
	for _, file := range byVolumeIdSymlinks {
		byVolumeIdSymlinkString += file.Name() + "_"
	}
	klog.Infof("createSymLink: byVolumeIdSymlinkString: %s", byVolumeIdSymlinkString)
	byIdPath := fmt.Sprintf("%s/%s", diskPath, "by-id")
	byIdSymlinks, err := os.ReadDir(byIdPath)
	if err != nil {
		return fmt.Errorf("createSymLink: failed to read by-id directory: %v", err)
	}
	klog.Infof("createSymLink: Files in the by-id directory: %s, %v", byIdPath, byIdSymlinks)
	for _, byIdSymlink := range byIdSymlinks {
		byIdSymlinkName := byIdSymlink.Name()
		readlink, err := os.Readlink(fmt.Sprintf("%s/%s", byIdPath, byIdSymlinkName))
		if err != nil {
			return fmt.Errorf("createSymLink: failed to read link: %v", err)
		}
		klog.Infof("createSymLink: byIdSymlink: %s, %s", byIdSymlinkName, readlink)
		klog.Infof("createSymLink: Check: %s, %s, %v", byIdSymlinkName, byVolumeIdSymlinkString, strings.Contains(byVolumeIdSymlinkString, byIdSymlinkName))
		if !strings.Contains(byVolumeIdSymlinkString, byIdSymlinkName) {
			byVolumeIdSymlinkPath := fmt.Sprintf("%s/%s/%s_%s", diskPath, "by-volumeid", byIdSymlinkName, hyperstackVolumeName)
			klog.Infof("createSymLink: Symlink not found in the by-volumeid directory for %s and %s\nCreating symlink between [%s -> %s]", byIdSymlinkName, readlink, byVolumeIdSymlinkPath, readlink)
			err = os.Symlink(readlink, byVolumeIdSymlinkPath)
			if err != nil {
				return fmt.Errorf("createSymLink: failed to create symlink: %v", err)
			}
		}
	}
	return nil
}

func formateAndMakeFS(device string, fstype string) error {
	klog.Infof("formateAndMakeFS: called with args %s, %s", device, fstype)
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

func getDevicePathToMount(hyperstackVolumeName string) (string, error) {
	diskPath := "/dev/disk/by-volumeid"
	byVolumeIdSymlinks, err := os.ReadDir(diskPath)
	if err != nil {
		return "", fmt.Errorf("error %s, reading the device path %s", err.Error(), diskPath)
	}
	for _, byVolumeIdSymlink := range byVolumeIdSymlinks {
		fmt.Println(byVolumeIdSymlink.Name())
		if strings.Contains(byVolumeIdSymlink.Name(), hyperstackVolumeName) {
			return fmt.Sprintf("%s/%s", diskPath, byVolumeIdSymlink.Name()), nil
		}
	}
	return "", fmt.Errorf("device path not found for %s", hyperstackVolumeName)
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

	// make sure the requried fields are set and not empty

	options := []string{"bind"}
	if req.Readonly {
		options = append(options, "ro")
	}

	// get req.VolumeCaps and make sure that you handle request for block mode as well
	// here we are just handling request for filesystem mode
	// in case of block mode, the source is going to be the device dir where volume was attached form ControllerPubVolume RPC

	fsType := "ext4"
	if req.VolumeCapability.GetMount().FsType != "" {
		fsType = req.VolumeCapability.GetMount().FsType
	}

	source := req.StagingTargetPath
	target := req.TargetPath

	// we want to run mount -t fstype source target -o bind,ro

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
