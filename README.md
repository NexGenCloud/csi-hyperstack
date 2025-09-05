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

## Usage

You can install the Hyperstack CSI Driver using the provided Helm chart. Before that clone the repository.


```bash
helm upgrade --install csi-hyperstack --values ./deploy/helm/values.yaml -n csi-driver-test --create-namespace ./deploy/helm
```
After installation, verify that all pods in the `csi-driver` namespace are running successfully. For example usage, please refer to [example/manifest](./example/manifest.yaml)


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