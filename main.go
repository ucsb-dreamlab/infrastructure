package main

import (
	"dreamlab/internal/dreamlab"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {

	pulumi.Run(func(ctx *pulumi.Context) error {
		dreamAWS, err := dreamlab.NewAWS(ctx)
		if err != nil {
			return err
		}
		if err := awsCoderVM(ctx, dreamAWS); err != nil {
			return err
		}
		return nil
	})
}
