package driver

import (
	"context"
	"fmt"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/csi-hyperstack/pkg/hyperstack"
	"k8s.io/csi-hyperstack/pkg/metrics"
	"k8s.io/csi-hyperstack/pkg/utils/metadata"
	"k8s.io/csi-hyperstack/pkg/utils/mount"
	"k8s.io/klog/v2"
)

var (
	DriverName    string
	DriverVersion string

	specVersion = "1.10.0"
)

type DriverOpts struct {
	Endpoint string
	// Environment          string
	// HyperstackClusterId  string
	// HyperstackNodeId     string
	HyperstackApiKey     string
	HyperstackApiAddress string
}

var (
	volNameKeyFromControllerPublishVolume = "hyperstack/volume-name"
)

type Driver struct {
	name    string
	version string

	opts *DriverOpts

	// serverMux *http.ServeMux

	hyperstackClient hyperstack.IHyperstack

	serviceIdentity   csi.IdentityServer
	serviceController csi.ControllerServer
	serviceNode       csi.NodeServer

	cscap []*csi.ControllerServiceCapability
	nscap []*csi.NodeServiceCapability
	vcap  []*csi.VolumeCapability_AccessMode
}

func NewDriver(opts *DriverOpts) *Driver {
	d := &Driver{}
	d.opts = opts
	fmt.Printf("Driver started with opts: %#v\n", d.opts)
	d.name = DriverName
	d.version = DriverVersion

	klog.Info("Driver: ", d.name)
	klog.Info("Driver version: ", d.version)
	klog.Info("CSI Spec version: ", specVersion)

	d.hyperstackClient = &hyperstack.Hyperstack{
		Client: hyperstack.NewHyperstackClient(
			opts.HyperstackApiKey,
			opts.HyperstackApiAddress,
		),
	}

	d.cscap = MapControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
		csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES_PUBLISHED_NODES,
		csi.ControllerServiceCapability_RPC_GET_VOLUME,
	})

	d.nscap = MapNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
		csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
	})

	d.vcap = MapVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	})

	return d
}

func (d *Driver) ValidateControllerServiceRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	if c == csi.ControllerServiceCapability_RPC_UNKNOWN {
		return nil
	}

	for _, cap := range d.cscap {
		if c == cap.GetRpc().GetType() {
			return nil
		}
	}

	return status.Error(codes.InvalidArgument, c.String())
}

func (d *Driver) SetupIdentityService() {
	klog.Info("Providing identity service")
	d.serviceIdentity = &identityServer{
		driver: d,
	}
}

func (d *Driver) SetupControllerService() {
	klog.Info("Providing controller service")
	d.serviceController = &controllerServer{
		driver: d,
	}
}

func (d *Driver) SetupNodeService() {
	klog.Info("Providing node service")
	d.serviceNode = &nodeServer{
		driver:   d,
		mount:    mount.GetMountProvider(),
		metadata: metadata.GetMetadataProvider(d.hyperstackClient.GetMetadataOpts().SearchOrder),
	}
}

func (d *Driver) Run(
	ctx context.Context,
	wg *sync.WaitGroup,
) (*grpc.Server, error) {
	if nil == d.serviceController && nil == d.serviceNode {
		return nil, fmt.Errorf("no CSI services initialized")
	}

	metrics.RegisterMetrics("hyperstack-csi")

	srv, err := RunGRPCServer(
		ctx,
		wg,
		d.opts.Endpoint,
		d.serviceIdentity,
		d.serviceController,
		d.serviceNode,
	)
	if err != nil {
		return nil, fmt.Errorf("failed running gRPC server: %w", err)
	}

	return srv, nil
}
