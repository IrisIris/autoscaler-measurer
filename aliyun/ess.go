package aliyun

import (
	"fmt"
	"github.com/IrisIris/autoscaler-measurer/types"
	"github.com/IrisIris/autoscaler-measurer/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	log "github.com/sirupsen/logrus"
	"time"
)

func filterNewScalingActivities(essClient *ess.Client, regionId string, scalingActivities ess.ScalingActivities, scalingGroupId string, startTime time.Time) (*ess.ScalingActivities, error) {
	filtered := ess.ScalingActivities{}
	if len(scalingActivities.ScalingActivity) == 0 {
		return nil, fmt.Errorf("scaling group %s  has no ScalingActivity", scalingGroupId)
	}
	for _, sa := range scalingActivities.ScalingActivity {
		layout := "2006-01-02T15:04Z"
		// filter time
		saStartTime, err := time.Parse(layout, sa.StartTime)
		if err != nil {
			log.Warnf("scaling group %s parse %s start time %v failed, err %v", scalingGroupId, sa.ScalingActivityId, sa.StartTime, err)
			continue
		}
		if saStartTime.After(startTime) {
			//如果是处理中，则继续执行
			if sa.StatusCode == "InProgress" || sa.StatusCode == "Success" {
				// record trigger time
				filtered.ScalingActivity = append(filtered.ScalingActivity, sa)
			}
			log.Warnf("The ScalingActivity %s excute reuslt is Reason : %s,Description : %s", sa.ScalingActivityId, sa.Cause, sa.Description)
			//如果失败，或者被拒绝
			if sa.StatusCode == "Failed" || sa.StatusCode == "Rejected" {
				detail := DescribeScalingActivitieDetails(essClient, regionId, sa.ScalingActivityId)
				log.Warnf("The ScalingActivity %s excute reuslt is Reason : %s,Description : %s, Detail : %s", sa.ScalingActivityId, sa.Cause, sa.Description, detail)
				//return true, fmt.Errorf("ScalingActivity.Failed", fmt.Sprintf("Reson : %s , Description : %s , Detail : %s", ac.Cause, ac.Description, detail), queryScalingActivityResponse.RequestId)
			}
		}
	}
	return &filtered, nil
}

func QueryScalingActivities(essClient *ess.Client, regionId string, scalingGroupId string, runStartTime time.Time, triggerLimitDuration int) (*ess.ScalingActivity, error) {
	log.Infof("Start to query scaling activities for scaling group %s", scalingGroupId)
	queryScalingActivityArgs := ess.CreateDescribeScalingActivitiesRequest()
	queryScalingActivityArgs.RegionId = regionId
	queryScalingActivityArgs.ScalingGroupId = scalingGroupId
	filtered := &ess.ScalingActivities{}

	err := utils.WaitForResultWithError(fmt.Sprintf("Wait for ScalingGroup %s trigger new scaling activity", scalingGroupId), func() (bool, error) {
		queryScalingActivityResponse, err := essClient.DescribeScalingActivities(queryScalingActivityArgs)
		if err != nil {
			return false, err
		}
		//log.Infof("%s Successfully to DescribeScalingActivities", scalingGroupId)
		//log.Infof("%s Successfully to DescribeScalingActivities Response ScalingActivities = %++v", scalingGroupId, queryScalingActivityResponse.ScalingActivities)
		filtered, err = filterNewScalingActivities(essClient, regionId, queryScalingActivityResponse.ScalingActivities, scalingGroupId, runStartTime)
		if err != nil {
			return false, nil
		}
		if len(filtered.ScalingActivity) == 0 {
			return false, fmt.Errorf("%s ScalingActivity not triggered new activities", scalingGroupId)
		}
		// assume one ScalingActivity
		log.Infof("%s after filter scaling groud %v has %d scaling activites: %v", time.Now(), scalingGroupId, len(filtered.ScalingActivity), filtered.ScalingActivity)
		return true, nil
	}, false, 1, triggerLimitDuration)
	return &filtered.ScalingActivity[0], err
}

func DescribeScalingActivities(essClient *ess.Client, regionId string, scalingGroupId, scalingActivityId string) error {
	log.Infof("Start to DescribeScalingActivities for Activities %s", scalingActivityId)
	queryScalingActivityArgs := ess.CreateDescribeScalingActivitiesRequest()
	queryScalingActivityArgs.RegionId = regionId
	queryScalingActivityArgs.ScalingGroupId = scalingGroupId
	queryScalingActivityArgs.ScalingActivityId1 = scalingActivityId

	err := utils.WaitForResultWithError(fmt.Sprintf("Wait for Activitie %s Ready", scalingActivityId), func() (bool, error) {
		queryScalingActivityResponse, err := essClient.DescribeScalingActivities(queryScalingActivityArgs)
		if err != nil {
			return false, err
		}
		log.Infof("Successfully to DescribeScalingActivities Response = %++v", queryScalingActivityResponse)

		if len(queryScalingActivityResponse.ScalingActivities.ScalingActivity) <= 0 {
			return false, fmt.Errorf("ScalingActivity %s not found", scalingActivityId)
		}

		ac := queryScalingActivityResponse.ScalingActivities.ScalingActivity[0]
		//如果是处理中，则继续执行
		if ac.StatusCode == "InProgress" {
			return false, fmt.Errorf("The ScalingActivity %s is InProgress", scalingActivityId)
		}
		log.Warnf("The ScalingActivity %s excute reuslt is Reason : %s,Description : %s", scalingActivityId, ac.Cause, ac.Description)
		//如果失败，或者被拒绝
		if ac.StatusCode == "Failed" || ac.StatusCode == "Rejected" {
			detail := DescribeScalingActivitieDetails(essClient, regionId, scalingActivityId)
			log.Warnf("The ScalingActivity %s excute reuslt is Reason : %s,Description : %s, Detail : %s", scalingActivityId, ac.Cause, ac.Description, detail)
			return true, fmt.Errorf("ScalingActivity.Failed", fmt.Sprintf("Reson : %s , Description : %s , Detail : %s", ac.Cause, ac.Description, detail), queryScalingActivityResponse.RequestId)
		}
		return true, nil
	}, false, 30, 1050)

	if err != nil {
		log.Errorf("Failed to DescribeScalingActivities error %++v", err)
	}
	return err
}

func DescribeScalingActivitieDetails(essClient *ess.Client, regionId string, scalingActivityId string) string {
	log.Infof("Start to DescribeScalingActivitieDetails for Activities %s", scalingActivityId)
	args := ess.CreateDescribeScalingActivityDetailRequest()
	args.RegionId = regionId
	args.ScalingActivityId = scalingActivityId

	queryScalingActivityDetailResponse, err := essClient.DescribeScalingActivityDetail(args)
	if err != nil {
		log.Errorf("Failed to DescribeScalingActivityDetail %++v", err)
		return ""
	}

	log.Infof("Successfully to DescribeScalingActivityDetail Response = %++v", queryScalingActivityDetailResponse)

	return queryScalingActivityDetailResponse.Detail
}

func DescribeScalingInstances(essClient *ess.Client, regionId, sgId string, timeRecorder *map[string]*types.CoreTimeStamp, readyNodes *map[string]string, triggerTime time.Time, exspectNodeNum int64) {
	for pageNum := 1; pageNum < int(exspectNodeNum/50+1)+1; pageNum++ {
		scalingInstances, err := DescribeInstanceScaling(essClient, regionId, sgId, pageNum)
		if err != nil || len(scalingInstances) == 0 {
			log.Warnf("page_num %d got 0 scaling instances", pageNum)
			break
		}
		for _, instance := range scalingInstances {
			if instance.CreationTime != "" {
				instanceCreatedTime, err := time.Parse(types.LAYOUT, instance.CreationTime)
				if err != nil {
					log.Warnf("parse %s start time %v failed, err %v", instance.CreationTime, err)
					continue
				}
				if instanceCreatedTime.Before(triggerTime) {
					//log.Warnf("instance %s created at %s before trigger time %s", instance.InstanceId, instanceCreatedTime, triggerTime)
					(*readyNodes)[instance.InstanceId] = "ready"
					continue
				}
			}

			if times, ok := (*timeRecorder)[instance.InstanceId]; ok && times.RunningTime != "" && times.InServiceTime != "" {
				continue
			}
			log.Infof("start to watch instance %s to be in service", instance.InstanceId)
			// ECS实例在伸缩组中的健康状态，未处于运行中（Running）状态的ECS实例会被判定为不健康的ECS实例
			if instance.HealthStatus == "Healthy" {
				if (*timeRecorder)[instance.InstanceId] == nil {
					(*timeRecorder)[instance.InstanceId] = &types.CoreTimeStamp{}
				}
				(*timeRecorder)[instance.InstanceId].RunningTime = fmt.Sprintf("%s", time.Now())
				if instance.LifecycleState == "InService" {
					(*timeRecorder)[instance.InstanceId].InServiceTime = fmt.Sprintf("%s", time.Now())
				}
			}
		}

	}
}

func DescribeInstanceScaling(essClient *ess.Client, regionId string, sgId string, pageNumber int) ([]ess.ScalingInstance, error) {
	//log.Infof("%s Start to DescribeScalingInstances page number %d", sgId, pageNumber)
	args := ess.CreateDescribeScalingInstancesRequest()
	args.RegionId = regionId
	args.PageSize = requests.NewInteger(50)
	args.PageNumber = requests.NewInteger(pageNumber)
	args.ScalingGroupId = sgId

	// log.Debugf("DescribeScalingInstances Args = %++v", args)
	response, err := essClient.DescribeScalingInstances(args)
	if err != nil {
		log.Errorf("Failed to DescribeScalingInstances error %++v", err)
		return nil, err
	}

	// log.Debugf("DescribeScalingInstances Response = %++v", response)

	return response.ScalingInstances.ScalingInstance, nil
}
