# **CSI Hyperstack Helm Chart**  
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)  
[![Helm](https://img.shields.io/badge/helm-chart-blue)](https://helm.sh/)  
[![Kubernetes](https://img.shields.io/badge/kubernetes-1.24+-green)](https://kubernetes.io/)

**CSI Hyperstack Helm Chart** provides an easy way to deploy the **Container Storage Interface (CSI) driver** for **Hyperstack** on Kubernetes.  

This Helm chart simplifies installing, upgrading, and managing the CSI driver required for dynamically provisioning storage volumes on **Hyperstack**.

---

## **Table of Contents**
- [Overview](#overview)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Upgrading](#upgrading)
- [Uninstalling](#uninstalling)
- [Configuration](#configuration)
- [Development](#development)
- [License](#license)

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
✅ Kubernetes **1.24+** tested  
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
helm repo add nexgencloud https://nexgencloud.github.io/csi-hyperstack-helm-chart
helm repo update
```

### **2. Install the Chart**
```bash
helm install csi-hyperstack nexgencloud/csi-hyperstack-helm-chart   --namespace kube-system   --create-namespace
```

### **3. Verify Installation**
```bash
kubectl get pods -n kube-system -l app.kubernetes.io/name=csi-hyperstack
```

You should see all CSI controller and node pods in **Running** state.

---

## **Upgrading**
To upgrade to the latest chart version:

```bash
helm upgrade csi-hyperstack nexgencloud/csi-hyperstack-helm-chart   --namespace kube-system
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

| **Parameter**                 | **Description**                    | **Default**          |
|------------------------------|------------------------------------|----------------------|
| `controller.replicas`       | Number of controller replicas      | `2`                  |
| `node.tolerations`          | Tolerations for node pods          | `[]`                 |
| `storageClass.name`         | Default StorageClass name          | `hyperstack-csi`     |
| `storageClass.reclaimPolicy`| PVC reclaim policy                 | `Delete`             |
| `hyperstack.apiKey`         | API key for Hyperstack            | `""`                 |
| `hyperstack.endpoint`      | Hyperstack API endpoint           | `""`                 |

To override defaults:

```bash
helm install csi-hyperstack nexgencloud/csi-hyperstack-helm-chart   --namespace kube-system   --set hyperstack.apiKey=YOUR_API_KEY   --set hyperstack.endpoint=https://infrahub-api.nexgencloud.com
```

---

## **Development**
If you want to make changes to the chart:

```bash
# Clone the repository
git clone https://github.com/NexGenCloud/csi-hyperstack-helm-chart.git
cd csi-hyperstack-helm-chart

# Lint the chart
helm lint .

# Test install locally
helm install csi-hyperstack ./ --dry-run --debug
```

---

## **License**
This project is licensed under the **Apache 2.0 License**.  
See [LICENSE](LICENSE) for details.

---

## **Next Steps**
- [Helm Documentation](https://helm.sh/docs/)
- [Kubernetes CSI Docs](https://kubernetes-csi.github.io/docs/)
- [NexGen Cloud Hyperstack](https://nexgencloud.com)
