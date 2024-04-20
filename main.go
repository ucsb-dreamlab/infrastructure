package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	ec2 "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	// "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	ec2x "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	// ecrx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	lbx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vpc, err := ec2x.NewVpc(ctx, "dreamlab-vpc", &ec2x.VpcArgs{
			NumberOfAvailabilityZones: pulumi.IntRef(2),
			NatGateways: &ec2x.NatGatewayConfigurationArgs{
				Strategy: ec2x.NatGatewayStrategySingle,
			}})
		if err != nil {
			return err
		}

		sg, err := ec2.NewSecurityGroup(ctx, "dreamlab-lb-tls-sg", &ec2.SecurityGroupArgs{
			VpcId: vpc.VpcId,
			Ingress: &ec2.SecurityGroupIngressArray{
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

		cert, err := acm.NewCertificate(ctx, "dreamlab-cert", &acm.CertificateArgs{
			DomainName:       pulumi.String("api.chaparral.io"),
			ValidationMethod: pulumi.String("DNS"),
		})
		zone, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{
			Name: pulumi.StringRef("chaparral.io"),
		})
		if err != nil {
			return err
		}

		domainValidOpt := cert.DomainValidationOptions.Index(pulumi.Int(0))

		domainValidRecord, err := route53.NewRecord(ctx, "dreamlab-cert-record", &route53.RecordArgs{
			ZoneId: pulumi.String(zone.Id),
			Name:   domainValidOpt.ResourceRecordName().Elem(),
			Type:   domainValidOpt.ResourceRecordType().Elem(),
			Records: pulumi.StringArray{
				domainValidOpt.ResourceRecordValue().Elem(),
			},
			Ttl: pulumi.Int(600),
		})
		if err != nil {
			return err
		}

		_, err = acm.NewCertificateValidation(ctx, "dreamlab-cert-validation", &acm.CertificateValidationArgs{
			CertificateArn:        cert.Arn,
			ValidationRecordFqdns: pulumi.StringArray{domainValidRecord.Fqdn},
		})
		if err != nil {
			return err
		}

		// An ALB to serve the container endpoint to the internet
		loadbalancer, err := lbx.NewApplicationLoadBalancer(ctx, "dreamlab-lb", &lbx.ApplicationLoadBalancerArgs{
			SubnetIds: vpc.PublicSubnetIds,
			Listener: &lbx.ListenerArgs{
				Protocol: pulumi.String("HTTPS"),
				// AlpnPolicy:     pulumi.String("HTTP2Preferred"),
				Port:           pulumi.Int(443),
				CertificateArn: cert.Arn,
			},
			DefaultTargetGroup: &lbx.TargetGroupArgs{
				Port:     pulumi.Int(80),
				Protocol: pulumi.String("HTTP"),
				// ProtocolVersion: pulumi.String("HTTP2"),
			},
			SecurityGroups: pulumi.StringArray{sg.ID()},
		})
		if err != nil {
			return err
		}

		// lbx.NewTargetGroupAttachment()

		_, err = route53.NewRecord(ctx, "dreamlab-api-dns", &route53.RecordArgs{
			Name:    pulumi.String("api.chaparral.io"),
			ZoneId:  pulumi.String(zone.Id),
			Type:    pulumi.String("CNAME"),
			Records: pulumi.StringArray{loadbalancer.LoadBalancer.DnsName()},
			Ttl:     pulumi.Int(600),
		})
		if err != nil {
			return err
		}
		// An ECR repository to store our application's container image
		// repo, err := ecrx.NewRepository(ctx, "dreamlab-repo", &ecrx.RepositoryArgs{
		// 	ForceDelete: pulumi.Bool(true),
		// })
		// if err != nil {
		// 	return err
		// }

		// An ECS cluster to deploy into
		// cluster, err = ecs.NewCluster(ctx, "dreamlab-cluster", nil)
		// if err != nil {
		// 	return err
		// }
		// chaparral(ctx, vpc, nil, cluster, loadbalancer)
		// The URL at which the container's HTTP endpoint will be available
		ctx.Export("url", pulumi.Sprintf("http://%s", loadbalancer.LoadBalancer.DnsName()))
		return nil
	})
}
