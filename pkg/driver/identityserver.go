package driver

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/ptypes/wrappers"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

type identityServer struct {
	driver *Driver
	csi.UnimplementedIdentityServer
}

func (ids *identityServer) GetPluginInfo(
	ctx context.Context,
	req *csi.GetPluginInfoRequest,
) (*csi.GetPluginInfoResponse, error) {
	if ids.driver.name == "" {
		return nil, status.Error(codes.Unavailable, "Driver name not configured")
	}

	if ids.driver.version == "" {
		return nil, status.Error(codes.Unavailable, "Driver is missing version")
	}

	return &csi.GetPluginInfoResponse{
		Name:          ids.driver.name,
		VendorVersion: ids.driver.version,
	}, nil
}

func (ids *identityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	klog.Infof("probe called")
	ids.driver.readyMu.Lock()
	defer ids.driver.readyMu.Unlock()
	return &csi.ProbeResponse{
		Ready: &wrappers.BoolValue{
			Value: ids.driver.ready,
		},
	}, nil
}

func (ids *identityServer) GetPluginCapabilities(
	ctx context.Context,
	req *csi.GetPluginCapabilitiesRequest,
) (*csi.GetPluginCapabilitiesResponse, error) {
	klog.V(5).Infof("GetPluginCapabilities called with req %+v", req)
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
			// VOLUME_ACCESSIBILITY_CONSTRAINTS intentionally not declared: volumes aren't
			// zone/node scoped in Hyperstack (CreateVolume places everything in one cluster-wide
			// environment, and ControllerPublishVolume can attach any volume to any node,
			// including auto-detaching from a previous node first). Declaring this capability
			// caused external-provisioner to collect NodeGetInfo's per-node instance-id topology
			// segment from every node and bake it into each PV's nodeAffinity as a hard
			// requirement, permanently pinning volumes to whichever nodes existed at creation
			// time — breaking replacement/migration of any node still holding a PVC.
			{
				Type: &csi.PluginCapability_VolumeExpansion_{
					VolumeExpansion: &csi.PluginCapability_VolumeExpansion{
						Type: csi.PluginCapability_VolumeExpansion_ONLINE,
					},
				},
			},
			{
				Type: &csi.PluginCapability_VolumeExpansion_{
					VolumeExpansion: &csi.PluginCapability_VolumeExpansion{
						Type: csi.PluginCapability_VolumeExpansion_OFFLINE,
					},
				},
			},
		},
	}, nil
}
