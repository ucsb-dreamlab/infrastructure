package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	ec2x "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	ecrx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	lbx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vpc, err := ec2x.NewVpc(ctx, "dreamlab-vpc", nil)
		if err != nil {
			return err
		}

		// An ECS cluster to deploy into
		cluster, err := ecs.NewCluster(ctx, "dreamlab-cluster", nil)
		if err != nil {
			return err
		}

		// An ALB to serve the container endpoint to the internet
		loadbalancer, err := lbx.NewApplicationLoadBalancer(ctx, "dreamlab-lb", &lbx.ApplicationLoadBalancerArgs{
			SubnetIds: vpc.PublicSubnetIds,
		})
		if err != nil {
			return err
		}

		// An ECR repository to store our application's container image
		repo, err := ecrx.NewRepository(ctx, "dreamlab-repo", &ecrx.RepositoryArgs{
			ForceDelete: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		chaparral(ctx, vpc, repo, cluster, loadbalancer)
		// The URL at which the container's HTTP endpoint will be available
		ctx.Export("url", pulumi.Sprintf("http://%s", loadbalancer.LoadBalancer.DnsName()))
		return nil
	})
}
