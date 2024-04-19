package main

import (
	ec2 "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/awsx"
	ec2x "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	ecrx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	ecsx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecs"
	lbx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func chaparral(ctx *pulumi.Context, vpc *ec2x.Vpc, repo *ecrx.Repository, cluster *ecs.Cluster, lb *lbx.ApplicationLoadBalancer) error {
	// cfg := config.New(ctx, "")
	containerPort := 8080
	// if param := cfg.GetInt("containerPort"); param != 0 {
	// 	containerPort = param
	// }
	cpu := 512
	// if param := cfg.GetInt("cpu"); param != 0 {
	// 	cpu = param
	// }
	memory := 128
	// if param := cfg.GetInt("memory"); param != 0 {
	// 	memory = param
	// }

	// Build and publish our application's container image from ./app to the ECR repository
	image, err := ecrx.NewImage(ctx, "image", &ecr.ImageArgs{
		RepositoryUrl: repo.Url,
		Context:       pulumi.String("./chaparral"),
		Platform:      pulumi.String("linux/amd64"),
	})
	if err != nil {
		return err
	}

	sg, err := ec2.NewSecurityGroup(ctx, "chaparral-sg", &ec2.SecurityGroupArgs{
		VpcId: vpc.VpcId,
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				FromPort: pulumi.Int(0),
				ToPort:   pulumi.Int(0),
				Protocol: pulumi.String("-1"),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
				Ipv6CidrBlocks: pulumi.StringArray{
					pulumi.String("::/0"),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// Deploy an ECS Service on Fargate to host the application container
	_, err = ecsx.NewFargateService(ctx, "chaparral-service", &ecsx.FargateServiceArgs{
		Cluster: cluster.Arn,
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.Bool(true),
			Subnets:        vpc.PublicSubnetIds,
			SecurityGroups: pulumi.StringArray{
				sg.ID(),
			},
		},
		TaskDefinitionArgs: &ecsx.FargateServiceTaskDefinitionArgs{
			Container: &ecsx.TaskDefinitionContainerDefinitionArgs{
				Name:      pulumi.String("app"),
				Image:     image.ImageUri,
				Cpu:       pulumi.Int(cpu),
				Memory:    pulumi.Int(memory),
				Essential: pulumi.Bool(true),
				Environment: ecsx.TaskDefinitionKeyValuePairArray{
					&ecsx.TaskDefinitionKeyValuePairArgs{
						Name:  pulumi.String("CHAPARRAL_BACKEND"),
						Value: pulumi.String(""),
					},
					&ecsx.TaskDefinitionKeyValuePairArgs{
						Name:  pulumi.String("LITESTREAM_REPLICA_URL"),
						Value: pulumi.String(""),
					},
				},
				PortMappings: ecsx.TaskDefinitionPortMappingArray{
					&ecsx.TaskDefinitionPortMappingArgs{
						ContainerPort: pulumi.Int(containerPort),
						TargetGroup:   lb.DefaultTargetGroup,
					},
				},
			},
			TaskRole: &awsx.DefaultRoleWithPolicyArgs{},
		},
	})
	if err != nil {
		return err
	}
	return nil
}
