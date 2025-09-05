# **CSI Hyperstack Helm Chart**  
[![Helm](https://img.shields.io/badge/helm-chart-blue)](https://helm.sh/)  
[![Kubernetes](https://img.shields.io/badge/kubernetes-1.24+-green)](https://kubernetes.io/)

**CSI Hyperstack Helm Chart** provides an easy way to deploy the **Container Storage Interface (CSI) driver** for **Hyperstack** on Kubernetes.  

This Helm chart simplifies installing, upgrading, and managing the CSI driver required for dynamically provisioning storage volumes on **Hyperstack**.

---

## **Overview**
The **CSI Hyperstack Helm Chart** enables seamless integration between your **Kubernetes cluster** and **Hyperstack**.  
It deploys all required CSI components, including:
- Controller pods
- Node plugins
- RBAC roles and bindings
- StorageClasses for dynamic volume provisioning

---

## **Features**
✅ Deploys the **Hyperstack CSI driver**  
✅ Supports **dynamic volume provisioning**  
✅ Works with **Helm 3+**  
✅ Kubernetes **1.24+**  
✅ Easily configurable via `values.yaml`

---

## **Prerequisites**
Before installing, ensure you have:
- A running **Kubernetes cluster** (v1.24 or later)
- [Helm 3+](https://helm.sh/docs/intro/install/)
- Access to **Hyperstack API key**
- Proper `kubeconfig` context set for your target cluster

---
## **Installation**

### **1. Add the Helm Repository**
```bash
helm repo add nexgencloud https://nexgencloud.github.io/csi-hyperstack
helm repo update
```

### **2. Install the Chart**
```bash
helm install csi-hyperstack nexgencloud/csi-hyperstack --namespace kube-system --create-namespace --set hyperstack.apiKey=<YOUR_HS_API_KEY>
```

### **3. Verify Installation**
```bash
kubectl get pods -n kube-system
```

You should see all CSI controller and node pods in **Running** state.

---

## **Upgrading**
To upgrade to the latest chart version:

```bash
helm upgrade csi-hyperstack nexgencloud/csi-hyperstack --namespace kube-system
```

---

## **Uninstalling**
To remove the chart:

```bash
helm uninstall csi-hyperstack -n kube-system
```

---

## **Configuration**
This chart supports custom configuration via a `values.yaml` file.  
Here are the most commonly used options:

Mandetory value during chart installations.

| Key                 | Type   | Description                                        | Example                            |
| ------------------- | ------ | -------------------------------------------------- | ---------------------------------- |
| `hyperstack.apiKey` | string | API key for authenticating with the Hyperstack API | `hyperstack.apiKey=abcd1234` |

Other commonly used optional values

| Key                              | Type   | Description                                                 | Default                           |
| -------------------------------- | ------ | ----------------------------------------------------------- | ------------------------------------------- |
| `components.csiHyperstack.tag`          | string | CSI Hyperstack Image tag                              | `latest` |
| `hyperstack.apiAddress`          | string | Base URL of the Hyperstack API                              | `https://infrahub-api.nexgencloud.com/v1` |
| `storageClass.enabled`           | bool   | Whether to create a default `StorageClass`                  | `true`                                      |
| `storageClass.name`              | string | Name of the `StorageClass`                                  | `csi-hyperstack`                          |
| `storageClass.volumeBindingMode` | string | Volume binding mode (`Immediate` or `WaitForFirstConsumer`) | `Immediate`                               |
| `storageClass.reclaimPolicy`     | string | Reclaim policy (`Delete` or `Retain`)                       | `Delete`                                  |


---

## **Development**
If you want to make changes to the chart:

```bash
# Lint the chart
helm lint .
# Test install locally
helm install csi-hyperstack --set hyperstack.apiKey=<YOUR_HS_API_key> . --dry-run --debug
```