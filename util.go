package k6

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/docker/docker/client"
	"github.com/dop251/goja"
	"github.com/goharbor/xk6-harbor/pkg/util"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

type (
	Option struct {
		Scheme   string // http or https
		Host     string
		Username string
		Password string
		Insecure bool // Allow insecure server connections when using SSL
	}

	Images struct {
		vu          modules.VU
		option      *Option
		initialized bool
		Client      *client.Client
	}
)

func (d *Images) SetupClient() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	d.Client = cli
}

type PushOption struct {
	Ref   string
	Store *ContentStore
	Blobs []ocispec.Descriptor
}

func (d *Images) Push(option PushOption, args ...goja.Value) string {
	d.mustInitialized()

	resolver := d.makeResolver(args...)
	ref := d.getRef(option.Ref)

	// this config makes the harbor identify the artifact as image
	configBytes, _ := json.Marshal(map[string]interface{}{"User": uuid.New().String()})
	config := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    digest.FromBytes(configBytes),
		Size:      int64(len(configBytes)),
	}

	_, err := writeBlob(option.Store.RootPath, configBytes)
	Checkf(d.vu.Runtime(), err, "faied to prepare the config for the %s", ref)

	manifest, err := oras.Push(d.vu.Context(), resolver, ref, option.Store.Store, option.Blobs, oras.WithConfig(config))
	Checkf(d.vu.Runtime(), err, "failed to push %s", ref)

	return manifest.Digest.String()
}

func (d *Images) mustInitialized() {
	if !d.initialized {
		common.Throw(d.vu.Runtime(), errors.New("harbor module not initialized"))
	}
}

func (d *Images) getRef(ref string) string {
	if !strings.HasPrefix(ref, d.option.Host) {
		return d.option.Host + "/" + ref
	}

	return ref
}

func writeBlob(rootPath string, data []byte) (digest.Digest, error) {
	dgt := digest.FromBytes(data)

	dir := path.Join(rootPath, "blobs", dgt.Algorithm().String())

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	filename := path.Join(dir, dgt.Hex())
	if err := os.WriteFile(filename, data, 0664); err != nil {
		return "", err
	}

	return dgt, nil
}

func Checkf(rt *goja.Runtime, err error, format string, a ...interface{}) {
	if err == nil {
		return
	}

	common.Throw(
		rt,
		fmt.Errorf("%s, error: %s", fmt.Sprintf(format, a...), err),
	)
}

func (d *Images) makeResolver(args ...goja.Value) remotes.Resolver {
	d.mustInitialized()

	log.StandardLogger().SetLevel(log.ErrorLevel)

	var transport http.RoundTripper
	if d.option.Insecure {
		transport = util.NewInsecureTransport()
	} else {
		transport = util.NewDefaultTransport()
	}

	client := &http.Client{Transport: transport}

	authorizer := docker.NewAuthorizer(client, func(host string) (string, string, error) {
		if host == d.option.Host {
			return d.option.Username, d.option.Password, nil
		}

		return "", "", nil
	})

	plainHTTP := func(host string) (bool, error) {
		if host == d.option.Host {
			return d.option.Scheme == "http", nil
		}

		return false, nil // default is https
	}

	return docker.NewResolver(docker.ResolverOptions{
		Hosts: docker.ConfigureDefaultRegistries(
			docker.WithAuthorizer(authorizer),
			docker.WithClient(client),
			docker.WithPlainHTTP(plainHTTP),
		),
	})
}
