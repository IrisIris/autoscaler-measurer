package aliyun

import (
	"encoding/json"
	"github.com/IrisIris/autoscaler-measurer/types"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	log "github.com/sirupsen/logrus"
)

func GetNodes(clusterId string, ak types.AKInfo) (*types.NodeResult, error) {
	client, err := sdk.NewClientWithAccessKey("default", ak.GetAccessKeyID(), ak.GetAccessKeySecret())
	if err != nil {
		panic(err)
	}

	request := requests.NewCommonRequest()
	request.Method = "GET"
	request.Scheme = "https" // https | http
	request.Domain = "cs.aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/clusters/" + clusterId + "/nodes"
	request.Headers["Content-Type"] = "application/json"
	//request.QueryParams["RegionId"] = "cn-hangzhou"
	request.QueryParams["pageSize"] = "300"

	body := `{}`
	request.Content = []byte(body)

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return nil, err
	}
	log.Tracef("acs get nodes response is %s", response.GetHttpContentString())
	nodesString := response.GetHttpContentString()
	nodes := types.NodeResult{}
	err = json.Unmarshal([]byte(nodesString), &nodes)
	if err != nil {
		log.Errorf("failed to decode to nodes: %v", nodesString)
		return nil, err
	}
	return &nodes, nil
}
