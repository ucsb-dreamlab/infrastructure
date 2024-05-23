package main

import (
	_ "embed"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	ec2x "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const policyDocEC2AssumeRole = `{
	"Version": "2012-10-17",
	"Statement": [{
		"Effect": "Allow",
      	"Action": "sts:AssumeRole",
       "Principal": {"Service": "ec2.amazonaws.com"}
    }]}`

//go:embed coder/policy.json
var awsPolicyCoder string

func awsVPC(ctx *pulumi.Context) (*ec2x.Vpc, error) {
	vpc, err := ec2x.NewVpc(ctx, "dreamlab-vpc", &ec2x.VpcArgs{
		NumberOfAvailabilityZones: pulumi.IntRef(2),
		NatGateways: &ec2x.NatGatewayConfigurationArgs{
			Strategy: ec2x.NatGatewayStrategySingle,
		}},
	)
	if err != nil {
		return nil, err
	}

	// the VPC's default security group allows inbound tcp port 22, 443
	_, err = ec2.NewDefaultSecurityGroup(ctx, "default-ssh-tls", &ec2.DefaultSecurityGroupArgs{
		VpcId: vpc.VpcId,
		Ingress: &ec2.DefaultSecurityGroupIngressArray{
			&ec2.DefaultSecurityGroupIngressArgs{
				FromPort:       pulumi.Int(22),
				ToPort:         pulumi.Int(22),
				Protocol:       pulumi.String("tcp"),
				CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			},
			&ec2.DefaultSecurityGroupIngressArgs{
				FromPort:       pulumi.Int(443),
				ToPort:         pulumi.Int(443),
				Protocol:       pulumi.String("tcp"),
				CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			},
		},
		Egress: &ec2.DefaultSecurityGroupEgressArray{
			&ec2.DefaultSecurityGroupEgressArgs{
				FromPort:       pulumi.Int(0),
				ToPort:         pulumi.Int(0),
				Protocol:       pulumi.String("-1"),
				CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return vpc, nil
}

func awsCoderVM(ctx *pulumi.Context, vpc *ec2x.Vpc) (*ec2.Instance, error) {
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
		SubnetId:                 vpc.PublicSubnetIds.Index(pulumi.Int(0)),
		AssociatePublicIpAddress: pulumi.Bool(true),
		Ami:                      pulumi.String(ami),
		InstanceType:             pulumi.String(instanceType),
		KeyName:                  kp.KeyName,
		VpcSecurityGroupIds:      pulumi.StringArray{vpc.Vpc.DefaultSecurityGroupId()},
		MetadataOptions: ec2.InstanceMetadataOptionsArgs{
			HttpPutResponseHopLimit: pulumi.Int(2),
			HttpTokens:              pulumi.String("required"),
		},
	}
	// every instance gets a role
	roleName := hostname + "-vm-role"
	role, err := iam.NewRole(ctx, roleName, &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(policyDocEC2AssumeRole),
		InlinePolicies: iam.RoleInlinePolicyArray{
			&iam.RoleInlinePolicyArgs{
				Name:   pulumi.String(roleName),
				Policy: pulumi.String(awsPolicyCoder),
			},
		},
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
