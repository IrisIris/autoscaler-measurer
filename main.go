package main

import (
	"fmt"
	"github.com/IrisIris/autoscaler-measurer/aliyun"
	innerK8s "github.com/IrisIris/autoscaler-measurer/kubernetes"
	"github.com/IrisIris/autoscaler-measurer/types"
	"github.com/IrisIris/autoscaler-measurer/utils"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"strconv"
	"strings"
	"time"
)

const (
	// user cluster kubeconfig path
	ClusterKubeConfigPath = ""
)

func init() {
	// 设置log
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	loglevel, _ := log.ParseLevel("info")
	log.SetLevel(loglevel)
}

func main() {
	// 初始化数据，需要用户输入
	clusterId := "c2536fd0f844b49c193166398bf856269"
	regionId := "cn-zhangjiakou"
	essAutoscalingGroupId := "asg-8vbedcx7707xcpjz2c46"
	akInfo := types.AKInfo{
		AccessKeyID:     "",
		AccessKeySecret: "",
	}

	// 超时时间限制(可自定义)
	triggerLimitDuration := time.Minute * 15
	readyLimitDuration := time.Minute * 120

	// key:instance id
	ecsStatusMap := map[string]*types.CoreTimeStamp{}
	// key:instance id,value: status
	readyNodes := map[string]string{}

	// test
	//client,_ := innerK8s.GetK8sClient(ClusterKubeConfigPath)
	//fmt.Println(innerK8s.IsNodeReady(client, "cn-zhangjiakou.192.168.0.95"))
	//return

	// 脚本运行起始时间
	startTime := time.Now()
	fmt.Printf("start run time is %s\n", time.Now())

	// get ess client
	essClient := aliyun.GetEssClient(&akInfo, regionId)
	if essClient == nil {
		panic("failed to get ess client, err")
	}

	// watch calling scaling Activities triggered time
	// 调用ess sdk 获取ess 伸缩活动时间 —— ECS开始驱动生产时间
	toWatchScalingActivity, err := aliyun.QueryScalingActivities(essClient, regionId, essAutoscalingGroupId, startTime, int(triggerLimitDuration.Seconds()))
	if err != nil {
		fmt.Printf("Feiled to wait scaling activity triggered, err: %v\n", err)
		return
	}
	log.Infof("Success to get scaling activity triggered, Trigger Time is %s for scaling group %s scaling activity %s (details: %v) \n", toWatchScalingActivity.StartTime, toWatchScalingActivity.ScalingGroupId, toWatchScalingActivity.ScalingActivityId, toWatchScalingActivity)

	splitSlice := strings.Split(toWatchScalingActivity.Cause, "to \"")
	resultSlice := strings.Split(splitSlice[len(splitSlice)-1], "\".")
	exspectNodeNumStr := resultSlice[0]
	exspectNodeNum, err := strconv.ParseInt(exspectNodeNumStr, 10, 64)
	if err != nil {
		log.Errorf("Feiled to parse toWatchScalingActivity.AutoCreatedCapacity %s to int,err: %v", toWatchScalingActivity.AutoCreatedCapacity, err)
	} else {
		log.Infof("ag all exspectNodeNum is %d", exspectNodeNum)
	}

	// watch ecs instances status
	// start to query ecs instance status
	k8sClient, err := innerK8s.GetK8sClient(ClusterKubeConfigPath)
	if err != nil {
		log.Panicf("failed to get k8s client, err %v", err)
		return
	}
	err = utils.WaitForResultWithError(fmt.Sprintf("Wait for nodes be ready"), func() (bool, error) {
		// 获取ecs弹出时间 判断状态是否为running —— —— ECS running 时间
		triggerTime, err := time.Parse(types.LAYOUT, toWatchScalingActivity.StartTime)
		if err != nil {
			return false, err
		}
		aliyun.DescribeScalingInstances(essClient, regionId, essAutoscalingGroupId, &ecsStatusMap, &readyNodes, triggerTime, exspectNodeNum)
		// ecs加入集群成为节点时间 —— node ready时间
		SetNodeReadyTime(k8sClient, &ecsStatusMap, &readyNodes, clusterId, akInfo)
		if len(readyNodes) < int(exspectNodeNum) {
			return false, fmt.Errorf("Time %s: %d instance is in service, exspect %d num", time.Now(), len(readyNodes), int(exspectNodeNum))
		}
		return true, nil
	}, false, 2, int(readyLimitDuration.Seconds()))
	if err != nil {
		log.Errorf("failed to wait for nodes be ready,err: %v", err)
	}

	// print results
	fmt.Printf("Success to wait for all nodes be ready, cluster has %d nodes，new Nodes time details are:\n", len(readyNodes))
	for instanceId, timeStamps := range ecsStatusMap {
		fmt.Printf("%s:  Trigger:Time %s, RunningTime: %s, InServiceTime: %s, ReadyTime: %s \n", instanceId, toWatchScalingActivity.StartTime, (*timeStamps).RunningTime, (*timeStamps).InServiceTime, (*timeStamps).ReadyTime)
	}
}

func SetNodeReadyTime(k8sClient *kubernetes.Clientset, timeRecorder *map[string]*types.CoreTimeStamp, readyNodes *map[string]string, clusterId string, akInfo types.AKInfo) {
	if k8sClient == nil {
		log.Warn("k8s client is nil")
		return
	}
	if len(*timeRecorder) == 0 {
		return
	}
	log.Debugf("Before SetNodeReadyTime readyNodes is %v, timeRecorder is %v", *readyNodes, *timeRecorder)
	resp, err := aliyun.GetNodes(clusterId, akInfo)
	if err != nil {
		log.Errorf("%s Failed to get nodes err: %v", clusterId, err)
		return
	}
	//k8sClinet, _ := getK8sClient()
	for _, node := range resp.Nodes {
		if (*readyNodes)[node.InstanceId] == "ready" {
			continue
		}
		if times, ok := (*timeRecorder)[node.InstanceId]; ok && times.ReadyTime != "" {
			continue
		}
		if (*timeRecorder)[node.InstanceId] == nil {
			continue
		}
		// search from apiserver
		nodeName := node.NodeName
		if nodeName == "" {
			if node.IpAddress[0] == "" {
				log.Warnf("%s node ip address is empty, cannot get node name", node.InstanceId)
				continue
			}
			nodeName = "cn-zhangjiakou." + node.IpAddress[0]
		}
		log.Infof("start to wait for %s （nodename: %s) to be Ready", node.InstanceId, nodeName)
		if innerK8s.IsNodeReady(k8sClient, nodeName) {
			(*timeRecorder)[node.InstanceId].ReadyTime = fmt.Sprintf("%s", time.Now())
			// del from queue
			log.Infof("%s is ready, not watch anymore", node.InstanceId)
			(*readyNodes)[node.InstanceId] = "ready"
		} else {
			//log.Infof("%s is not ready, status is %s, nodeinfo is %v", node.InstanceId, node.NodeStatus, *node)
		}
	}
	//log.Infof("After SetNodeReadyTime readyNodes is %v ", *readyNodes)
}
