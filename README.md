# Hyperstack CSI Driver
Current released image - `reg.digitalocean.ngbackend.cloud/hyperstack-csi-driver/csi:v0.0.1`

## Introduction
This documentation provides instructions for installing and using the Hyperstack CSI Driver. The CSI provisioner for hyerstack CSI driver is `hyperstack.csi.nexgencloud.com`.
Before you begin, ensure you have the following tools installed:

* Go 1.24
* Helm
* A Hyperstack Kubernetes cluster

### CLI Dependencies (installed via Go)

* [GoReleaser](https://goreleaser.com/) – Manages Go builds.

  ```bash
  go install github.com/goreleaser/goreleaser@latest
  ```

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
task build
```

## Usage

Refer to the [charts/csi-hyperstack](./charts/csi-hyperstack/README.md) documentation for details on installation and usage with Helm.


## Documentation
For more information about the features of the Hyperstack API, visit
the [Hyperstack Documentation](https://infrahub-doc.nexgencloud.com/docs/features/).

Relevant docs:

- [Simple guide](https://arslan.io/2018/06/21/how-to-write-a-container-storage-interface-csi-plugin/)
- [Kubernetes CSI Developer Documentation](https://kubernetes-csi.github.io/docs/introduction.html):
  official documentation on CSI plugin development
- [CSI Volume Plugins in Kubernetes Design Doc](https://github.com/kubernetes/design-proposals-archive/blob/main/storage/container-storage-interface.md):
  initial CSI proposal

Examples (reference for development):

- [Openstack cinder driver](https://github.com/kubernetes/cloud-provider-openstack/blob/master/pkg/csi/cinder/driver.go)
- [Azure file](https://github.com/kubernetes-sigs/azurefile-csi-driver/blob/master/pkg/azurefile/azurefile.go)
- [Synology](https://github.com/SynologyOpenSource/synology-csi/blob/main/main.go)
- [Deployment example](https://github.com/kubernetes-csi/csi-driver-host-path/blob/master/deploy/kubernetes-1.27/hostpath/csi-hostpath-plugin.yaml)