package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/one2nc/cloudlens/internal/config"
	"github.com/rs/zerolog/log"
)

func GetInstances(cfg aws.Config) ([]EC2Resp, error) {
	var ec2Info []EC2Resp
	ec2Client := ec2.NewFromConfig(cfg)
	resultec2, err := ec2Client.DescribeInstances(context.TODO(), nil)
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error fetching instances: %v", err))
		return nil, err
	}

	// Iterate through the instances and print their ID and state
	for _, reservation := range resultec2.Reservations {
		for _, instance := range reservation.Instances {
			launchTime := instance.LaunchTime
			localZone, err := config.GetLocalTimeZone() // Empty string loads the local timezone
			if err != nil {
				fmt.Println("Error loading local timezone:", err)
				return nil, err
			}
			loc, _ := time.LoadLocation(localZone)
			IST := launchTime.In(loc)

			tags := map[string]string{}
			for key := range instance.Tags {
				tags[*instance.Tags[key].Key] = *instance.Tags[key].Value
			}

			ec2Resp := &EC2Resp{
				Name:             tags["Name"],
				InstanceId:       *instance.InstanceId,
				InstanceType:     string(instance.InstanceType),
				AvailabilityZone: *instance.Placement.AvailabilityZone,
				InstanceState:    string(instance.State.Name),
				PublicDNS:        *instance.PublicDnsName,
				MonitoringState:  string(instance.Monitoring.State),
				LaunchTime:       IST.Format("Mon Jan _2 15:04:05 2006")}
			ec2Info = append(ec2Info, *ec2Resp)
		}
	}
	return ec2Info, nil
}

func GetSingleInstance(cfg aws.Config, insId string) string {
	ec2Client := ec2.NewFromConfig(cfg)
	result, err := ec2Client.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{insId},
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error fetching instance with id: %s, err: %v", insId, err))
		return ""
	}
	r, _ := json.MarshalIndent(result, "", " ")
	return string(r)
}

func GetSingleInstanceDetail(cfg aws.Config, insId string) *EC2DetailResp {
	ec2Client := ec2.NewFromConfig(cfg)
	result, err := ec2Client.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{insId},
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error fetching instance with id: %s, err: %v", insId, err))
		return nil
	}
	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil
	}

	inst := result.Reservations[0].Instances[0]

	tags := map[string]string{}
	for _, t := range inst.Tags {
		tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}

	sgs := make([]EC2SecurityGroup, 0, len(inst.SecurityGroups))
	for _, sg := range inst.SecurityGroups {
		sgs = append(sgs, EC2SecurityGroup{
			GroupId:   aws.ToString(sg.GroupId),
			GroupName: aws.ToString(sg.GroupName),
		})
	}

	iamProfile := ""
	if inst.IamInstanceProfile != nil {
		iamProfile = aws.ToString(inst.IamInstanceProfile.Arn)
	}

	localZone, err := config.GetLocalTimeZone()
	launchTimeStr := ""
	if inst.LaunchTime != nil {
		if err == nil {
			loc, _ := time.LoadLocation(localZone)
			launchTimeStr = inst.LaunchTime.In(loc).Format("Mon Jan _2 15:04:05 2006")
		} else {
			launchTimeStr = inst.LaunchTime.UTC().Format("Mon Jan _2 15:04:05 2006")
		}
	}

	tenancy := ""
	az := ""
	if inst.Placement != nil {
		tenancy = string(inst.Placement.Tenancy)
		az = aws.ToString(inst.Placement.AvailabilityZone)
	}

	return &EC2DetailResp{
		InstanceId:      aws.ToString(inst.InstanceId),
		Name:            tags["Name"],
		InstanceState:   string(inst.State.Name),
		InstanceType:    string(inst.InstanceType),
		AvailabilityZone: az,
		Tenancy:         tenancy,
		VpcId:           aws.ToString(inst.VpcId),
		SubnetId:        aws.ToString(inst.SubnetId),
		PrivateIP:       aws.ToString(inst.PrivateIpAddress),
		PublicIP:        aws.ToString(inst.PublicIpAddress),
		PrivateDNS:      aws.ToString(inst.PrivateDnsName),
		PublicDNS:       aws.ToString(inst.PublicDnsName),
		ImageId:         aws.ToString(inst.ImageId),
		KeyName:         aws.ToString(inst.KeyName),
		Architecture:    string(inst.Architecture),
		RootDeviceName:  aws.ToString(inst.RootDeviceName),
		RootDeviceType:  string(inst.RootDeviceType),
		IamProfile:      iamProfile,
		MonitoringState: string(inst.Monitoring.State),
		LaunchTime:      launchTimeStr,
		SecurityGroups:  sgs,
		Tags:            tags,
	}
}

func GetInstanceMonitoring(cfg aws.Config, instanceId string) *EC2MonitoringResp {
	result := &EC2MonitoringResp{
		InstanceStatus: "unavailable",
		SystemStatus:   "unavailable",
	}

	ec2Client := ec2.NewFromConfig(cfg)
	statusOut, err := ec2Client.DescribeInstanceStatus(context.Background(), &ec2.DescribeInstanceStatusInput{
		InstanceIds:         []string{instanceId},
		IncludeAllInstances: aws.Bool(true),
	})
	if err == nil && len(statusOut.InstanceStatuses) > 0 {
		s := statusOut.InstanceStatuses[0]
		result.InstanceStatus = string(s.InstanceStatus.Status)
		result.SystemStatus = string(s.SystemStatus.Status)
	}

	cwClient := cloudwatch.NewFromConfig(cfg)
	now := time.Now()
	start1h := now.Add(-1 * time.Hour)
	start5m := now.Add(-5 * time.Minute)
	dim := []cwtypes.Dimension{{Name: aws.String("InstanceId"), Value: aws.String(instanceId)}}

	cpuOut, err := cwClient.GetMetricStatistics(context.Background(), &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("CPUUtilization"),
		Dimensions: dim,
		StartTime:  &start1h,
		EndTime:    &now,
		Period:     aws.Int32(3600),
		Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage},
	})
	if err == nil && len(cpuOut.Datapoints) > 0 {
		result.CPUAvg1h = aws.ToFloat64(cpuOut.Datapoints[0].Average)
		result.CPUAvg1hOK = true
	}

	sparkOut, err := cwClient.GetMetricStatistics(context.Background(), &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("CPUUtilization"),
		Dimensions: dim,
		StartTime:  &start1h,
		EndTime:    &now,
		Period:     aws.Int32(300),
		Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage},
	})
	if err == nil && len(sparkOut.Datapoints) > 0 {
		pts := sparkOut.Datapoints
		for i := 0; i < len(pts)-1; i++ {
			for j := i + 1; j < len(pts); j++ {
				if pts[j].Timestamp.Before(*pts[i].Timestamp) {
					pts[i], pts[j] = pts[j], pts[i]
				}
			}
		}
		spark := make([]float64, len(pts))
		for i, p := range pts {
			spark[i] = aws.ToFloat64(p.Average)
		}
		result.CPUSpark = spark
	}

	memOut, err := cwClient.GetMetricStatistics(context.Background(), &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("CWAgent"),
		MetricName: aws.String("mem_used_percent"),
		Dimensions: dim,
		StartTime:  &start5m,
		EndTime:    &now,
		Period:     aws.Int32(300),
		Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage},
	})
	if err == nil && len(memOut.Datapoints) > 0 {
		result.MemUsedPct = aws.ToFloat64(memOut.Datapoints[0].Average)
		result.MemOK = true
	}

	netIn, err := cwClient.GetMetricStatistics(context.Background(), &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("NetworkIn"),
		Dimensions: dim,
		StartTime:  &start5m,
		EndTime:    &now,
		Period:     aws.Int32(300),
		Statistics: []cwtypes.Statistic{cwtypes.StatisticSum},
	})
	netOut, err2 := cwClient.GetMetricStatistics(context.Background(), &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("NetworkOut"),
		Dimensions: dim,
		StartTime:  &start5m,
		EndTime:    &now,
		Period:     aws.Int32(300),
		Statistics: []cwtypes.Statistic{cwtypes.StatisticSum},
	})
	if err == nil && len(netIn.Datapoints) > 0 {
		result.NetInAvg5m = aws.ToFloat64(netIn.Datapoints[0].Sum)
	}
	if err2 == nil && len(netOut.Datapoints) > 0 {
		result.NetOutAvg5m = aws.ToFloat64(netOut.Datapoints[0].Sum)
	}
	result.NetOK = err == nil || err2 == nil

	return result
}

func GetSecGrps(cfg aws.Config) ([]SGResp, error) {
	var sgInfo []SGResp
	ec2Client := ec2.NewFromConfig(cfg)
	result, err := ec2Client.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		panic("failed to describe security groups, " + err.Error())
	}

	for _, sg := range result.SecurityGroups {
		sgResp := &SGResp{
			GroupId:     *sg.GroupId,
			GroupName:   *sg.GroupName,
			Description: *sg.Description,
			OwnerId:     *sg.OwnerId,
			VpcId:       *sg.VpcId,
		}
		sgInfo = append(sgInfo, *sgResp)
	}
	return sgInfo, nil
}

func GetSingleSecGrp(cfg aws.Config, sgId string) string {
	ec2Serv := *ec2.NewFromConfig(cfg)
	result, err := ec2Serv.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{sgId},
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching Security Group: %s err: %v ", sgId, err))
		return ""
	}
	r, _ := json.MarshalIndent(result, "", " ")
	return string(r)
}

func GetSingleSecurityGroup(cfg aws.Config, sgId string) *SGDetailResp {
	ec2Serv := *ec2.NewFromConfig(cfg)
	result, err := ec2Serv.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{sgId},
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching Security Group: %s err: %v ", sgId, err))
		return nil
	}

	if len(result.SecurityGroups) == 0 {
		return nil
	}

	sg := result.SecurityGroups[0]
	detail := &SGDetailResp{
		GroupId:              *sg.GroupId,
		GroupName:            *sg.GroupName,
		Description:          *sg.Description,
		OwnerId:              *sg.OwnerId,
		VpcId:                *sg.VpcId,
		IpPermissions:        make([]IpPermission, 0),
		IpPermissionsEgress:  make([]IpPermission, 0),
	}

	for _, perm := range sg.IpPermissions {
		p := IpPermission{
			IpProtocol:       perm.IpProtocol,
			FromPort:         perm.FromPort,
			ToPort:           perm.ToPort,
			IpRanges:         make([]IpRange, 0),
			UserIdGroupPairs: make([]UserIdGroupPair, 0),
			PrefixListIds:    make([]PrefixListId, 0),
		}
		for _, ipRange := range perm.IpRanges {
			p.IpRanges = append(p.IpRanges, IpRange{
				CidrIp:      *ipRange.CidrIp,
				Description: derefString(ipRange.Description),
			})
		}
		for _, pair := range perm.UserIdGroupPairs {
			p.UserIdGroupPairs = append(p.UserIdGroupPairs, UserIdGroupPair{
				GroupId:     derefString(pair.GroupId),
				Description: derefString(pair.Description),
			})
		}
		for _, pl := range perm.PrefixListIds {
			p.PrefixListIds = append(p.PrefixListIds, PrefixListId{
				PrefixListId: *pl.PrefixListId,
			})
		}
		detail.IpPermissions = append(detail.IpPermissions, p)
	}

	for _, perm := range sg.IpPermissionsEgress {
		p := IpPermission{
			IpProtocol:       perm.IpProtocol,
			FromPort:         perm.FromPort,
			ToPort:           perm.ToPort,
			IpRanges:         make([]IpRange, 0),
			UserIdGroupPairs: make([]UserIdGroupPair, 0),
			PrefixListIds:    make([]PrefixListId, 0),
		}
		for _, ipRange := range perm.IpRanges {
			p.IpRanges = append(p.IpRanges, IpRange{
				CidrIp:      *ipRange.CidrIp,
				Description: derefString(ipRange.Description),
			})
		}
		for _, pair := range perm.UserIdGroupPairs {
			p.UserIdGroupPairs = append(p.UserIdGroupPairs, UserIdGroupPair{
				GroupId:     derefString(pair.GroupId),
				Description: derefString(pair.Description),
			})
		}
		for _, pl := range perm.PrefixListIds {
			p.PrefixListIds = append(p.PrefixListIds, PrefixListId{
				PrefixListId: *pl.PrefixListId,
			})
		}
		detail.IpPermissionsEgress = append(detail.IpPermissionsEgress, p)
	}

	return detail
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func GetVolumes(cfg aws.Config) ([]EBSResp, error) {
	var volumes []EBSResp
	ec2Client := ec2.NewFromConfig(cfg)
	result, err := ec2Client.DescribeVolumes(context.Background(), &ec2.DescribeVolumesInput{})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching Volumes. err: %v", err))
		return nil, err
	}
	for _, v := range result.Volumes {
		launchTime := v.CreateTime
		localZone, err := config.GetLocalTimeZone() // Empty string loads the local timezone
		if err != nil {
			fmt.Println("Error loading local timezone:", err)
			return nil, err
		}
		loc, _ := time.LoadLocation(localZone)
		IST := launchTime.In(loc)
		IST.Format("Mon Jan _2 15:04:05 2006")
		volume := EBSResp{
			VolumeId:         *v.VolumeId,
			Size:             strconv.Itoa(int(*v.Size)) + " GB",
			VolumeType:       string(v.VolumeType),
			State:            string(v.State),
			AvailabilityZone: *v.AvailabilityZone,
			Snapshot:         *v.SnapshotId,
			CreationTime:     IST.String(),
		}
		volumes = append(volumes, volume)
	}
	return volumes, nil
}

func GetSingleVolume(cfg aws.Config, vId string) string {
	ec2Client := ec2.NewFromConfig(cfg)
	result, err := ec2Client.DescribeVolumes(context.Background(), &ec2.DescribeVolumesInput{
		VolumeIds: []string{vId},
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching Volume: %s err: %v", vId, err))
	}
	volString, err := json.MarshalIndent(result.Volumes[0], "", " ")
	return string(volString)
}

/*
Snapshots are region specific
Localstack does have default snapshots, so we can see some of the snapshots that we never created
*/
func GetSnapshots(cfg aws.Config) []Snapshot {
	ec2Client := ec2.NewFromConfig(cfg)
	result, err := ec2Client.DescribeSnapshots(context.Background(), &ec2.DescribeSnapshotsInput{})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching Snapshots, err: %v", err))
		return nil
	}
	var snapshots []Snapshot
	for _, s := range result.Snapshots {
		launchTime := s.StartTime
		localZone, err := config.GetLocalTimeZone() // Empty string loads the local timezone
		if err != nil {
			fmt.Println("Error loading local timezone:", err)
			return nil
		}
		loc, _ := time.LoadLocation(localZone)
		IST := launchTime.In(loc)
		IST.Format("Mon Jan _2 15:04:05 2006")
		snapshot := Snapshot{
			SnapshotId: *s.SnapshotId,
			OwnerId:    *s.OwnerId,
			VolumeId:   *s.VolumeId,
			VolumeSize: strconv.Itoa(int(*s.VolumeSize)),
			StartTime:  IST.String(),
			State:      string(s.State),
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

func GetSingleSnapshot(cfg aws.Config, sId string) string {
	ec2Serv := ec2.NewFromConfig(cfg)
	result, err := ec2Serv.DescribeSnapshots(context.Background(), &ec2.DescribeSnapshotsInput{
		SnapshotIds: []string{sId},
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching Snapshot: %s err: %v", sId, err))
	}
	snapshotString, err := json.MarshalIndent(result.Snapshots[0], "", " ")
	return string(snapshotString)
}

/*
	AMIs are region specific
	Localstack does have default some AMIs, so we can see some of the AMIs that we never created
*/

func GetAMIs(cfg aws.Config) []ImageResp {
	ec2Serv := ec2.NewFromConfig(cfg)
	result, err := ec2Serv.DescribeImages(context.Background(), &ec2.DescribeImagesInput{})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching AMIs, err: %v", err))
		return nil
	}
	var images []ImageResp
	for _, i := range result.Images {
		image := ImageResp{
			ImageId:       *i.ImageId,
			OwnerId:       *i.OwnerId,
			ImageLocation: *i.ImageLocation,
			Name:          *i.Name,
			ImageType:     string(i.ImageType),
		}
		images = append(images, image)
	}
	return images
}

func GetSingleAMI(cfg aws.Config, amiId string) string {
	ec2Serv := ec2.NewFromConfig(cfg)
	result, err := ec2Serv.DescribeImages(context.Background(), &ec2.DescribeImagesInput{
		ImageIds: []string{amiId},
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching AMI: %s err: %v ", amiId, err))
	}
	volString, err := json.MarshalIndent(result.Images[0], "", " ")
	return string(volString)
}

func GetVPCs(cfg aws.Config) []VpcResp {
	ec2Serv := ec2.NewFromConfig(cfg)
	result, err := ec2Serv.DescribeVpcs(context.Background(), &ec2.DescribeVpcsInput{})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching VPCs. err: %v ", err))
		return nil
	}
	var vpcs []VpcResp
	for _, v := range result.Vpcs {
		vpc := VpcResp{
			VpcId:           *v.VpcId,
			OwnerId:         *v.OwnerId,
			CidrBlock:       *v.CidrBlock,
			InstanceTenancy: string(v.InstanceTenancy),
			State:           string(v.State),
		}
		vpcs = append(vpcs, vpc)
	}
	return vpcs
}

func GetSingleVPC(cfg aws.Config, vpcId string) string {
	ec2Serv := ec2.NewFromConfig(cfg)
	result, err := ec2Serv.DescribeVpcs(context.Background(), &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcId},
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching VPC: %s, err: %v", vpcId, err))
		return ""
	}
	vpcString, err := json.MarshalIndent(result.Vpcs[0], "", " ")
	return string(vpcString)
}

func GetSubnets(cfg aws.Config, vpcId string) []SubnetResp {
	ec2Serv := ec2.NewFromConfig(cfg)
	result, err := ec2Serv.DescribeSubnets(context.Background(),
		&ec2.DescribeSubnetsInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: []string{(vpcId)},
				},
			},
		})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching Subnets. err: %v", err))
		return nil
	}
	var subnets []SubnetResp
	for _, s := range result.Subnets {
		subnet := SubnetResp{
			SubnetId:         *s.SubnetId,
			OwnerId:          *s.OwnerId,
			CidrBlock:        *s.CidrBlock,
			AvailabilityZone: *s.AvailabilityZone,
			State:            string(s.State),
		}
		subnets = append(subnets, subnet)
	}
	return subnets
}

func GetSingleSubnet(cfg aws.Config, sId string) string {
	ec2Serv := ec2.NewFromConfig(cfg)
	result, err := ec2Serv.DescribeSubnets(context.Background(), &ec2.DescribeSubnetsInput{
		SubnetIds: []string{sId},
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error in fetching Subnet: %s, err: %v", sId, err))
		return ""
	}
	subnetString, err := json.MarshalIndent(result.Subnets[0], "", " ")
	return string(subnetString)
}
