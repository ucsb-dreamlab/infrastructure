package main

import (
	_ "embed"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	dreamlab = "dreamlab"

	vpc     = "dreamlab_vpc"
	vpcCidr = "10.226.42.192/26"

	pubAZ      = "us-west-2a"
	pubSubnet  = "public_subnet"
	pubCidr    = "10.226.42.192/27"
	pubTagName = "dreamlab Public Subnet 1"

	privSubnet  = "private_subnet"
	privCidr    = "10.226.42.224/27"
	privAZ      = "us-west-2a"
	privTagName = "dreamlab Private Subnet 1"

	policyEC2AssumeRole = `{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			  "Action": "sts:AssumeRole",
		   "Principal": {"Service": "ec2.amazonaws.com"}
		}]}`
)

//go:embed coder/policy.json
var awsPolicyCoder string

func awsVPC(ctx *pulumi.Context) error {

	// imported vpc
	vpc, err := ec2.NewVpc(ctx, vpc, &ec2.VpcArgs{
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
		return err
	}

	_, err = ec2.NewSubnet(ctx, pubSubnet, &ec2.SubnetArgs{
		AvailabilityZone:               pulumi.String(pubAZ),
		CidrBlock:                      pulumi.String(pubCidr),
		MapPublicIpOnLaunch:            pulumi.Bool(true),
		PrivateDnsHostnameTypeOnLaunch: pulumi.String("ip-name"),
		Tags: pulumi.StringMap{
			"Name":         pulumi.String(pubTagName),
			"Network":      pulumi.String("Public"),
			"ucsb:service": pulumi.String("UCSB Campus Cloud Portfolio"),
		},
		VpcId: vpc.ID(),
	}, pulumi.Protect(true))
	if err != nil {
		return err
	}

	_, err = ec2.NewSubnet(ctx, privSubnet, &ec2.SubnetArgs{
		AvailabilityZone:               pulumi.String(privAZ),
		CidrBlock:                      pulumi.String(privCidr),
		PrivateDnsHostnameTypeOnLaunch: pulumi.String("ip-name"),
		Tags: pulumi.StringMap{
			"Name":                   pulumi.String(privTagName),
			"Network":                pulumi.String("Private"),
			"ucsb:service":           pulumi.String("UCSB Campus Cloud Portfolio"),
			"dreamlab:service:coder": pulumi.String("workers"),
		},
		VpcId: vpc.ID(),
	}, pulumi.Protect(true))
	if err != nil {
		return err
	}
	return nil
}

func awsCoderVM(ctx *pulumi.Context) (*ec2.Instance, error) {
	conf := config.New(ctx, "")
	var (
		hostname     = `coder`
		ami          = conf.Get("coder_instance_ami")
		instanceType = conf.Get("coder_instance_type")
	)
	zoneID, zoneName, err := awsDNSZone(ctx)
	if err != nil {
		return nil, err
	}

	pubSubnet, err := ec2.LookupSubnet(ctx, &ec2.LookupSubnetArgs{
		Tags: map[string]string{"Name": pubTagName},
	})
	if err != nil {
		return nil, err
	}
	vpc, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{
		Tags: map[string]string{"Name": dreamlab},
	})
	if err != nil {
		return nil, err
	}

	sgName := hostname + "-sg"
	sg, err := ec2.NewSecurityGroup(ctx, sgName, &ec2.SecurityGroupArgs{
		Name:  pulumi.String(sgName),
		VpcId: pulumi.String(vpc.Id),
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
		}})
	if err != nil {
		return nil, err
	}

	pubKey, err := getSSHKey(ctx, hostname)
	if err != nil {
		return nil, fmt.Errorf("ssh key for %q: %w", hostname, err)
	}
	keypairName := hostname + "-vm-keypair"
	kp, err := ec2.NewKeyPair(ctx, keypairName, &ec2.KeyPairArgs{
		KeyName:   pulumi.String(keypairName),
		PublicKey: pulumi.String(pubKey),
	})
	if err != nil {
		return nil, err
	}

	instanceArgs := &ec2.InstanceArgs{
		SubnetId:                 pulumi.String(pubSubnet.Id),
		AssociatePublicIpAddress: pulumi.Bool(true),
		Ami:                      pulumi.String(ami),
		InstanceType:             pulumi.String(instanceType),
		KeyName:                  kp.KeyName,
		VpcSecurityGroupIds:      pulumi.StringArray{sg.ID()},
		MetadataOptions: ec2.InstanceMetadataOptionsArgs{
			HttpPutResponseHopLimit: pulumi.Int(2),
			HttpTokens:              pulumi.String("required"),
		},
		//UserData: ,
	}
	// every instance gets a role
	roleName := hostname + "-vm-role"
	role, err := iam.NewRole(ctx, roleName, &iam.RoleArgs{
		Name:             pulumi.String(roleName),
		AssumeRolePolicy: pulumi.String(policyEC2AssumeRole),
	})
	if err != nil {
		return nil, err
	}
	rolePolicyName := roleName + "-policy"
	_, err = iam.NewRolePolicy(ctx, rolePolicyName, &iam.RolePolicyArgs{
		Name:   pulumi.String(rolePolicyName),
		Role:   role.Name,
		Policy: pulumi.String(awsPolicyCoder),
	})
	if err != nil {
		return nil, err
	}

	profileName := hostname + "-vm-profile"
	profile, err := iam.NewInstanceProfile(ctx, profileName, &iam.InstanceProfileArgs{
		Name: pulumi.String(profileName),
		Role: role.Name,
	})
	if err != nil {
		return nil, err
	}
	instanceArgs.IamInstanceProfile = profile.Name
	inst, err := ec2.NewInstance(ctx, hostname, instanceArgs)
	if err != nil {
		return nil, err
	}
	recordName := hostname + "-dns"
	_, err = route53.NewRecord(ctx, recordName, &route53.RecordArgs{
		Name:    pulumi.String(hostname + "." + zoneName),
		ZoneId:  pulumi.String(zoneID),
		Type:    pulumi.String("A"),
		Records: pulumi.StringArray{inst.PublicIp},
		Ttl:     pulumi.Int(600),
	})
	if err != nil {
		return nil, err
	}

	wildcardRecordName := hostname + "-wildcard-dns"
	_, err = route53.NewRecord(ctx, wildcardRecordName, &route53.RecordArgs{
		Name:    pulumi.String("*." + hostname + "." + zoneName),
		ZoneId:  pulumi.String(zoneID),
		Type:    pulumi.String("A"),
		Records: pulumi.StringArray{inst.PublicIp},
		Ttl:     pulumi.Int(600),
	})
	if err != nil {
		return nil, err
	}
	return inst, nil
}

func awsDNSZone(ctx *pulumi.Context) (id string, name string, err error) {
	conf := config.New(ctx, "")
	zone := conf.Get("dns_zone")
	if zone == "" {
		err = fmt.Errorf("missing dns_zone config")
		return
	}
	lookup, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{Name: &zone})
	if err != nil {
		return
	}
	id = lookup.Id
	name = lookup.Name
	return
}

// func bucketPolicyDocument(bucket string) pulumi.String {
// 	doc, err := json.Marshal(map[string]any{
// 		"Version": "2012-10-17",
// 		"Statement": []any{
// 			map[string]any{
// 				"Effect": "Allow",
// 				"Action": []any{
// 					"s3:GetBucketLocation",
// 					"s3:ListBucket",
// 				},
// 				"Resource": "arn:aws:s3:::" + bucket,
// 			},
// 			map[string]any{
// 				"Effect": "Allow",
// 				"Action": []any{
// 					"s3:PutObject",
// 					"s3:DeleteObject",
// 					"s3:GetObject",
// 				},
// 				"Resource": "arn:aws:s3:::" + bucket + "/*",
// 			},
// 			map[string]any{
// 				"Effect": "Allow",
// 				"Action": []any{
// 					"s3:ListAllMyBuckets",
// 				},
// 				"Resource": "*",
// 			},
// 		},
// 	})
// 	if err != nil {
// 		panic(fmt.Errorf("generating chaparral task policy: %w", err))
// 	}
// 	return pulumi.String(string(doc))
// }
