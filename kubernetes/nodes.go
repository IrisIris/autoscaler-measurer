package kubernetes

import (
	"context"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
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
	node, err := k8sClient.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("failed to get node %s, err: %v", nodeName, err)
	}
	for _, v := range node.Status.Conditions {
		if v.Type == "Ready" && v.Status == "True" {
			//res[node.ObjectMeta.Name] = v
			log.Infof("Node %s is ready", node.Name)
			return true
		}
	}
	return false
}

func IsDeploymentReady(k8sClient *kubernetes.Clientset, depName string) bool {
	if k8sClient == nil {
		log.Warn("k8s client is nil")
		return false
	}
	if depName == "" {
		log.Warn("dep name is empty")
		return false
	}
	//res := map[string]coreV1.NodeCondition{}
	// access the API to list nodes
	dep, err := k8sClient.AppsV1().Deployments("default").Get(context.TODO(), depName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("failed to get node %s, err: %v", depName, err)
	}
	bytes, _ := json.Marshal(dep)
	log.Infof("dep is %s", string(bytes))
	if dep.Status.UnavailableReplicas == *dep.Spec.Replicas {
		return false
	}
	if dep.Status.AvailableReplicas == *dep.Spec.Replicas {
		return true
	}
	for _, v := range dep.Status.Conditions {
		if v.Type == "Ready" && v.Status == "True" {
			//res[node.ObjectMeta.Name] = v
			return true
		}
	}
	return false
}

func GetDeployments(k8sClient *kubernetes.Clientset, depName string) *appsv1.Deployment {
	if k8sClient == nil {
		log.Warn("k8s client is nil")
		return nil
	}
	if depName == "" {
		log.Warn("dep name is empty")
		return nil
	}
	//res := map[string]coreV1.NodeCondition{}
	// access the API to list nodes
	dep, err := k8sClient.AppsV1().Deployments("default").Get(context.TODO(), depName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("failed to get node %s, err: %v", depName, err)
		return nil
	}
	return dep
}

func GetPodsList(k8sClient *kubernetes.Clientset) (*v1.PodList, error) {
	if k8sClient == nil {
		log.Warn("k8s client is nil")
	}
	podsList, err := k8sClient.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("failed to get pods list, err: %v", err)
		return nil, err
	}

	return podsList, nil
}

func GetDepPodsTime(k8sClient *kubernetes.Clientset, depName string) bool {
	dep := GetDeployments(k8sClient, depName)
	if dep == nil {
		return false
	}
	podsList, err := GetPodsList(k8sClient)
	if err != nil {
		return false
	}
	log.Infof(" int(*dep.Spec.Replicas) is %d, len(podsList.Items) is %d", int(*dep.Spec.Replicas), len(podsList.Items))
	if int(*dep.Spec.Replicas)+0 == len(podsList.Items) {
		return true
	}
	return false
}

func GetAllPodUnscheduleTime(k8sClient *kubernetes.Clientset, count int) *time.Time {
	podsList, err := GetPodsList(k8sClient)
	if err != nil {
		return nil
	}
	latestTime := time.Now().AddDate(0, 0, -1)
	unscheduledNum := 0
	for _, pod := range podsList.Items {
		for _, v := range pod.Status.Conditions {
			if v.Type == "PodScheduled" && v.Status == "False" {
				//res[node.ObjectMeta.Name] = v
				//log.Infof("%s", v.LastTransitionTime)
				unscheduledNum += 1
				if v.LastTransitionTime.After(latestTime) {
					latestTime = v.LastTransitionTime.Time
				}
			}
		}
	}
	if unscheduledNum >= count {
		return &latestTime
	}
	log.Infof("unscheduledNum is %d", unscheduledNum)
	return nil
}

func GetAllPodsReadyTime(k8sClient *kubernetes.Clientset, count int) *time.Time {
	podsList, err := GetPodsList(k8sClient)
	if err != nil {
		return nil
	}
	latestTime := time.Now().AddDate(0, 0, -1)
	unscheduledNum := 0
	for _, pod := range podsList.Items {
		for _, v := range pod.Status.Conditions {
			if v.Type == "Ready" && v.Status == "True" {
				//res[node.ObjectMeta.Name] = v
				//log.Infof("%s", v.LastTransitionTime)
				unscheduledNum += 1
				if v.LastTransitionTime.After(latestTime) {
					latestTime = v.LastTransitionTime.Time
				}
			}
		}
	}
	if unscheduledNum >= count {
		return &latestTime
	}
	log.Infof("ready pods Num is %d", unscheduledNum)
	return nil
}
