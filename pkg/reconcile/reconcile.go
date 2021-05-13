package reconcile

import (
	"context"
	"github.com/nolancon/gitdrops/pkg/dolocal"

	"log"
	"os"

	"github.com/digitalocean/godo"
)

type ReconcileDroplets struct {
	client                     *godo.Client
	activeDroplets             []godo.Droplet
	localDropletCreateRequests []dolocal.LocalDropletCreateRequest
}

func NewReconcileDroplets(ctx context.Context) (ReconcileDroplets, error) {
	localDropletCreateRequests, err := dolocal.ReadLocalDropletCreateRequests()
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
	activeDroplets, err := dolocal.ListDroplets(ctx, client)
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

func (rd *ReconcileDroplets) Reconcile(ctx context.Context) error {

	for _, localDropletCreateRequest := range rd.localDropletCreateRequests {
		dropletIsActive := false
		for _, activeDroplet := range rd.activeDroplets {
			if localDropletCreateRequest.Name == activeDroplet.Name {
				//droplet already exists, check for change in request
				log.Println("droplet found check for change")
				dropletIsActive = true
				continue
			}
		}
		if !dropletIsActive {
			//create droplet from local request
			log.Println("droplet not active, create droplet ", localDropletCreateRequest)
			break
		}
	}
	// if privilege is set in config (TODO) delete dropletsToDelete
	return nil
}
