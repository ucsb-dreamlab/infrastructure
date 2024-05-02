package main

import (
	"fmt"

	// "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"

	ec2 "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	ec2x "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"

	// ecrx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		conf := config.New(ctx, "")
		resourceName := func(n string) string { return fmt.Sprintf("%s-%s-%s", n, ctx.Project(), ctx.Stack()) }

		vpc, err := ec2x.NewVpc(ctx, resourceName("vpc"), &ec2x.VpcArgs{
			NumberOfAvailabilityZones: pulumi.IntRef(2),
			NatGateways: &ec2x.NatGatewayConfigurationArgs{
				Strategy: ec2x.NatGatewayStrategyNone,
			}})
		if err != nil {
			return err
		}

		sg, err := ec2.NewSecurityGroup(ctx, resourceName("ssh_tls"), &ec2.SecurityGroupArgs{
			VpcId: vpc.VpcId,
			Ingress: &ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					FromPort:       pulumi.Int(22),
					ToPort:         pulumi.Int(22),
					Protocol:       pulumi.String("tcp"),
					CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
					Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					FromPort:       pulumi.Int(443),
					ToPort:         pulumi.Int(443),
					Protocol:       pulumi.String("tcp"),
					CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
					Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
				},
			},
			Egress: &ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort:       pulumi.Int(0),
					ToPort:         pulumi.Int(0),
					Protocol:       pulumi.String("-1"),
					CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
					Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
				},
			},
		})
		if err != nil {
			return err
		}
		zoneLookup, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{
			Name: pulumi.StringRef(conf.Get("dns_zone")),
		})
		if err != nil {
			return err
		}
		zoneID := pulumi.String(zoneLookup.Id)

		apiInstance, err := newInstance(ctx, "api", vpc, sg, bucketPolicyDocument("ocfl"), zoneID)
		coderInstance, err := newInstance(ctx, "coder", vpc, sg, pulumi.String(awsPolicyCoder), zoneID)
		ctx.Export("api-ip", apiInstance.PublicIp)
		ctx.Export("coder-ip", coderInstance.PublicIp)
		return nil
	})
}
