package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vpc, err := awsVPC(ctx)
		if err != nil {
			return err
		}

		// tag the first public subnet in the vpc for use by Coder
		_, err = ec2.NewTag(ctx, "coderSubnetTag", &ec2.TagArgs{
			ResourceId: vpc.PrivateSubnetIds.Index(pulumi.Int(0)),
			Key:        pulumi.String("Coder_Workspaces"),
			Value:      pulumi.String("true"),
		})
		if err != nil {
			return err
		}

		// create an coder instance
		coder, err := awsCoderVM(ctx, vpc)
		if err != nil {
			return err
		}

		ctx.Export("vpc-id", vpc.VpcId)
		ctx.Export("coder-ip", coder.PublicIp)
		return nil
	})
}
