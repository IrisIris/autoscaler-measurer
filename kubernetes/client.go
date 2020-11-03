package kubernetes

import (
	"context"
	log "github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func GetK8sClient(clusterKubeConfigPath string) (*kubernetes.Clientset, error) {
	// uses the current context in kubeconfig
	// path-to-kubeconfig -- for example, /root/.kube/config
	config, _ := clientcmd.BuildConfigFromFlags("", clusterKubeConfigPath)
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("failed to get k8s client, err %v", err)
		return nil, err
	}
	return clientset, err
}

func getNodesFromApiServer() map[string]coreV1.NodeCondition {
	res := map[string]coreV1.NodeCondition{}
	// uses the current context in kubeconfig
	// path-to-kubeconfig -- for example, /root/.kube/config
	config, _ := clientcmd.BuildConfigFromFlags("", "/Users/hexixi/Documents/code/go/src/github.com/IrisIris/autoscaler-measurer/testClusterConfig")
	// creates the clientset
	clientset, _ := kubernetes.NewForConfig(config)
	// access the API to list nodes
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, nds := range nodes.Items {
		//bytes,_ := json.Marshal(nds)
		//fmt.Printf("NodeName: %s\n", string(bytes))
		//return
		for _, v := range nds.Status.Conditions {
			if v.Type == "Ready" && v.Status == "True" {
				res[nds.ObjectMeta.Name] = v
			}
		}
	}
	return res
	//pods, _ := clientset.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
	//fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
}
