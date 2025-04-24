# Hyperstack CSI driver

> This is an official kubernetes CSI Driver for Nexgen Hyperstack platform. It
> uses the Hyperstack Go SDK to interact with the Hyperstack API and manage
> resources. The driver is written in Go and is inspired by other popular cloud
> adapters.

## 🚀 Getting Started

> This repository contains [Taskfile](./Taskfile.yaml) with various
> helper commands to manage current project

Before you start, make sure you have the following tools installed:

- [Go 1.22](https://golang.org/dl/)
- [Task 3.25](https://taskfile.dev/installation/): A task runner for executing
  project tasks.
- [Docker](https://docs.docker.com/get-docker/): container builder and runtime
- [docker-compose](https://docs.docker.com/compose/install/): local docker
  automation

There are also CLI dependencies that are installed with Go:

- [GoReleaser](https://goreleaser.com/): manages Go builds
  ````bash
  go install github.com/goreleaser/goreleaser@latest
- optional / [gRPCurl](https://github.com/fullstorydev/grpcurl): helps debugging
  gRPC calls
  ````bash
  go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

## 🏗️ Usage

Hyperstack CSI driver is deployed as a Helm chart. To provision it to the
cluster:

- Provision a k8s cluster on Hyperstack
- Ensure that `KUBECONFIG` env variable is set to kubeconfig file path
- Create a new namespace:
  ````bash
  NS="csi-driver"
  kubectl create namespace "${NS}"
  ````
- Create `values.yaml` (see `deploy/helm/values.yaml` for details):
  ````yaml
  components:
    csiHyperstack:
      # Set tag to latest if you are testing locally pushed build
      image: reg.digitalocean.ngbackend.cloud/nexgenk8s/csi-hyperstack:main-latest
      hyperstackClusterId: "Your cluster ID"
  hyperstack:
    apiKey: "Your key"
    apiAddress: "Here you can change address if needed"
  ````
    - Set API address
    - Set API key
    - Set ClusterID to correct cluster ID (returned by Hyperstack API)
- Run install:
  ````bash
  helm upgrade --install -f values.yaml -n "${NS}" csi-hyperstack ./deploy/helm
  kubectl -n "${NS}" get po -w
  ````

## 💻 Development

To build the driver, run the following command:

````bash
task build
````

If you want to push images for testing:

- Login to Harbor
  ````bash
  # User Profile -> Username
  USER=""
  # User Profile -> CLI Secret
  PASSWORD=""
  docker login -u "${USER}" -p "${PASSWORD}" https://reg.digitalocean.ngbackend.cloud
  ````
- Build and push image:
  ````bash
  task docker-build
  task docker-push
  ````

This will compile the CSI driver and output the binary in the `dist`
directory.

## 🔎 Testing

This project is intended to be used in a k8s cluster running on Hyperstack. To
run it locally (for testing) you can run it directly:

````bash
export HYPERSTACK_API_KEY=""
export HYPERSTACK_API_ADDRESS=""

# Start the csi-hyperstack binary
./dist/csi-hyperstack start \
  --endpoint "unix://tmp/csi-hyperstack.sock" \
  --hyperstack-cluster-id "" \
  --hyperstack-node-id "" \
  --service-controller-enabled \
  --service-node-enabled
````

To get more information on available commands and options run help:

````bash
./dist/csi-hyperstack --help
````

Better approach is using docker-compose.

- Firstly run docker build:
  ```bash
  task docker-build
  ```
- Create a local `docker-compose.local.yml` with required env variables:
  ```yaml
  services:
    csi-hyperstack:
      environment:
        HYPERSTACK_API_KEY: "Your key"
        HYPERSTACK_API_ADDRESS: "Here you can change address if needed"
  ```
- Run it:
  ````bash
  task start
  ````
- Check [metrics](http://localhost:8080/metrics)
- Check that `./data/csi/csi.sock` UNIX socket file is created

### Testing gRPC calls

You can use `gRPCurl` to send requests and read responses:

- For local run with UNIX socket:
  ````bash
  alias gsi='grpcurl -plaintext -unix /tmp/csi-hyperstack.sock'
  gsi list
  ````
- When running in docker:
  ````bash
  alias gsi='grpcurl -plaintext localhost:8081'
  gsi list
  ````

Example output for list:

````
csi.v1.Controller
csi.v1.Identity
csi.v1.Node
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
````

Some helpful endpoints (
see [specification](https://github.com/container-storage-interface/spec/blob/master/spec.md)
for more info):

- Driver name and version:
  ````bash
  gsi csi.v1.Identity/GetPluginInfo
  ````
- Plugin capabilities:
  ````bash
  gsi csi.v1.Identity/GetPluginCapabilities
  ````
- Controller capabilities:
  ````bash
  gsi csi.v1.Controller/ControllerGetCapabilities
  ````
- Node capabilities:
  ````bash
  gsi csi.v1.Node/NodeGetCapabilities
  ````

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

## Contributing

Contributions to this project are welcome. Please make sure to test your changes
before submitting a pull request.
