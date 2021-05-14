package dolocal

import (
	"context"
	"github.com/digitalocean/godo"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

const (
	gitdropsYamlPath = "./gitdrops.yaml"
)

// LocalDropletCreateRequest is a simplified representation of godo.DropletCreateRequest.
// It is only a single level deep to enable unmarshalling from gitdrops.yaml.
type LocalDropletCreateRequest struct {
	Name              string   `yaml:"name"`
	Region            string   `yaml:"region"`
	Size              string   `yaml:"size"`
	Image             string   `yaml:"image"`
	SSHKeyFingerprint string   `yaml:"sshKeyFingerprint"`
	Backups           bool     `yaml:"backups"`
	IPv6              bool     `yaml:"ipv6"`
	PrivateNetworking bool     `yaml:"privateNetworking"`
	Monitoring        bool     `yaml:"monitoring"`
	UserData          string   `yaml:"userData,omitempty"`
	Volumes           []string `yaml:"volumes,omitempty"`
	Tags              []string `yaml:"tags"`
	VPCUUID           string   `yaml:"vpcuuid,omitempty"`
}

// ReadLocalDropletCreateRequests reads and unmarshals from gitops.yaml
func ReadLocalDropletCreateRequests() ([]LocalDropletCreateRequest, error) {
	gitdropsYaml, err := ioutil.ReadFile(gitdropsYamlPath)
	if err != nil {
		return nil, err
	}

	var localDropletCreateRequests []LocalDropletCreateRequest
	err = yaml.Unmarshal(gitdropsYaml, &localDropletCreateRequests)
	if err != nil {
		return nil, err
	}

	return localDropletCreateRequests, nil
}

// ListDroplets lists all active droplets on DO account
func ListDroplets(ctx context.Context, client *godo.Client) ([]godo.Droplet, error) {
	// create a list to hold our droplets
	list := []godo.Droplet{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		// append the current page's droplets to our list
		list = append(list, droplets...)

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	return list, nil
}
