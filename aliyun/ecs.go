package aliyun

import (
	"github.com/IrisIris/autoscaler-measurer/types"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	log "github.com/sirupsen/logrus"
)

func IsEcsRunningTime(regionId string, instanceId string, akInfo types.AccessKey) bool {
	describeRes, err := DescribeEcsInstances(regionId, instanceId, akInfo)
	if err != nil {
		return false
	}
	if len(describeRes.Instances.Instance) == 0 {
		return false
	}
	if describeRes.Instances.Instance[0].Status == "Running" {
		return true
	}
	return false
}

func DescribeEcsInstances(regionId string, instanceId string, akInfo types.AccessKey) (*ecs.DescribeInstancesResponse, error) {
	client, err := ecs.NewClientWithAccessKey(regionId, akInfo.GetAccessKeyID(), akInfo.GetAccessKeySecret())

	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"

	request.InstanceIds = "[\"" + instanceId + "\"]"

	response, err := client.DescribeInstances(request)
	if err != nil {
		log.Errorf("failed to DescribeEcsInstances %s, err: %v", instanceId, err)
		return nil, err
	}
	return response, err
}
