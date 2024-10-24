package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {

	pulumi.Run(func(ctx *pulumi.Context) error {

		if err := awsVPC(ctx); err != nil {
			return err
		}

		// // create an coder instance
		if err := awsCoderVM(ctx); err != nil {
			return err
		}

		// ctx.Export("vpc-id", vpc.VpcId)
		// ctx.Export("coder-ip", inst.PublicIp)
		return nil
	})
}
