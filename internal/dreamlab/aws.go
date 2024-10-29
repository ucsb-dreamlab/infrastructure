package dreamlab

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	ZONE = "dreamlab.ucsb.edu"

	dreamlab = "dreamlab"

	vpcResource = "dreamlab_vpc"
	vpcCidr     = "10.226.42.192/26"

	pubSubnetResource = "public_subnet"
	pubSubnetAZ       = "us-west-2a"
	pubSubnetCidr     = "10.226.42.192/27"
	pubSubnetTagName  = "dreamlab Public Subnet 1"

	privSubnetResource = "private_subnet"
	privSubnetAZ       = "us-west-2a"
	privSubnetCidr     = "10.226.42.224/27"
	privSubnetTagName  = "dreamlab Private Subnet 1"
)

type AWS struct {
	Vpc     *ec2.Vpc
	Private *ec2.Subnet
	Public  *ec2.Subnet
	DNS     *route53.Zone
}

func NewAWS(ctx *pulumi.Context) (*AWS, error) {
	// The VPC and Subnets were created using the ""
	vpc, err := ec2.NewVpc(ctx, vpcResource, &ec2.VpcArgs{
		CidrBlock:          pulumi.String(vpcCidr),
		EnableDnsHostnames: pulumi.Bool(true),
		InstanceTenancy:    pulumi.String("default"),
		Tags: pulumi.StringMap{
			"Name":            pulumi.String(dreamlab),
			"tgw-auto-attach": pulumi.String("true"),
			"ucsb:service":    pulumi.String("UCSB Campus Cloud Portfolio"),
		},
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}
	pub, err := ec2.NewSubnet(ctx, pubSubnetResource, &ec2.SubnetArgs{
		AvailabilityZone:               pulumi.String(pubSubnetAZ),
		CidrBlock:                      pulumi.String(pubSubnetCidr),
		MapPublicIpOnLaunch:            pulumi.Bool(true),
		PrivateDnsHostnameTypeOnLaunch: pulumi.String("ip-name"),
		Tags: pulumi.StringMap{
			"Name":         pulumi.String(pubSubnetTagName),
			"Network":      pulumi.String("Public"),
			"ucsb:service": pulumi.String("UCSB Campus Cloud Portfolio"),
		},
		VpcId: vpc.ID(),
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}
	priv, err := ec2.NewSubnet(ctx, privSubnetResource, &ec2.SubnetArgs{
		AvailabilityZone:               pulumi.String(privSubnetAZ),
		CidrBlock:                      pulumi.String(privSubnetCidr),
		PrivateDnsHostnameTypeOnLaunch: pulumi.String("ip-name"),
		Tags: pulumi.StringMap{
			"Name":                   pulumi.String(privSubnetTagName),
			"Network":                pulumi.String("Private"),
			"ucsb:service":           pulumi.String("UCSB Campus Cloud Portfolio"),
			"dreamlab:service:coder": pulumi.String("workers"),
		},
		VpcId: vpc.ID(),
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}
	zone, err := route53.NewZone(ctx, "dreamlab_dns", &route53.ZoneArgs{
		Comment: pulumi.String(""),
		Name:    pulumi.String(ZONE),
		Tags: pulumi.StringMap{
			"Coder_Managed": pulumi.String("true"),
		},
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}
	return &AWS{
		Vpc:     vpc,
		Public:  pub,
		Private: priv,
		DNS:     zone,
	}, nil
}
