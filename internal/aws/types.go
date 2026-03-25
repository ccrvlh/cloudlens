package aws

import (
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Resp struct {
	Instance         ec2.Instance
	InstanceId       string
	InstanceType     string
	AvailabilityZone string
	InstanceState    string
	PublicDNS        string
	MonitoringState  string
	LaunchTime       string
	Name             string
}

type EC2SecurityGroup struct {
	GroupId   string
	GroupName string
}

type EC2DetailResp struct {
	InstanceId       string
	Name             string
	InstanceState    string
	InstanceType     string
	AvailabilityZone string
	Tenancy          string
	VpcId            string
	SubnetId         string
	PrivateIP        string
	PublicIP         string
	PrivateDNS       string
	PublicDNS        string
	ImageId          string
	KeyName          string
	Architecture     string
	RootDeviceName   string
	RootDeviceType   string
	IamProfile       string
	MonitoringState  string
	LaunchTime       string
	SecurityGroups   []EC2SecurityGroup
	Tags             map[string]string
}

type S3Object struct {
	SizeInBytes                                        int64
	Name, ObjectType, LastModified, Size, StorageClass string
}

type BucketInfo struct {
	EncryptionConfiguration types.ServerSideEncryptionConfiguration
	LifeCycleRules          []types.LifecycleRule
}

type IAMUSerResp struct {
	UserId       string
	UserName     string
	ARN          string
	CreationTime string
}

type IAMUSerGroupResp struct {
	GroupId      string
	GroupName    string
	ARN          string
	CreationTime string
}

type IAMUSerPolicyResponse struct {
	PolicyArn  string
	PolicyName string
}

type EBSResp struct {
	VolumeId         string
	Size             string
	VolumeType       string
	State            string
	AvailabilityZone string
	Snapshot         string
	CreationTime     string
}

type IAMUSerGroupPolicyResponse struct {
	PolicyArn  string
	PolicyName string
}

type IamRoleResp struct {
	RoleId       string
	RoleName     string
	ARN          string
	CreationTime string
}

type IamRolePolicyResponse struct {
	PolicyArn  string
	PolicyName string
}

type SQSResp struct {
	Name              string
	URL               string
	Type              string
	Created           string
	MessagesAvailable string
	Encryption        string
	MaxMessageSize    string
}

type Snapshot struct {
	SnapshotId string
	OwnerId    string
	VolumeId   string
	VolumeSize string
	StartTime  string
	State      string
}

type ImageResp struct {
	ImageId       string
	OwnerId       string
	ImageLocation string
	Name          string
	ImageType     string
}

type VpcResp struct {
	VpcId           string
	OwnerId         string
	CidrBlock       string
	InstanceTenancy string
	State           string
}

type LambdaResp struct {
	FunctionName string
	Description  string
	Role         string
	FunctionArn  string
	CodeSize     string
	LastModified string
}

type SubnetResp struct {
	SubnetId         string
	OwnerId          string
	CidrBlock        string
	AvailabilityZone string
	State            string
}

type SGResp struct {
	GroupId     string
	GroupName   string
	Description string
	OwnerId     string
	VpcId       string
}

// IpRange represents an IP range with optional description
type IpRange struct {
	CidrIp      string
	Description string
}

// IpPermission represents an ingress/egress permission rule
type IpPermission struct {
	IpProtocol       *string
	FromPort         *int32
	ToPort           *int32
	IpRanges         []IpRange
	UserIdGroupPairs []UserIdGroupPair
	PrefixListIds    []PrefixListId
}

// UserIdGroupPair represents a security group pair reference
type UserIdGroupPair struct {
	GroupId     string
	Description string
}

// PrefixListId represents a prefix list ID
type PrefixListId struct {
	PrefixListId string
}

// SGDetailResp represents detailed security group data with rules
type SGDetailResp struct {
	GroupId              string
	GroupName            string
	Description          string
	OwnerId              string
	VpcId                string
	IpPermissions       []IpPermission
	IpPermissionsEgress []IpPermission
}

type EC2MonitoringResp struct {
	InstanceStatus string
	SystemStatus   string
	CPUAvg1h       float64
	CPUAvg1hOK     bool
	CPUSpark       []float64 // up to 12 × 5-min samples, oldest first
	MemUsedPct     float64
	MemOK          bool
	NetInAvg5m     float64
	NetOutAvg5m    float64
	NetOK          bool
}

type EcsClusterResp struct {
	ClusterName       string
	Status            string
	ClusterArn        string
	RunningTasksCount string
}

type EcsServiceResp struct {
	ServiceName    string
	Status         string
	DesiredCount   string
	RunningCount   string
	TaskDefinition string
	ServiceArn     string
}

type EcsTaskResp struct {
	TaskId string
	*ecsTypes.Task
}

type EKSClusterResp struct {
	Name      string
	Status    string
	Version   string
	Arn       string
	CreatedAt string
}

type EKSClusterDetailResp struct {
	Name             string
	Status           string
	Version          string
	Arn              string
	Endpoint         string
	RoleArn          string
	VpcId            string
	SubnetIds        []string
	SecurityGroupIds []string
	ClusterSGId      string
	PublicAccess     bool
	PrivateAccess    bool
	CreatedAt        string
	Tags             map[string]string
}

type EKSNodeGroupResp struct {
	Name          string
	Status        string
	InstanceTypes []string
	CapacityType  string
	AmiType       string
	DesiredSize   int32
	MinSize       int32
	MaxSize       int32
	DiskSize      int32
	NodeGroupArn  string
}

type EKSAddonResp struct {
	Name    string
	Version string
	Status  string
}

type APIGatewayResp struct {
	ID           string
	Name         string
	Description  string
	EndpointType string
	CreatedDate  string
}

type APIGatewayStage struct {
	StageName    string
	Description  string
	DeploymentID string
	CreatedDate  string
	LastUpdated  string
}

type APIGatewayDetailResp struct {
	ID           string
	Name         string
	Description  string
	EndpointType string
	APIKeySource string
	CreatedDate  string
	Tags         map[string]string
	Stages       []APIGatewayStage
}

type BedrockModelResp struct {
	ModelId          string
	ModelName        string
	ProviderName     string
	InputModalities  string
	OutputModalities string
	Streaming        string
}

type BedrockModelDetailResp struct {
	ModelId            string
	ModelName          string
	ProviderName       string
	InputModalities    []string
	OutputModalities   []string
	StreamingSupported bool
	InferenceTypes     []string
	Customizations     []string
}

type BedrockMetricsResp struct {
	Invocations   float64
	InvocationsOK bool
	LatencyAvgMs  float64
	LatencyOK     bool
	ClientErrors  float64
	ServerErrors  float64
	ErrorsOK      bool
	InputTokens   float64
	OutputTokens  float64
	TokensOK      bool
}

type BillingServiceResp struct {
	ServiceName       string
	CurrentMonthCost  string
	PreviousMonthCost string
	Unit              string
}

type BillingMonthlyResp struct {
	Month  string
	Amount string
	Unit   string
}
