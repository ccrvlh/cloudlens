package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/one2nc/cloudlens/internal/config"
	"github.com/rs/zerolog/log"
)

func GetEKSClusters(cfg aws.Config) ([]EKSClusterResp, error) {
	eksClient := eks.NewFromConfig(cfg)
	result, err := eksClient.ListClusters(context.Background(), &eks.ListClustersInput{})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error listing EKS clusters: %v", err))
		return nil, err
	}

	var clusters []EKSClusterResp
	for _, name := range result.Clusters {
		clusterName := name
		desc, err := eksClient.DescribeCluster(context.Background(), &eks.DescribeClusterInput{
			Name: &clusterName,
		})
		if err != nil {
			log.Info().Msg(fmt.Sprintf("Error describing EKS cluster %s: %v", clusterName, err))
			continue
		}
		c := desc.Cluster
		createdAt := ""
		if c.CreatedAt != nil {
			localZone, tzErr := config.GetLocalTimeZone()
			if tzErr == nil {
				loc, _ := time.LoadLocation(localZone)
				createdAt = c.CreatedAt.In(loc).Format("Mon Jan _2 15:04:05 2006")
			} else {
				createdAt = c.CreatedAt.UTC().Format("Mon Jan _2 15:04:05 2006")
			}
		}
		clusters = append(clusters, EKSClusterResp{
			Name:      aws.ToString(c.Name),
			Status:    string(c.Status),
			Version:   aws.ToString(c.Version),
			Arn:       aws.ToString(c.Arn),
			CreatedAt: createdAt,
		})
	}
	return clusters, nil
}

func GetEKSCluster(cfg aws.Config, clusterName string) *EKSClusterDetailResp {
	eksClient := eks.NewFromConfig(cfg)
	result, err := eksClient.DescribeCluster(context.Background(), &eks.DescribeClusterInput{
		Name: &clusterName,
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error describing EKS cluster %s: %v", clusterName, err))
		return nil
	}
	c := result.Cluster

	detail := &EKSClusterDetailResp{
		Name:     aws.ToString(c.Name),
		Status:   string(c.Status),
		Version:  aws.ToString(c.Version),
		Arn:      aws.ToString(c.Arn),
		Endpoint: aws.ToString(c.Endpoint),
		RoleArn:  aws.ToString(c.RoleArn),
		Tags:     c.Tags,
	}

	if c.ResourcesVpcConfig != nil {
		detail.VpcId            = aws.ToString(c.ResourcesVpcConfig.VpcId)
		detail.SubnetIds        = c.ResourcesVpcConfig.SubnetIds
		detail.SecurityGroupIds = c.ResourcesVpcConfig.SecurityGroupIds
		detail.ClusterSGId      = aws.ToString(c.ResourcesVpcConfig.ClusterSecurityGroupId)
		detail.PublicAccess     = c.ResourcesVpcConfig.EndpointPublicAccess
		detail.PrivateAccess    = c.ResourcesVpcConfig.EndpointPrivateAccess
	}

	if c.CreatedAt != nil {
		localZone, tzErr := config.GetLocalTimeZone()
		if tzErr == nil {
			loc, _ := time.LoadLocation(localZone)
			detail.CreatedAt = c.CreatedAt.In(loc).Format("Mon Jan _2 15:04:05 2006")
		} else {
			detail.CreatedAt = c.CreatedAt.UTC().Format("Mon Jan _2 15:04:05 2006")
		}
	}

	return detail
}

func GetEKSNodeGroups(cfg aws.Config, clusterName string) []EKSNodeGroupResp {
	eksClient := eks.NewFromConfig(cfg)
	listResult, err := eksClient.ListNodegroups(context.Background(), &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error listing node groups for cluster %s: %v", clusterName, err))
		return nil
	}

	var nodeGroups []EKSNodeGroupResp
	for _, name := range listResult.Nodegroups {
		ngName := name
		desc, err := eksClient.DescribeNodegroup(context.Background(), &eks.DescribeNodegroupInput{
			ClusterName:   &clusterName,
			NodegroupName: &ngName,
		})
		if err != nil {
			log.Info().Msg(fmt.Sprintf("Error describing node group %s: %v", ngName, err))
			continue
		}
		ng := desc.Nodegroup
		resp := EKSNodeGroupResp{
			Name:         aws.ToString(ng.NodegroupName),
			Status:       string(ng.Status),
			InstanceTypes: ng.InstanceTypes,
			CapacityType: string(ng.CapacityType),
			AmiType:      string(ng.AmiType),
			NodeGroupArn: aws.ToString(ng.NodegroupArn),
		}
		if ng.DiskSize != nil {
			resp.DiskSize = *ng.DiskSize
		}
		if ng.ScalingConfig != nil {
			if ng.ScalingConfig.DesiredSize != nil {
				resp.DesiredSize = *ng.ScalingConfig.DesiredSize
			}
			if ng.ScalingConfig.MinSize != nil {
				resp.MinSize = *ng.ScalingConfig.MinSize
			}
			if ng.ScalingConfig.MaxSize != nil {
				resp.MaxSize = *ng.ScalingConfig.MaxSize
			}
		}
		nodeGroups = append(nodeGroups, resp)
	}
	return nodeGroups
}

func GetEKSAddons(cfg aws.Config, clusterName string) []EKSAddonResp {
	eksClient := eks.NewFromConfig(cfg)
	listResult, err := eksClient.ListAddons(context.Background(), &eks.ListAddonsInput{
		ClusterName: &clusterName,
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error listing add-ons for cluster %s: %v", clusterName, err))
		return nil
	}

	var addons []EKSAddonResp
	for _, name := range listResult.Addons {
		addonName := name
		desc, err := eksClient.DescribeAddon(context.Background(), &eks.DescribeAddonInput{
			ClusterName: &clusterName,
			AddonName:   &addonName,
		})
		if err != nil {
			log.Info().Msg(fmt.Sprintf("Error describing add-on %s: %v", addonName, err))
			continue
		}
		a := desc.Addon
		addons = append(addons, EKSAddonResp{
			Name:    aws.ToString(a.AddonName),
			Version: aws.ToString(a.AddonVersion),
			Status:  string(a.Status),
		})
	}
	return addons
}

func GetEKSClusterJSON(cfg aws.Config, clusterName string) string {
	eksClient := eks.NewFromConfig(cfg)
	result, err := eksClient.DescribeCluster(context.Background(), &eks.DescribeClusterInput{
		Name: &clusterName,
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error describing EKS cluster %s: %v", clusterName, err))
		return ""
	}
	r, _ := json.MarshalIndent(result, "", " ")
	return string(r)
}
