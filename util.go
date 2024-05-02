package main

import (
	"crypto/ed25519"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	ec2x "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"golang.org/x/crypto/ssh"
)

//go:embed assets/aws-policy-coder.json
var awsPolicyCoder string

const (
	keyDir              = "keys"
	ec2AssumeRolePolicy = `{
	"Version": "2012-10-17",
	"Statement": [{
		"Effect": "Allow",
      	"Action": "sts:AssumeRole",
       "Principal": {"Service": "ec2.amazonaws.com"}
    }]}`
)

func newInstance(ctx *pulumi.Context, name string, vpc *ec2x.Vpc, sg *ec2.SecurityGroup, policy, zoneID pulumi.String) (*ec2.Instance, error) {
	conf := config.New(ctx, "")
	prefix := fmt.Sprintf("%s-%s-", ctx.Project(), ctx.Stack())
	zone := conf.Get("dns_zone")
	// every instance gets a key
	pubKey, err := getSSHKey(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("ssh key for %q: %w", name, err)
	}
	kp, err := ec2.NewKeyPair(ctx, name+"-keypair", &ec2.KeyPairArgs{
		KeyName:   pulumi.String(prefix + name),
		PublicKey: pulumi.String(pubKey),
	})

	instanceArgs := &ec2.InstanceArgs{
		SubnetId:                 vpc.PublicSubnetIds.Index(pulumi.Int(0)),
		AssociatePublicIpAddress: pulumi.Bool(true),
		Ami:                      pulumi.String(conf.Get("default_instance_ami")),
		InstanceType:             pulumi.String(conf.Get("default_instance_type")),
		KeyName:                  kp.KeyName,
		VpcSecurityGroupIds:      pulumi.StringArray{sg.ID()},
		MetadataOptions: ec2.InstanceMetadataOptionsArgs{
			HttpPutResponseHopLimit: pulumi.Int(2),
			HttpTokens:              pulumi.String("required"),
		},
	}
	// every instance gets a role
	if policy != "" {
		role, err := iam.NewRole(ctx, name+"-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(ec2AssumeRolePolicy),
			InlinePolicies: iam.RoleInlinePolicyArray{
				&iam.RoleInlinePolicyArgs{
					Name:   pulumi.String(name + "-role-policy"),
					Policy: pulumi.String(policy),
				},
			},
		})
		if err != nil {
			return nil, err
		}
		profile, err := iam.NewInstanceProfile(ctx, name+"-profile", &iam.InstanceProfileArgs{
			Name: pulumi.String(prefix + name),
			Role: role.Name,
		})
		if err != nil {
			return nil, err
		}
		instanceArgs.IamInstanceProfile = profile.Name
	}
	inst, err := ec2.NewInstance(ctx, name, instanceArgs)
	if err != nil {
		return nil, err
	}
	_, err = route53.NewRecord(ctx, name+"-dns", &route53.RecordArgs{
		Name:    pulumi.String(name + "." + zone),
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

func getSSHKey(ctx *pulumi.Context, name string) (string, error) {
	keyDir := filepath.Join(keyDir, ctx.Stack())
	keyPath := filepath.Join(keyDir, name)
	pubBytes, err := os.ReadFile(keyPath + ".pub")
	if err == nil {
		return string(pubBytes), nil
	}
	// generate a new key
	ed25519Pub, ed25519Priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil
	}
	sshPrivateKey, err := ssh.MarshalPrivateKey(ed25519Priv, "")
	if err != nil {
		return "", err
	}
	sshPubKey, err := ssh.NewPublicKey(ed25519Pub)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(keyDir, 0750); err != nil {
		return "", nil
	}
	// write private key file
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return "", err
	}
	defer keyFile.Close()
	if err := pem.Encode(keyFile, sshPrivateKey); err != nil {
		return "", err
	}
	pubBytes = ssh.MarshalAuthorizedKey(sshPubKey)
	// write public key file
	if err := os.WriteFile(keyPath+".pub", pubBytes, 0640); err != nil {
		return "", err
	}
	return string(pubBytes), nil
}
func bucketPolicyDocument(bucket string) pulumi.String {
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
			map[string]any{
				"Effect": "Allow",
				"Action": []any{
					"s3:ListAllMyBuckets",
				},
				"Resource": "*",
			},
		},
	})
	if err != nil {
		panic(fmt.Errorf("generating chaparral task policy: %w", err))
	}
	return pulumi.String(string(doc))
}

func newValidCert(ctx *pulumi.Context, zoneID string, name string) (*acm.Certificate, error) {
	prefix := ctx.Project() + "-" + ctx.Stack() + "-"
	cert, err := acm.NewCertificate(ctx, prefix+"cert", &acm.CertificateArgs{
		DomainName:       pulumi.String(name),
		ValidationMethod: pulumi.String("DNS"),
	})
	domainValidOpt := cert.DomainValidationOptions.Index(pulumi.Int(0))
	domainValidRecord, err := route53.NewRecord(ctx, prefix+"cert-record", &route53.RecordArgs{
		ZoneId: pulumi.String(zoneID),
		Name:   domainValidOpt.ResourceRecordName().Elem(),
		Type:   domainValidOpt.ResourceRecordType().Elem(),
		Records: pulumi.StringArray{
			domainValidOpt.ResourceRecordValue().Elem(),
		},
		Ttl: pulumi.Int(600),
	})
	if err != nil {
		return nil, err
	}
	_, err = acm.NewCertificateValidation(ctx, prefix+"cert-validation", &acm.CertificateValidationArgs{
		CertificateArn:        cert.Arn,
		ValidationRecordFqdns: pulumi.StringArray{domainValidRecord.Fqdn},
	})
	if err != nil {
		return nil, err
	}
	return cert, nil
}
