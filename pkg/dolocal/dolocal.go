package dolocal

import (
	"context"
	"errors"
	"github.com/digitalocean/godo"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

const (
	gitdropsYamlPath = "./gitdrops.yaml"
	retries          = 10
	resize           = "resize"
	rebuild          = "rebuild"
)

// ReadGitDrops reads and unmarshals from gitdrops.yaml
func ReadGitDrops() (GitDrops, error) {
	gitDrops := GitDrops{}

	gitdropsYaml, err := ioutil.ReadFile(gitdropsYamlPath)
	if err != nil {
		return gitDrops, err
	}

	err = yaml.Unmarshal(gitdropsYaml, &gitDrops)
	if err != nil {
		return gitDrops, err
	}
	for i, droplet := range gitDrops.Droplets {
		if droplet.UserData.Path == "" {
			continue
		}
		userData, err := ioutil.ReadFile(droplet.UserData.Path)
		if err != nil {
			return gitDrops, err
		}
		gitDrops.Droplets[i].UserData.Data = string(userData)
	}
	log.Println("created:", gitDrops)
	return gitDrops, nil
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

func UpdateDroplet(ctx context.Context, client *godo.Client, id int, action, value string) error {
	switch action {
	case resize:
		for i := 0; i < retries; i++ {
			_, response, err := client.DropletActions.Resize(ctx, id, value, true)
			if err != nil {
				log.Println("error resizing droplet ", id)
				if i == retries-1 {
					return err
				}
			} else {
				log.Println("droplet action request for resize ", id, "returned ", response.Status)
				break
			}
		}
	case rebuild:
		for i := 0; i < retries; i++ {
			_, response, err := client.DropletActions.RebuildByImageSlug(ctx, id, value)
			if err != nil {
				log.Println("error resizing droplet ", id)
				if i == retries-1 {
					return err
				}
			} else {
				log.Println("droplet action request for rebuild ", id, "returned ", response.Status)
				break
			}
		}

	}
	return nil
}
