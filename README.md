# Hyperstack CSI Driver
Current released image - `reg.digitalocean.ngbackend.cloud/hyperstack-csi-driver/csi:v0.0.1`

## Introduction
This documentation provides instructions for installing and using the Hyperstack CSI Driver. The CSI provisioner for hyerstack CSI driver is `hyperstack.csi.nexgencloud.com`.
Before you begin, ensure you have the following tools installed:

* Go 1.24
* Helm
* A Hyperstack Kubernetes cluster

### CLI Dependencies (installed via Go)
* (Optional) [gRPCurl](https://github.com/fullstorydev/grpcurl) – Useful for debugging gRPC calls.

  ```bash
  go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
  ```

## Development

Run `go mod tidy` before starting the gRPC server:

```bash
go run main.go start \
  --endpoint="unix://tmp/csi-hyperstack.sock" \
  --hyperstack-api-address="<API_BASE_URL>" \
  --hyperstack-api-key="<API_KEY>" \
  --service-controller-enabled \
  --service-node-enabled
```

To list the available gRPC calls:

```bash
grpcurl --plaintext unix:///tmp/csi-hyperstack.sock list
```

You can invoke a specific RPC method for a given operation using `grpcurl`.

To build the project:

```bash
make build VERSION=<VERSION>
```

## Usage

Refer to the [charts/csi-hyperstack](./charts/csi-hyperstack/README.md) documentation for details on installation and usage with Helm.


## Documentation
For more information about the features of the Hyperstack API, visit
the [Hyperstack Documentation](https://infrahub-doc.nexgencloud.com/docs/features/).

Relevant docs:
- [Kubernetes CSI Developer Documentation](https://kubernetes-csi.github.io/docs/introduction.html):
- [CSI Specification](https://github.com/container-storage-interface/spec/blob/master/spec.md)
- [Openstack cinder driver](https://github.com/kubernetes/cloud-provider-openstack/blob/master/pkg/csi/cinder/driver.go)