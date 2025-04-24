package driver

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

func RunGRPCServer(
	ctx context.Context,
	wg *sync.WaitGroup,
	endpoint string,
	ids csi.IdentityServer,
	cs csi.ControllerServer,
	ns csi.NodeServer,
) (*grpc.Server, error) {

	proto, addr, err := ParseEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to remove %s, error: %w", addr, err)
		}
	}

	listenConf := net.ListenConfig{}
	listener, err := listenConf.Listen(ctx, proto, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)

	// Register reflection service on gRPC server
	reflection.Register(server)

	if ids != nil {
		csi.RegisterIdentityServer(server, ids)
	}
	if cs != nil {
		csi.RegisterControllerServer(server, cs)
	}
	if ns != nil {
		csi.RegisterNodeServer(server, ns)
	}

	go func() {
		defer wg.Done()
		klog.Infof("gRPC listening on address: %s", listener.Addr().String())
		if err := server.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			klog.Errorf("Server stopped with: %v", err)
		}
	}()

	return server, nil
}
