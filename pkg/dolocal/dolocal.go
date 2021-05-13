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
	Name              string   `json:"name"`
	Region            string   `json:"region"`
	Size              string   `json:"size"`
	Image             string   `json:"image"`
	SSHKeys           []string `json:"ssh_keys"`
	Backups           bool     `json:"backups"`
	IPv6              bool     `json:"ipv6"`
	PrivateNetworking bool     `json:"private_networking"`
	Monitoring        bool     `json:"monitoring"`
	UserData          string   `json:"user_data,omitempty"`
	Volumes           []string `json:"volumes,omitempty"`
	Tags              []string `json:"tags"`
	VPCUUID           string   `json:"vpc_uuid,omitempty"`
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
