package kubernetes

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/csi-hyperstack/pkg/utils/metadata"
)

func GetNodeLabel(labelKey string) (string, error) {
	// kubeconfig := "/home/administrator/Desktop/nexgen/codebase/kubeconfig"
	// config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes client: %v", err)
	}
	instanceHostname, err := GetCurrentInstanHostname()
	if err != nil {
		return "", fmt.Errorf("failed to get node name: %v", err)
	}
	if instanceHostname == "" {
		return "", fmt.Errorf("node name not available")
	}

	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), instanceHostname, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get node %s: %v", instanceHostname, err)
	}
	if value, exists := node.Labels[labelKey]; exists {
		return value, nil
	}
	return "", fmt.Errorf("label %s not found on node %s", labelKey, instanceHostname)
}

func GetCurrentInstanHostname() (string, error) {
	searchOrder := "metadataService,configDrive"

	metadataProvider := metadata.GetMetadataProvider(searchOrder)
	instanceHostname, err := metadataProvider.GetInstanceHostname()
	if err != nil {
		return "", fmt.Errorf("failed to get node name: %v", err)
	}
	return instanceHostname, nil
}
