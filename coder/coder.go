package coder

import (
	"dreamlab/internal/dreamlab"
	_ "embed"
	"fmt"
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

const policyEC2AssumeRole = `{
	"Version": "2012-10-17",
	"Statement": [{
		"Effect": "Allow",
		"Action": "sts:AssumeRole",
		"Principal": {"Service": "ec2.amazonaws.com"}
	}]
}`

//go:embed policy.json
var awsPolicyCoder string

//go:embed butane.yml
var butaneYML string

type Config struct {
	VPC          *dreamlab.AWSVPC
	DNS          *dreamlab.DNS
	Hostname     string
	InstanceAMI  string // should be fedora coreos
	InstanceType string // shoube be arm64
}

func New(ctx *pulumi.Context, resource string, coderConfig *Config) error {
	sgResource := resource + "-sg"
	sg, err := ec2.NewSecurityGroup(ctx, sgResource, &ec2.SecurityGroupArgs{
		Name:  pulumi.String(sgResource),
		VpcId: coderConfig.VPC.Vpc.ID(),
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
	pubKey, err := dreamlab.GetCreateSSHKey(filepath.Join("keys", ctx.Stack()), coderConfig.Hostname)
	if err != nil {
		return fmt.Errorf("ssh key for %q: %w", coderConfig.Hostname, err)
	}
	keypairResource := resource + "-vm-keypair"
	kp, err := ec2.NewKeyPair(ctx, keypairResource, &ec2.KeyPairArgs{
		KeyName:   pulumi.String(keypairResource),
		PublicKey: pulumi.String(pubKey),
	})
	if err != nil {
		return err
	}
	// create an instance profile for the vm
	roleResource := resource + "-role"
	role, err := iam.NewRole(ctx, roleResource, &iam.RoleArgs{
		Name:             pulumi.String(roleResource),
		AssumeRolePolicy: pulumi.String(policyEC2AssumeRole),
	})
	if err != nil {
		return err
	}
	policyResource := roleResource + "-policy"
	_, err = iam.NewRolePolicy(ctx, policyResource, &iam.RolePolicyArgs{
		Role:   role.Name,
		Policy: pulumi.String(awsPolicyCoder),
	})
	if err != nil {
		return err
	}
	profileResource := resource + "-profile"
	profile, err := iam.NewInstanceProfile(ctx, profileResource, &iam.InstanceProfileArgs{
		Role: role.Name,
	})
	if err != nil {
		return err
	}
	// persistent storage
	varVolRescource := resource + "-var"
	varVol, err := ebs.NewVolume(ctx, varVolRescource, &ebs.VolumeArgs{
		AvailabilityZone: coderConfig.VPC.Public.AvailabilityZone,
		Size:             pulumi.IntPtr(64),
		Type:             pulumi.StringPtr("gp3"),
	})
	if err != nil {
		return err
	}
	userData, err := ignition(ctx, coderConfig)
	if err != nil {
		return err
	}
	instanceArgs := &ec2.InstanceArgs{
		IamInstanceProfile:  profile.Name,
		SubnetId:            coderConfig.VPC.Public.ID(),
		Ami:                 pulumi.String(coderConfig.InstanceAMI),
		InstanceType:        pulumi.String(coderConfig.InstanceType),
		KeyName:             kp.KeyName,
		VpcSecurityGroupIds: pulumi.StringArray{sg.ID()},
		MetadataOptions: ec2.InstanceMetadataOptionsArgs{
			HttpPutResponseHopLimit: pulumi.Int(2),
			HttpTokens:              pulumi.String("required"),
		},
		UserData:                userData,
		UserDataReplaceOnChange: pulumi.Bool(true),
	}
	inst, err := ec2.NewInstance(ctx, resource, instanceArgs, pulumi.DeleteBeforeReplace(true))
	if err != nil {
		return err
	}
	// Attach the volume to the existing EC2 instance.
	_, err = ec2.NewVolumeAttachment(ctx, varVolRescource+"-attach", &ec2.VolumeAttachmentArgs{
		InstanceId:                  inst.ID(),
		VolumeId:                    varVol.ID(),
		DeviceName:                  pulumi.String("/dev/sdf"),
		StopInstanceBeforeDetaching: pulumi.BoolPtr(true),
	}, pulumi.DeleteBeforeReplace(true))
	if err != nil {
		return err
	}
	eiPResource := resource + "-eip"
	eip, err := ec2.NewEip(ctx, eiPResource, &ec2.EipArgs{
		Domain:   pulumi.String("vpc"),
		Instance: inst.ID(),
	})
	if err != nil {
		return err
	}
	recordResource := resource + "-dns"
	_, err = route53.NewRecord(ctx, recordResource, &route53.RecordArgs{
		Name:    pulumi.String(coderConfig.Hostname + "." + coderConfig.DNS.Domain()),
		ZoneId:  coderConfig.DNS.ZoneId,
		Type:    pulumi.String("A"),
		Records: pulumi.StringArray{eip.PublicIp},
		Ttl:     pulumi.Int(600),
	})
	if err != nil {
		return err
	}

	wildcardRecordResource := resource + "-wildcard-dns"
	_, err = route53.NewRecord(ctx, wildcardRecordResource, &route53.RecordArgs{
		Name:    pulumi.String("*." + coderConfig.Hostname + "." + coderConfig.DNS.Domain()),
		ZoneId:  coderConfig.DNS.ZoneId,
		Type:    pulumi.String("A"),
		Records: pulumi.StringArray{eip.PublicIp},
		Ttl:     pulumi.Int(600),
	})
	if err != nil {
		return err
	}

	recordPrivateResource := resource + "-private-dns"
	_, err = route53.NewRecord(ctx, recordPrivateResource, &route53.RecordArgs{
		Name:    pulumi.String(coderConfig.Hostname + "-private." + coderConfig.DNS.Domain()),
		ZoneId:  coderConfig.DNS.ZoneId,
		Type:    pulumi.String("A"),
		Records: pulumi.StringArray{eip.PublicIp},
		Ttl:     pulumi.Int(600),
	})
	if err != nil {
		return err
	}

	wildcardRecordPrivateResource := resource + "-wildcard-private-dns"
	_, err = route53.NewRecord(ctx, wildcardRecordPrivateResource, &route53.RecordArgs{
		Name:    pulumi.String("*." + coderConfig.Hostname + "-private." + coderConfig.DNS.Domain()),
		ZoneId:  coderConfig.DNS.ZoneId,
		Type:    pulumi.String("A"),
		Records: pulumi.StringArray{eip.PublicIp},
		Ttl:     pulumi.Int(600),
	})
	if err != nil {
		return err
	}

	ctx.Export(coderConfig.Hostname+"-publicIP", eip.PublicIp)
	return nil
}

// build fedora coreos ignition user data for the machine.
func ignition(ctx *pulumi.Context, coderConfig *Config) (pulumi.StringOutput, error) {
	var out pulumi.StringOutput
	cfg := config.New(ctx, "")
	out = pulumi.All(
		cfg.GetSecret("googleOAuth2ClientID"),
		cfg.GetSecret("googleOAuth2ClientSecret"),
		cfg.Get("LSITClusterServer"),
		cfg.GetSecret("LSITClusterToken"),
		cfg.GetSecret("LSITOuterRimToken"),
	).ApplyT(func(args []interface{}) (string, error) {
		vals := struct {
			OIDCClientID      string
			OIDCClientSecret  string
			LSITClusterServer string
			LSITClusterToken  string
			LSITOuterRimToken string
			Hostname          string
			Domain            string
		}{
			OIDCClientID:      args[0].(string),
			OIDCClientSecret:  args[1].(string),
			LSITClusterServer: args[2].(string),
			LSITClusterToken:  args[3].(string),
			LSITOuterRimToken: args[4].(string),
			Hostname:          coderConfig.Hostname,
			Domain:            coderConfig.DNS.Domain(),
		}
		myFuncs := template.FuncMap{
			"domainEscape": func(d string) string {
				return strings.ReplaceAll(d, `.`, `\\.`)
			},
		}
		tpl, err := template.New("butane").Funcs(myFuncs).Parse(string(butaneYML))
		if err != nil {
			return "", err
		}
		builder := &strings.Builder{}
		if err := tpl.Execute(builder, vals); err != nil {
			return "", err
		}
		opts := common.TranslateBytesOptions{
			TranslateOptions: common.TranslateOptions{
				FilesDir: "coder",
			},
		}
		ign, report, err := butaneConfig.TranslateBytes([]byte(builder.String()), opts)
		if err != nil {
			return "", err
		}
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
