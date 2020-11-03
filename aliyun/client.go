package aliyun

import (
	"github.com/IrisIris/autoscaler-measurer/types"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	log "github.com/sirupsen/logrus"
)

func GetEssClient(ak types.AccessKey, region string) *ess.Client {
	client, err := ess.NewClientWithAccessKey(region, ak.GetAccessKeyID(), ak.GetAccessKeySecret())
	if err != nil {
		log.Errorf("failed to create ess client with AccessKeyId and AccessKeySecret,Because of %s", err.Error())
	}
	client.GetConfig().MaxRetryTime = 1
	return client
}
