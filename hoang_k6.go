package k6

import (
	"github.com/docker/docker/client"
	"go.k6.io/k6/js/modules"
)

type (
	Containers struct {
		Client *client.Client
	}
	Volumes struct {
		Client *client.Client
	}
	Networks struct {
		Client *client.Client
	}
)

func (d *Containers) SetupClient() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	d.Client = cli
}

func (v *Volumes) SetupClient() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	v.Client = cli
}

func (nw *Networks) SetupClient() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	nw.Client = cli
}

func init() {
	modules.Register("k6/x/docker", &Docker{})

	containers := Containers{}
	containers.SetupClient()
	modules.Register("k6/x/docker/containers", &containers)

	volumes := Volumes{}
	volumes.SetupClient()
	modules.Register("k6/x/docker/volumes", &volumes)

	networks := Networks{}
	networks.SetupClient()
	modules.Register("k6/x/docker/networks", &networks)

	images := Images{}
	images.SetupClient()
	modules.Register("k6/x/docker/images", &images)
}

// Docker is the main export of k6 docker extension
type Docker struct{}
