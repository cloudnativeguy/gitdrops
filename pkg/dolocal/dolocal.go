package dolocal

import (
	"context"
	"errors"
	"io/ioutil"
	"log"

	"github.com/digitalocean/godo"

	"gopkg.in/yaml.v2"
)

const (
	gitdropsYamlPath = "./gitdrops.yaml"
	retries          = 10
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
		droplets := []godo.Droplet{}
		resp := &godo.Response{}
		err := errors.New("")
		for i := 0; i < retries; i++ {
			droplets, resp, err = client.Droplets.List(ctx, opt)
			if err != nil {
				log.Println("error listing droplets", err)
				if i == retries-1 {
					return list, err
				}
			} else {
				break
			}
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

func DeleteDroplet(ctx context.Context, client *godo.Client, id int) error {
	for i := 0; i < retries; i++ {
		response, err := client.Droplets.Delete(ctx, id)
		if err != nil {
			log.Println("error during delete request for droplet ", id, " error: ", err)
			if i == retries-1 {
				return err
			}
		} else {
			log.Println("delete request for droplet ", id, " returned: ", response.StatusCode)
			break
		}
	}
	return nil
}

func CreateDroplet(ctx context.Context, client *godo.Client, dropletCreateRequest *godo.DropletCreateRequest) error {
	for i := 0; i < retries; i++ {
		_, response, err := client.Droplets.Create(ctx, dropletCreateRequest)
		if err != nil {
			log.Println("error creating droplet ", dropletCreateRequest.Name)
			if i == retries-1 {
				return err
			}
		} else {
			log.Println("create request for ", dropletCreateRequest.Name, "returned ", response.Status)
			break
		}
	}
	return nil
}
