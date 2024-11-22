package dreamlab

import (
	"github.com/pulumi/pulumi-openstack/sdk/v4/go/openstack/networking"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const js2PublicNetID = pulumi.String("3fe22c05-6206-4db2-9a13-44f04b6796e6")

type Jetstream2 struct {
	Network *networking.Network
}

func NewJetstream2(ctx *pulumi.Context) (*Jetstream2, error) {
	js2 := &Jetstream2{}
	var err error

	js2.Network, err = networking.NewNetwork(ctx, "dreamlab", &networking.NetworkArgs{
		Name:         pulumi.String("dreamlab_network"),
		AdminStateUp: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	subn, err := networking.NewSubnet(ctx, "subnet", &networking.SubnetArgs{
		NetworkId: js2.Network.ID(),
		Cidr:      pulumi.String("192.168.159.0/24"),
	})
	if err != nil {
		return nil, err
	}

	router, err := networking.NewRouter(ctx, "router", &networking.RouterArgs{
		Name:              pulumi.String("dreamlab_router"),
		AdminStateUp:      pulumi.Bool(true),
		ExternalNetworkId: js2PublicNetID,
	})
	if err != nil {
		return nil, err
	}

	_, err = networking.NewRouterInterface(ctx, "interface01", &networking.RouterInterfaceArgs{
		RouterId: router.ID(),
		SubnetId: subn.ID(),
	})
	if err != nil {
		return nil, err
	}
	return js2, nil
}
