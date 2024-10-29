package main

import (
	"dreamlab/internal/dreamlab"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	butaneConfig "github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"github.com/hashicorp/go-multierror"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ebs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
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

func awsCoderVM(ctx *pulumi.Context, vpc *dreamlab.AWS) error {
	conf := config.New(ctx, "")
	var (
		hostname     = `coder-test`
		ami          = conf.Get("coder_instance_ami")
		instanceType = conf.Get("coder_instance_type")
	)
	sgName := hostname + "-sg"
	sg, err := ec2.NewSecurityGroup(ctx, sgName, &ec2.SecurityGroupArgs{
		Name:  pulumi.String(sgName),
		VpcId: vpc.Vpc.ID(),
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
			&ec2.SecurityGroupIngressArgs{
				FromPort:       pulumi.Int(80),
				ToPort:         pulumi.Int(80),
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
		return err
	}

	pubKey, err := getSSHKey(ctx, hostname)
	if err != nil {
		return fmt.Errorf("ssh key for %q: %w", hostname, err)
	}
	keypairName := hostname + "-vm-keypair"
	kp, err := ec2.NewKeyPair(ctx, keypairName, &ec2.KeyPairArgs{
		KeyName:   pulumi.String(keypairName),
		PublicKey: pulumi.String(pubKey),
	})
	if err != nil {
		return err
	}

	userData, err := ignition(ctx)
	if err != nil {
		return err
	}

	roleName := hostname + "-vm-role"
	role, err := iam.NewRole(ctx, roleName, &iam.RoleArgs{
		Name:             pulumi.String(roleName),
		AssumeRolePolicy: pulumi.String(policyEC2AssumeRole),
	})
	if err != nil {
		return err
	}
	rolePolicyName := roleName + "-policy"
	_, err = iam.NewRolePolicy(ctx, rolePolicyName, &iam.RolePolicyArgs{
		Name:   pulumi.String(rolePolicyName),
		Role:   role.Name,
		Policy: pulumi.String(awsPolicyCoder),
	})
	if err != nil {
		return err
	}

	profileName := hostname + "-vm-profile"
	profile, err := iam.NewInstanceProfile(ctx, profileName, &iam.InstanceProfileArgs{
		Name: pulumi.String(profileName),
		Role: role.Name,
	})
	if err != nil {
		return err
	}
	varVolName := hostname + "-var"
	varVol, err := ebs.NewVolume(ctx, varVolName, &ebs.VolumeArgs{
		AvailabilityZone: vpc.Public.AvailabilityZone,
		Size:             pulumi.IntPtr(64),
		Type:             pulumi.StringPtr("gp3"),
	})
	instanceArgs := &ec2.InstanceArgs{
		IamInstanceProfile:  profile.Name,
		SubnetId:            vpc.Public.ID(),
		Ami:                 pulumi.String(ami),
		InstanceType:        pulumi.String(instanceType),
		KeyName:             kp.KeyName,
		VpcSecurityGroupIds: pulumi.StringArray{sg.ID()},
		MetadataOptions: ec2.InstanceMetadataOptionsArgs{
			HttpPutResponseHopLimit: pulumi.Int(2),
			HttpTokens:              pulumi.String("required"),
		},
		UserData:                userData,
		UserDataReplaceOnChange: pulumi.Bool(true),
	}
	inst, err := ec2.NewInstance(ctx, hostname, instanceArgs, pulumi.DeleteBeforeReplace(true))
	if err != nil {
		return err
	}
	// Attach the volume to the existing EC2 instance.
	_, err = ec2.NewVolumeAttachment(ctx, varVolName+"-attach", &ec2.VolumeAttachmentArgs{
		InstanceId:                  inst.ID(),
		VolumeId:                    varVol.ID(),
		DeviceName:                  pulumi.String("/dev/sdf"),
		StopInstanceBeforeDetaching: pulumi.BoolPtr(true),
	}, pulumi.DeleteBeforeReplace(true))
	if err != nil {
		return err
	}
	eiPName := hostname + "-eip"
	eip, err := ec2.NewEip(ctx, eiPName, &ec2.EipArgs{
		Domain:   pulumi.String("vpc"),
		Instance: inst.ID(),
	})
	if err != nil {
		return err
	}

	recordName := hostname + "-dns"
	_, err = route53.NewRecord(ctx, recordName, &route53.RecordArgs{
		Name:    pulumi.String(hostname + "." + dreamlab.ZONE),
		ZoneId:  vpc.DNS.ZoneId,
		Type:    pulumi.String("A"),
		Records: pulumi.StringArray{eip.PublicIp},
		Ttl:     pulumi.Int(600),
	})
	if err != nil {
		return err
	}

	wildcardRecordName := hostname + "-wildcard-dns"
	_, err = route53.NewRecord(ctx, wildcardRecordName, &route53.RecordArgs{
		Name:    pulumi.String("*." + hostname + "." + dreamlab.ZONE),
		ZoneId:  vpc.DNS.ZoneId,
		Type:    pulumi.String("A"),
		Records: pulumi.StringArray{eip.PublicIp},
		Ttl:     pulumi.Int(600),
	})
	if err != nil {
		return err
	}
	ctx.Export(hostname+"-publicIP", eip.PublicIp)
	return nil
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

func ignition(ctx *pulumi.Context) (pulumi.StringOutput, error) {
	var out pulumi.StringOutput
	wd, err := os.Getwd()
	if err != nil {
		return out, err
	}
	raw, err := os.ReadFile(filepath.Join(wd, "coder", "butane.yml"))
	if err != nil {
		return out, err
	}
	cfg := config.New(ctx, "")

	out = pulumi.All(
		cfg.GetSecret("googleOAuth2ClientID"),
		cfg.GetSecret("googleOAuth2ClientSecret"),
	).ApplyT(func(args []interface{}) (string, error) {
		vals := struct {
			OIDCClientID     string
			OIDCClientSecret string
		}{
			OIDCClientID:     args[0].(string),
			OIDCClientSecret: args[1].(string),
		}
		tpl, err := template.New("butane").Parse(string(raw))
		if err != nil {
			return "", err
		}
		builder := &strings.Builder{}
		if err := tpl.Execute(builder, vals); err != nil {
			return "", err
		}

		opts := common.TranslateBytesOptions{
			TranslateOptions: common.TranslateOptions{
				FilesDir: filepath.Join(wd, "coder"),
			},
		}
		ign, report, err := butaneConfig.TranslateBytes([]byte(builder.String()), opts)
		if report.IsFatal() {
			err := &multierror.Error{}
			for _, e := range report.Entries {
				err = multierror.Append(err, fmt.Errorf("%s: %s", e.Kind, e.Message))
			}
			return "", err
		}
		return string(ign), nil
	}).(pulumi.StringOutput)
	return out, nil
}
