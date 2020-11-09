package main

import (
	"fmt"
	"github.com/IrisIris/autoscaler-measurer/aliyun"
	innerK8s "github.com/IrisIris/autoscaler-measurer/kubernetes"
	"github.com/IrisIris/autoscaler-measurer/types"
	"github.com/IrisIris/autoscaler-measurer/utils"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"time"
)

const (
	// user cluster kubeconfig path
	ClusterKubeConfigPath = "/Users/hexixi/Documents/code/go/src/github.com/IrisIris/autoscaler-measurer/testClusterConfig"
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
	successExpectNum := 4900
	failedExpectNum := 5000
	regionId := "cn-beijing"
	//clusterId := ""
	// essAutoscalingGroupId := ""
	// 1.16.9
	clusterId := ""
	essAutoscalingGroupId := ""
	essScalingPolicy := "recycle" // or "release"
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
	client, _ := innerK8s.GetK8sClient(ClusterKubeConfigPath)
	//fmt.Println(innerK8s.IsNodeReady(client, "cn-beijing.10.128.37.66"))
	//fmt.Println(innerK8s.IsDeploymentReady(client, "nginx"))
	//return

	// pods fail test case
	//depName := "nginx-ok-selector" //
	depName := "nginx-fail" //
	startRunTime := time.Now()
	allPodsExist := time.Now()
	log.Infof("script starts at %s", startRunTime)

	// get deptime
	err := utils.WaitForResultWithError(fmt.Sprintf("Wait for dep replica = pods counts"), func() (bool, error) {
		// 获取ecs弹出时间 判断状态是否为running —— —— ECS running 时间
		equal := innerK8s.GetDepPodsTime(client, depName)
		if equal {
			allPodsExist = time.Now()
			log.Infof("all pods exist time is %s, durations is %d seconds", allPodsExist, allPodsExist.Unix()-startRunTime.Unix())
			return true, nil
		}
		return false, fmt.Errorf("dep replica != pods counts")
	}, false, 2, int(readyLimitDuration.Seconds()))
	latestTime := time.Now().AddDate(-1, 0, 0)
	err = utils.WaitForResultWithError(fmt.Sprintf("Wait for dep replica = pods counts"), func() (bool, error) {
		// 获取ecs弹出时间 判断状态是否为running —— —— ECS running 时间
		lt := innerK8s.GetAllPodUnscheduleTime(client, failedExpectNum)
		if lt != nil {
			latestTime = *lt
			log.Infof("All pods unschedule time is %s durations is %d seconds", latestTime, latestTime.Unix()-allPodsExist.Unix())
			return true, nil
		}
		return false, fmt.Errorf("not get latest time yet")
	}, false, 20, int(readyLimitDuration.Seconds()))
	//return
	//time.Sleep(12)
	// 脚本运行起始时间
	startTime := time.Now()
	fmt.Printf("scale up test start run time is %s\n", time.Now())
	time.Sleep(60 * time.Second)
	log.Infof("start to update sg please")
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

	exspectNodeNum := aliyun.GetScalingActivitAimNum(toWatchScalingActivity.Cause)
	log.Infof("ag all exspectNodeNum is %d", exspectNodeNum)

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
		aliyun.DescribeScalingInstances(essClient, regionId, essAutoscalingGroupId, &ecsStatusMap, &readyNodes, triggerTime, exspectNodeNum, essScalingPolicy, &akInfo)
		// ecs加入集群成为节点时间 —— node ready时间
		SetNodeReadyTime(k8sClient, &ecsStatusMap, &readyNodes, clusterId, akInfo, regionId)
		if len(readyNodes) < int(exspectNodeNum) {
			return false, fmt.Errorf("Time %s: %d instance is in service, exspect %d num", time.Now(), len(readyNodes), int(exspectNodeNum))
		}
		log.Infof("Time %s:all %d nodes are Ready", time.Now(), len(readyNodes))
		return true, nil
	}, false, 10, int(readyLimitDuration.Seconds()))
	if err != nil {
		log.Errorf("failed to wait for nodes be ready,err: %v", err)
	}
	fmt.Printf("Success to wait for all nodes be ready, cluster has %d nodes，new Nodes time details are:\n", len(readyNodes))
	for instanceId, timeStamps := range ecsStatusMap {
		fmt.Printf("%s:  Trigger:Time %s, RunningTime: %s, InServiceTime: %s, ReadyTime: %s \n", instanceId, toWatchScalingActivity.StartTime, (*timeStamps).RunningTime, (*timeStamps).InServiceTime, (*timeStamps).ReadyTime)
	}
	countLatestTime(ecsStatusMap)

	latestTime = time.Now().AddDate(-1, 0, 0)
	err = utils.WaitForResultWithError(fmt.Sprintf("Wait for dep replica = pods counts"), func() (bool, error) {
		// 获取ecs弹出时间 判断状态是否为running —— —— ECS running 时间
		lt := innerK8s.GetAllPodsReadyTime(client, successExpectNum)
		if lt != nil {
			latestTime = *lt
			return true, nil
		}
		return false, fmt.Errorf("not get latest time yet")
	}, false, 10, int(readyLimitDuration.Seconds()))

	// print results
	log.Infof("All pods ready time is %s durations is %d seconds", latestTime, latestTime.Unix()-allPodsExist.Unix())

}

func countLatestTime(ecsStatusMap map[string]*types.CoreTimeStamp) {
	latestRunningTime := time.Now().AddDate(0, 0, -1)
	latestInserviceTime := time.Now().AddDate(0, 0, -1)
	latestReadyTime := time.Now().AddDate(0, 0, -1)
	for _, times := range ecsStatusMap {
		nowRunningTime := times.RunningTime
		//nowRunningTime, err := time.Parse("2006-01-02 15:04:05.000", times.RunningTime)
		//if err != nil {
		//	continue
		//}
		if nowRunningTime.After(latestRunningTime) {
			latestRunningTime = nowRunningTime
		}

		nowInserviceTime := times.InServiceTime
		//nowInserviceTime, err := time.Parse("2006-01-02 15:04:05.000", times.InServiceTime)
		//if err != nil {
		//	continue
		//}
		if nowInserviceTime.After(latestInserviceTime) {
			latestInserviceTime = nowInserviceTime
		}
		nowReadyTime := times.ReadyTime
		//nowReadyTime, err := time.Parse("2006-01-02 15:04:05.000", times.ReadyTime)
		//if err != nil {
		//	continue
		//}
		if nowReadyTime.After(latestReadyTime) {
			latestReadyTime = nowReadyTime
		}
	}
	log.Infof("Latest node running time is %s, Latest ready time is %s, Latest node ready time is %s", latestRunningTime, latestInserviceTime, latestReadyTime)
}
func SetNodeReadyTime(k8sClient *kubernetes.Clientset, timeRecorder *map[string]*types.CoreTimeStamp, readyNodes *map[string]string, clusterId string, akInfo types.AKInfo, region string) {
	if k8sClient == nil {
		log.Warn("k8s client is nil")
		return
	}
	if len(*timeRecorder) == 0 {
		return
	}
	//log.Infof("Before SetNodeReadyTime readyNodes is %v timeRecorder is %v", *readyNodes, *timeRecorder)
	//log.Infof("Before SetNodeReadyTime timeRecorder is %v", *timeRecorder)
	//for k,v := range *timeRecorder {
	//	log.Infof("Before timeRecorder K: %s v: %v", k, *v)
	//}
	resp, err := aliyun.GetNodes(clusterId, akInfo)
	if err != nil {
		log.Errorf("%s Failed to get nodes err: %v", clusterId, err)
		return
	}

	//k8sClinet, _ := getK8sClient()
	log.Infof("nodes length is %d", len(resp.Nodes))
	for _, node := range resp.Nodes {
		if (*readyNodes)[node.InstanceId] == "ready" {
			continue
		}
		//log.Infof("%s is not in readyNodes", node.InstanceId)
		//if times, ok := (*timeRecorder)[node.InstanceId]; ok && !(times.ReadyTime.IsZero()) {
		//	continue
		//}
		if (*timeRecorder)[node.InstanceId] == nil {
			continue
		}
		//log.Infof("%s is in timeRecorder", node.InstanceId)
		// search from apiserver
		nodeName := node.NodeName
		if nodeName == "" {
			if node.IpAddress[0] == "" {
				log.Warnf("%s node ip address is empty, cannot get node name", node.InstanceId)
				continue
			}
			nodeName = region + "." + node.IpAddress[0]
		}
		log.Infof("start to wait for %s （nodename: %s) to be Ready", node.InstanceId, nodeName)
		if innerK8s.IsNodeReady(k8sClient, nodeName) {
			//(*timeRecorder)[node.InstanceId].ReadyTime = fmt.Sprintf("%s", time.Now())
			(*timeRecorder)[node.InstanceId].ReadyTime = time.Now()
			(*readyNodes)[node.InstanceId] = "ready"
			// del from queue
			log.Infof("node %s (instance %s) is ready, not watch anymore", nodeName, node.InstanceId)
		} else {
			//log.Infof("%d nodes(%v) ready", len(*readyNodes), *readyNodes)
			//log.Infof("%s is not ready, status is %s, nodeinfo is %v", node.InstanceId, node.NodeStatus, *node)
		}
	}
	//log.Infof("After SetNodeReadyTime readyNodes is %v ", *readyNodes)
}

func CheckPodReadyTime(k8sClient *kubernetes.Clientset, timeRecorder *map[string]*types.CoreTimeStamp, readyNodes *map[string]string, clusterId string, akInfo types.AKInfo) {

}
