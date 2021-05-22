package gitdrops

import (
	"context"
	"io/ioutil"
	"log"

	"github.com/digitalocean/godo"

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
	log.Println("gitdrops.yaml contains", len(gitDrops.Droplets), "droplet(s) and", len(gitDrops.Volumes), "volume(s)")
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
		for i := 0; i < retries; i++ {
			dropletsTmp, respTmp, err := client.Droplets.List(ctx, opt)
			if err != nil {
				log.Println("error listing droplets", err)
				if i == retries-1 {
					return list, err
				}
			} else {
				droplets = dropletsTmp
				resp = respTmp
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

// DeleteDroplet attempts to delete droplet from DO by ID
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

// DeleteDroplet attempts to create droplet on DO by dropletCreateRequest
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

// UpdateDroplet attempts to perform an action (resize or rebuild) on an active droplet on DO by ID
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

// ListVolumes lists all active volumes on DO account
func ListVolumes(ctx context.Context, client *godo.Client) ([]godo.Volume, error) {
	list := []godo.Volume{}

	// create options. initially, these will be blank
	listOpt := &godo.ListOptions{}
	opt := &godo.ListVolumeParams{ListOptions: listOpt}
	for {
		volumes := []godo.Volume{}
		resp := &godo.Response{}
		for i := 0; i < retries; i++ {
			volumesTmp, respTmp, err := client.Storage.ListVolumes(ctx, opt)
			if err != nil {
				log.Println("error listing volumes", err)
				if i == retries-1 {
					return list, err
				}
			} else {
				volumes = volumesTmp
				resp = respTmp
				break
			}
		}
		// append the current page's volumes to our list
		list = append(list, volumes...)

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.ListOptions.Page = page + 1
	}

	return list, nil
}

// DeleteVolume attempts to delete volume from DO by ID
func DeleteVolume(ctx context.Context, client *godo.Client, id string) error {
	for i := 0; i < retries; i++ {
		response, err := client.Storage.DeleteVolume(ctx, id)
		if err != nil {
			log.Println("error during delete request for volume ", id, " error: ", err)
			if i == retries-1 {
				return err
			}
		} else {
			log.Println("delete request for volume ", id, " returned: ", response.StatusCode)
			break
		}
	}
	return nil
}

// CreateVolume attempts to create droplet on DO by dropletCreateRequest
func CreateVolume(ctx context.Context, client *godo.Client, volumeCreateRequest *godo.VolumeCreateRequest) error {
	for i := 0; i < retries; i++ {
		_, response, err := client.Storage.CreateVolume(ctx, volumeCreateRequest)
		if err != nil {
			log.Println("error creating volume ", volumeCreateRequest.Name)
			if i == retries-1 {
				return err
			}
		} else {
			log.Println("create request for ", volumeCreateRequest.Name, "returned ", response.Status)
			break
		}
	}
	return nil
}

// AttachVolume attempts to attach a volume to a droplet
func AttachVolume(ctx context.Context, client *godo.Client, volID string, dropletID int) error {
	for i := 0; i < retries; i++ {
		_, response, err := client.StorageActions.Attach(ctx, volID, dropletID)
		if err != nil {
			log.Println("error attaching volume ", volID, " to droplet ", dropletID)
			if i == retries-1 {
				return err
			}
		} else {
			log.Println("volume action request for attachment ", volID, "returned ", response.Status)
			break
		}
	}
	return nil
}

// AttachVolume attempts to attach a volume to a droplet
func DetachVolume(ctx context.Context, client *godo.Client, volID string, dropletID int) error {
	for i := 0; i < retries; i++ {
		_, response, err := client.StorageActions.DetachByDropletID(ctx, volID, dropletID)
		if err != nil {
			log.Println("error detaching volume ", volID)
			if i == retries-1 {
				return err
			}
		} else {
			log.Println("volume action request for detachment ", volID, "returned ", response.Status)
			break
		}
	}
	return nil
}

// ResizeVolume attempts to perform an action (resize) on an active volume on DO by ID
func ResizeVolume(ctx context.Context, client *godo.Client, volID, region string, value interface{}) error {
	for i := 0; i < retries; i++ {
		_, response, err := client.StorageActions.Resize(ctx, volID, value.(int), region)
		if err != nil {
			log.Println("error resizing volume ", volID)
			if i == retries-1 {
				return err
			}
		} else {
			log.Println("volume action request for resize ", volID, "returned ", response.Status)
			break
		}
	}
	return nil
}
