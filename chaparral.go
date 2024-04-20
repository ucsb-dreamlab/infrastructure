package main

import (
	"encoding/json"
	"fmt"

	ec2 "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/awsx"
	ec2x "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	ecrx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	ecsx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecs"
	lbx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const assumeRolPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": ["ecs-task.amazonaws.com"]
      }
    }
  ]
}`

func chaparralTaskPolicyDocument(bucket string) pulumi.String {
	doc, err := json.Marshal(map[string]any{
		"Version": "2012-10-17",
		"Statement": []any{
			map[string]any{
				"Effect": "Allow",
				"Action": []any{
					"s3:GetBucketLocation",
					"s3:ListBucket",
				},
				"Resource": "arn:aws:s3:::" + bucket,
			},
			map[string]any{
				"Effect": "Allow",
				"Action": []any{
					"s3:PutObject",
					"s3:DeleteObject",
					"s3:GetObject",
				},
				"Resource": "arn:aws:s3:::" + bucket + "/*",
			},
		},
	})
	if err != nil {
		panic(fmt.Errorf("generating chaparral task policy: %w", err))
	}
	return pulumi.String(string(doc))
}

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

	dockerV := ecrx.BuilderVersionBuilderV1

	// Build and publish our application's container image from ./app to the ECR repository
	image, err := ecrx.NewImage(ctx, "image", &ecr.ImageArgs{
		RepositoryUrl:  repo.Url,
		BuilderVersion: &dockerV,
		Context:        pulumi.String("./chaparral"),
		Platform:       pulumi.String("linux/amd64"),
	})
	if err != nil {
		return err
	}

	sg, err := ec2.NewSecurityGroup(ctx, "chaparral-sg", &ec2.SecurityGroupArgs{
		VpcId: vpc.VpcId,
		Ingress: &ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				FromPort:       pulumi.Int(8080),
				ToPort:         pulumi.Int(8080),
				Protocol:       pulumi.String("tcp"),
				CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			},
		},
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

	taskRole, err := iam.NewRole(ctx, "chaparral-task-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(assumeRolPolicy),
		InlinePolicies: iam.RoleInlinePolicyArray{
			&iam.RoleInlinePolicyArgs{
				Name:   pulumi.String("chaparral-task-policy"),
				Policy: chaparralTaskPolicyDocument("ocfl"),
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
			// AssignPublicIp: pulumi.Bool(true),
			Subnets: vpc.PrivateSubnetIds,
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
						HostPort:      pulumi.Int(containerPort),
						TargetGroup:   lb.DefaultTargetGroup,
					},
				},
			},
			TaskRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: taskRole.Arn,
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}
