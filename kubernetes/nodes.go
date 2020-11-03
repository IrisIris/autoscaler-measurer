package kubernetes

import (
	"context"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func IsNodeReady(k8sClient *kubernetes.Clientset, nodeName string) bool {
	if k8sClient == nil {
		log.Warn("k8s client is nil")
		return false
	}
	if nodeName == "" {
		log.Warn("node name is empty")
		return false
	}
	//res := map[string]coreV1.NodeCondition{}
	// access the API to list nodes
	node, err := k8sClient.CoreV1().Nodes().Get(context.TODO(), nodeName, v1.GetOptions{})
	if err != nil {
		log.Errorf("failed to get node %s, err: %v", nodeName, err)
	}
	for _, v := range node.Status.Conditions {
		if v.Type == "Ready" && v.Status == "True" {
			//res[node.ObjectMeta.Name] = v
			return true
		}
	}
	return false
}
