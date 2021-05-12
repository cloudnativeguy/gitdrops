package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/digitalocean/godo"
	"gopkg.in/yaml.v2"
)

const (
	shortDuration    = 3 * time.Second
	gitdropsYamlPath = "./gitdrops.yaml"
)

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

type ReconcileDroplets struct {
	client                     *godo.Client
	activeDroplets             []godo.Droplet
	localDropletCreateRequests []LocalDropletCreateRequest
}

func newReconcileDroplets(ctx context.Context) (ReconcileDroplets, error) {
	localDropletCreateRequests, err := fetchLocalDropletCreateRequests()
	if err != nil {
		log.Println(err)
		return ReconcileDroplets{}, err
	}
	if len(localDropletCreateRequests) == 0 {
		log.Println("no droplet create requests found in gitdrops file")
		return ReconcileDroplets{}, nil
		///TODO: handle this to abort before reconcile
	}

	log.Println("GitDrops:", localDropletCreateRequests)

	client := godo.NewFromToken(os.Getenv("DIGITALOCEAN_TOKEN"))
	activeDroplets, err := DropletList(ctx, client)
	if err != nil {
		log.Println("Error while listing droplets", err)
		return ReconcileDroplets{}, err
	}
	log.Println(activeDroplets)

	return ReconcileDroplets{
		client:                     client,
		localDropletCreateRequests: localDropletCreateRequests,
		activeDroplets:             activeDroplets,
	}, nil
}

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), shortDuration)
	defer cancel()
	reconcileDroplets, err := newReconcileDroplets(ctx)
	if err != nil {
		return
	}
	err = reconcileDroplets.Reconcile(ctx)
	if err != nil {
		return
	}
}

func fetchLocalDropletCreateRequests() ([]LocalDropletCreateRequest, error) {
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

func (rd *ReconcileDroplets) Reconcile(ctx context.Context) error {
	dropletsToDelete := make([]int, 0)
	for _, localDropletCreateRequest := range rd.localDropletCreateRequests {
		dropletIsActive := false
		for _, activeDroplet := range rd.activeDroplets {
			if localDropletCreateRequest.Name == activeDroplet.Name {
				//droplet already exists, check for change in request
				log.Println("droplet found check for change")
				dropletIsActive = true
				continue
			}
			dropletsToDelete = append(dropletsToDelete, activeDroplet.ID)
		}
		if !dropletIsActive {
			//create droplet from local request
			log.Println("droplet not active, create droplet ", localDropletCreateRequest)
		}
	}
	log.Println("dropletsToDelete", dropletsToDelete)
	// if privilege is set in config (TODO) delete dropletsToDelete
	return nil
}

func DropletList(ctx context.Context, client *godo.Client) ([]godo.Droplet, error) {
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
