package types

import "time"

const LAYOUT = "2006-01-02T15:04Z"

type CoreTimeStamp struct {
	TriggerTime   string
	RunningTime   string
	InServiceTime string
	ReadyTime     string
}

type NodeResult struct {
	Nodes []*NodeAttribute `json:"nodes"`
	Page  Pagination       `json:"page"`
}

type NodeRole string
type State string
type InstanceChargeType string

type NodeAttribute struct {
	InstanceId         string             `json:"instance_id"`
	InstanceRole       NodeRole           `json:"instance_role"`
	InstanceName       string             `json:"instance_name"`
	HostName           string             `json:"host_name"`
	NodeName           string             `json:"node_name"`
	InstanceType       string             `json:"instance_type"`
	CreationTime       time.Time          `json:"creation_time"`
	ExpiredTime        time.Time          `json:"expired_time"`
	InstanceChargeType InstanceChargeType `json:"instance_charge_type"`
	ImageId            string             `json:"image_id"`
	InstanceTypeFamily string             `json:"instance_type_family"`
	IpAddress          []string           `json:"ip_address"`
	IsNewNode          int                `json:"-"`
	DockerVersion      string             `json:"docker_version,omitempty"`
	AgentVersion       string             `json:"agent_version,omitempty"`
	IsLeader           bool               `json:"is_leader,omitempty"`
	Containers         int                `json:"containers,omitempty"`
	IsAliyunNode       bool               `json:"is_aliyun_node"`
	State              State              `json:"state"`
	Source             string             `json:"source"`
	NodePoolId         string             `json:"nodepool_id"`
	ErrorMessage       string             `json:"error_message"`
	NodeStatus         string             `json:"node_status"`
	InstanceStatus     string             `json:"instance_status"`
}

type Pagination struct {
	TotalCount int64 `json:"total_count"`
	PageNumber int64 `json:"page_number"`
	PageSize   int64 `json:"page_size"`
}

type AccessKey interface {
	// Get ID
	GetAccessKeyID() string
	GetAccessKeySecret() string
	GetSecurityToken() string
}

type AKInfo struct {
	AccessKeyID     string `json:"access_key_id,omitempty"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
	SecurityToken   string `json:"security_token,omitempty"`
}

func (a *AKInfo) GetAccessKeyID() string {
	return a.AccessKeyID
}
func (a *AKInfo) GetAccessKeySecret() string {
	return a.AccessKeySecret
}
func (a *AKInfo) GetSecurityToken() string {
	return a.SecurityToken
}
