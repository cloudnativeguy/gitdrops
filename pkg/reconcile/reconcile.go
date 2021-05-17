package reconcile

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/nolancon/gitdrops/pkg/dolocal"

	"github.com/digitalocean/godo"
)

const (
	update            = "update"
	create            = "create"
	resize            = "resize"
	rebuild           = "rebuild"
	digitaloceanToken = "DIGITALOCEAN_TOKEN"
)

type ReconcileDroplets struct {
	client                     *godo.Client
	activeDroplets             []godo.Droplet
	localDropletCreateRequests []dolocal.LocalDropletCreateRequest
}

type dropletActionsByID map[int][]dropletAction

type dropletAction struct {
	action string
	value  string
}

func NewReconcileDroplets(ctx context.Context) (ReconcileDroplets, error) {
	localDropletCreateRequests, err := dolocal.ReadLocalDropletCreateRequests()
	if err != nil {
		log.Println(err)
		return ReconcileDroplets{}, err
	}

	log.Println("gitdrops droplet create requests:", localDropletCreateRequests)

	client := godo.NewFromToken(os.Getenv(digitaloceanToken))
	activeDroplets, err := dolocal.ListDroplets(ctx, client)
	if err != nil {
		log.Println("Error while listing droplets", err)
		return ReconcileDroplets{}, err
	}
	log.Println("active droplets on DO:", activeDroplets)

	return ReconcileDroplets{
		client:                     client,
		localDropletCreateRequests: localDropletCreateRequests,
		activeDroplets:             activeDroplets,
	}, nil
}

func (rd *ReconcileDroplets) Reconcile(ctx context.Context) error {
	dropletsToUpdate, dropletsToCreate := rd.dropletsToUpdateCreate()
	log.Println("active droplets to update", dropletsToUpdate)
	log.Println("active droplets to create ", dropletsToCreate)
	dropletsToDelete := rd.activeDropletsToDelete()
	log.Println("active droplets to delete ", dropletsToDelete)

	dropletsToDelete = append(dropletsToDelete)
	err := rd.deleteDroplets(ctx, dropletsToDelete)
	if err != nil {
		log.Println("error deleting droplet")
		return err
	}
	err = rd.createDroplets(ctx, dropletsToCreate)
	if err != nil {
		log.Println("error creating droplet")
		return err
	}
	err = rd.updateDroplets(ctx, dropletsToUpdate)
	if err != nil {
		log.Println("error creating droplet")
		return err
	}

	return nil
}

func translateDropletCreateRequest(localDropletCreateRequest dolocal.LocalDropletCreateRequest) (*godo.DropletCreateRequest, error) {
	createRequest := &godo.DropletCreateRequest{}
	if localDropletCreateRequest.Name == "" {
		return createRequest, errors.New("droplet name not specified")
	}
	if localDropletCreateRequest.Region == "" {
		return createRequest, errors.New("droplet region not specified")
	}
	if localDropletCreateRequest.Size == "" {
		return createRequest, errors.New("droplet size not specified")
	}
	if localDropletCreateRequest.Image == "" {
		return createRequest, errors.New("droplet image not specified")
	}
	createRequest.Name = localDropletCreateRequest.Name
	createRequest.Region = localDropletCreateRequest.Region
	createRequest.Size = localDropletCreateRequest.Size
	dropletImage := godo.DropletCreateImage{}
	dropletImage.Slug = localDropletCreateRequest.Image
	createRequest.Image = dropletImage

	if len(localDropletCreateRequest.SSHKeyFingerprint) != 0 {
		dropletCreateSSHKey := godo.DropletCreateSSHKey{Fingerprint: localDropletCreateRequest.SSHKeyFingerprint}
		dropletCreateSSHKeys := make([]godo.DropletCreateSSHKey, 0)
		dropletCreateSSHKeys = append(dropletCreateSSHKeys, dropletCreateSSHKey)
		createRequest.SSHKeys = dropletCreateSSHKeys
	}

	if localDropletCreateRequest.VPCUUID != "" {
		createRequest.VPCUUID = localDropletCreateRequest.VPCUUID
	}

	if len(localDropletCreateRequest.Volumes) != 0 {
		dropletCreateVolumes := make([]godo.DropletCreateVolume, 0)
		for _, vol := range localDropletCreateRequest.Volumes {
			dropletCreateVolume := godo.DropletCreateVolume{ID: vol}
			dropletCreateVolumes = append(dropletCreateVolumes, dropletCreateVolume)
		}
		createRequest.Volumes = dropletCreateVolumes
	}
	if len(localDropletCreateRequest.Tags) != 0 {
		createRequest.Tags = localDropletCreateRequest.Tags
	}
	if localDropletCreateRequest.VPCUUID != "" {
		createRequest.VPCUUID = localDropletCreateRequest.VPCUUID
	}

	return createRequest, nil

}

// dropletsToUpdateCreate returns two lists:
// * dropletsToUpdate: dropletActionsByID of droplets that are active on DO and are defined in
// gitdrops.yaml, but the active droplets are no longer in sync with the local gitdrops version.
// * dropletsToCreate: LocalDropletCreateRequests of droplets defined in gitdrops.yaml that are NOT
// active on DO and therefore should be created.
func (rd *ReconcileDroplets) dropletsToUpdateCreate() (dropletActionsByID, []dolocal.LocalDropletCreateRequest) {
	dropletsToCreate := make([]dolocal.LocalDropletCreateRequest, 0)
	dropletActionsByID := make(dropletActionsByID)
	for _, localDropletCreateRequest := range rd.localDropletCreateRequests {
		dropletIsActive := false
		for _, activeDroplet := range rd.activeDroplets {
			if localDropletCreateRequest.Name == activeDroplet.Name {
				//droplet already exists, check for change in request
				log.Println("droplet found check for change")
				// only do below check if delete privileges are granted
				dropletActions := getDropletActions(localDropletCreateRequest, activeDroplet)
				if len(dropletActions) != 0 {
					dropletActionsByID[activeDroplet.ID] = dropletActions
				}
				dropletIsActive = true
				continue
			}
		}
		if !dropletIsActive {
			//create droplet from local request
			log.Println("droplet not active, create droplet ", localDropletCreateRequest)
			dropletsToCreate = append(dropletsToCreate, localDropletCreateRequest)
		}
	}
	return dropletActionsByID, dropletsToCreate
}

// activeDropletsToDelete return a list of IDs for droplets that need to be deleted upon reconciliation
// of gitdrops.yaml (ie these droplets are active but not present in the spec)
func (rd *ReconcileDroplets) activeDropletsToDelete() []int {
	dropletsToDelete := make([]int, 0)

	for _, activeDroplet := range rd.activeDroplets {
		activeDropletInSpec := false
		for _, localDropletCreateRequest := range rd.localDropletCreateRequests {
			if localDropletCreateRequest.Name == activeDroplet.Name {
				activeDropletInSpec = true
				continue
			}
		}
		if !activeDropletInSpec {
			//create droplet from local request
			dropletsToDelete = append(dropletsToDelete, activeDroplet.ID)
		}
	}
	return dropletsToDelete
}

func getDropletActions(localDropletCreateRequest dolocal.LocalDropletCreateRequest, activeDroplet godo.Droplet) []dropletAction {
	var dropletActions []dropletAction
	if activeDroplet.Size != nil && activeDroplet.Size.Slug != localDropletCreateRequest.Size {
		log.Println("droplet (name)", activeDroplet.Name, " (ID)", activeDroplet.ID, " size has been updated in gitdrops.yaml")

		dropletAction := dropletAction{
			action: resize,
			value:  localDropletCreateRequest.Size,
		}
		dropletActions = append(dropletActions, dropletAction)

	}
	if activeDroplet.Image != nil && activeDroplet.Image.Slug != localDropletCreateRequest.Image {
		log.Println("droplet (name)", activeDroplet.Name, " (ID)", activeDroplet.ID, " image  has been updated in gitdrops.yaml")
		dropletAction := dropletAction{
			action: rebuild,
			value:  localDropletCreateRequest.Image,
		}
		dropletActions = append(dropletActions, dropletAction)
	}
	return dropletActions
}

func (rd *ReconcileDroplets) deleteDroplets(ctx context.Context, dropletsToDelete []int) error {
	for _, id := range dropletsToDelete {
		err := dolocal.DeleteDroplet(ctx, rd.client, id)
		if err != nil {
			log.Println("error during delete request for droplet ", id, " error: ", err)
			return err
		}
	}
	return nil
}

func (rd *ReconcileDroplets) createDroplets(ctx context.Context, dropletsToCreate []dolocal.LocalDropletCreateRequest) error {
	for _, dropletToCreate := range dropletsToCreate {
		dropletCreateRequest, err := translateDropletCreateRequest(dropletToCreate)
		if err != nil {
			log.Println("error converting gitdrops.yaml to droplet create request:")
			return err
		}
		log.Println("dropletCreateRequest", dropletCreateRequest)
		err = dolocal.CreateDroplet(ctx, rd.client, dropletCreateRequest)
		if err != nil {
			log.Println("error creating droplet ", dropletToCreate.Name)
			return err
		}

	}
	return nil
}

func (rd *ReconcileDroplets) updateDroplets(ctx context.Context, dropletsToUpdate dropletActionsByID) error {
	for id, dropletActions := range dropletsToUpdate {
		for _, dropletAction := range dropletActions {
			err := dolocal.UpdateDroplet(ctx, rd.client, id, dropletAction.action, dropletAction.value)
			if err != nil {
				log.Println("error during action request for droplet ", id, " error: ", err)
				return err
			}
		}
	}
	return nil
}
