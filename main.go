package main

import (
	"dreamlab/coder"
	"dreamlab/internal/dreamlab"
	"dreamlab/ocfl"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vpc, err := dreamlab.NewAWSVPC(ctx)
		if err != nil {
			return err
		}
		dns, err := dreamlab.NewDNSZone(ctx)
		if err != nil {
			return err
		}
		stackConfig := config.New(ctx, "")
		// coder.dreamlab.ucsb.edu
		if err := coder.New(ctx, "coder", &coder.Config{
			Hostname:     "coder",
			VPC:          vpc,
			DNS:          dns,
			InstanceAMI:  stackConfig.Get("coder_instance_ami"),
			InstanceType: stackConfig.Get("coder_instance_type"),
		}); err != nil {
			return err
		}

		// data.dreamlab.ucsb.edu runs ocfl-server
		if err := ocfl.New(ctx, "data", &ocfl.Config{
			Hostname:     "data",
			VPC:          vpc,
			DNS:          dns,
			InstanceAMI:  stackConfig.Get("coder_instance_ami"),
			InstanceType: stackConfig.Get("coder_instance_type"),
		}); err != nil {
			return err
		}

		return nil
	})
}
