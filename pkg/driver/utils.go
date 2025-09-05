package driver

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

var (
	serverGRPCEndpointCallCounter uint64
)

func MapControllerServiceCapabilities(
	input []csi.ControllerServiceCapability_RPC_Type,
) []*csi.ControllerServiceCapability {
	items := make([]*csi.ControllerServiceCapability, 0, len(input))

	for _, c := range input {
		klog.Infof("Enabling controller service capability: %v", c.String())
		items = append(items, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: c,
				},
			},
		})
	}
	return items
}

func MapVolumeCapabilityAccessModes(
	input []csi.VolumeCapability_AccessMode_Mode,
) []*csi.VolumeCapability_AccessMode {
	items := make([]*csi.VolumeCapability_AccessMode, 0, len(input))

	for _, c := range input {
		klog.Infof("Enabling volume access mode: %v", c.String())
		items = append(items, &csi.VolumeCapability_AccessMode{Mode: c})
	}
	return items
}

func MapNodeServiceCapabilities(
	input []csi.NodeServiceCapability_RPC_Type,
) []*csi.NodeServiceCapability {
	items := make([]*csi.NodeServiceCapability, 0, len(input))

	for _, c := range input {
		klog.Infof("Enabling node service capability: %v", c.String())
		items = append(items, &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: c,
				},
			},
		})
	}

	return items
}

func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	callID := atomic.AddUint64(&serverGRPCEndpointCallCounter, 1)

	klog.Infof("[ID:%d] GRPC call: %s", callID, info.FullMethod)
	klog.Infof("[ID:%d] GRPC request: %s", callID, protosanitizer.StripSecrets(req))
	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf("[ID:%d] GRPC error: %v", callID, err)
	} else {
		klog.Infof("[ID:%d] GRPC response: %s", callID, protosanitizer.StripSecrets(resp))
	}

	return resp, err
}
