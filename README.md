# Hyperstack CSI Driver
Current released image - `ghcr.io/nexgencloud/csi-hyperstack/csi:v0.0.13`

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

To build the image locally:

```bash
make docker-build VERSION=<VERSION>          # build only
make docker-build-push VERSION=<VERSION>     # build and push to GHCR
```

## Usage

Refer to the [charts/csi-hyperstack](./charts/csi-hyperstack/README.md) documentation for details on installation and usage with Helm.

## Releasing

Pushing a `v*.*.*` tag is the whole release. It runs `.github/workflows/release.yml`,
which builds the image and then publishes the chart.

```bash
git tag v0.0.12 origin/main
git push origin v0.0.12
```

The tag is the single source of truth for both versions:

| | value for tag `v0.0.12` |
|---|---|
| image | `ghcr.io/nexgencloud/csi-hyperstack/csi:v0.0.12` |
| chart `version` | `v0.0.12` |
| chart `appVersion` | `v0.0.12` |

`Chart.yaml` is rewritten inside the workflow before packaging, so its committed
values are ignored — do not bump them by hand.

The chart job runs only after the image push succeeds, so a published chart can
never point at an image that does not exist.

Charts are published as GitHub Releases tagged `csi-hyperstack-<version>`, with
the index committed to the `gh-pages` branch and served at
<https://nexgencloud.github.io/csi-hyperstack>.

Versions are compared with the leading `v` ignored, so each tag must sort above
the last released one — `v0.0.7` would rank below the existing `0.0.10` and
never be served as latest.


## Documentation
For more information about the features of the Hyperstack API, visit
the [Hyperstack Documentation](https://infrahub-doc.nexgencloud.com/docs/features/).

Relevant docs:
- [Kubernetes CSI Developer Documentation](https://kubernetes-csi.github.io/docs/introduction.html):
- [CSI Specification](https://github.com/container-storage-interface/spec/blob/master/spec.md)
- [Openstack cinder driver](https://github.com/kubernetes/cloud-provider-openstack/blob/master/pkg/csi/cinder/driver.go)
