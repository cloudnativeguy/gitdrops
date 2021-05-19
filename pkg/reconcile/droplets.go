package reconcile

import (
	"context"
	"errors"
	"log"

	"github.com/nolancon/gitdrops/pkg/dolocal"

	"github.com/digitalocean/godo"
)

type DropletReconciler struct {
	client                     *godo.Client
	activeDroplets             []godo.Droplet
	localDropletCreateRequests []dolocal.LocalDropletCreateRequest
	dropletsToCreate           []dolocal.LocalDropletCreateRequest
	dropletsToUpdate           actionsByID
	dropletsToDelete           []int
}

func (dr *DropletReconciler) Populate(ctx context.Context) error {
	activeDroplets, err := dolocal.ListDroplets(ctx, dr.client)
	if err != nil {
		log.Println("Error while listing droplets", err)
		return err
	}

	dr.activeDroplets = activeDroplets
	dr.ObjectsToUpdateAndCreate()
	dr.ObjectsToDelete()

	log.Println("active droplets:", len(activeDroplets))
	log.Println("active droplets to delete:", dr.dropletsToDelete)
	log.Println("gitdrops droplets to update:", dr.dropletsToUpdate)
	log.Println("gitdrops droplets to create:", dr.dropletsToCreate)

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
	if localDropletCreateRequest.UserData.Data != "" {
		createRequest.UserData = localDropletCreateRequest.UserData.Data
	}
	return createRequest, nil

}

// dropletsToUpdateCreate poulates DropletReconciler with two lists:
// * dropletsToUpdate: dropletActionsByID of droplets that are active on DO and are defined in
// gitdrops.yaml, but the active droplets are no longer in sync with the local gitdrops version.
// * dropletsToCreate: LocalDropletCreateRequests of droplets defined in gitdrops.yaml that are NOT
// active on DO and therefore should be created.
func (dr *DropletReconciler) ObjectsToUpdateAndCreate() {
	dropletsToCreate := make([]dolocal.LocalDropletCreateRequest, 0)
	dropletActionsByID := make(actionsByID)
	for _, localDropletCreateRequest := range dr.localDropletCreateRequests {
		dropletIsActive := false
		for _, activeDroplet := range dr.activeDroplets {
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
	dr.dropletsToUpdate = dropletActionsByID
	dr.dropletsToCreate = dropletsToCreate
}

// ObjectToDelete populates DropletReconciler with  a list of IDs for droplets that need
// to be deleted upon reconciliation of gitdrops.yaml (ie these droplets are active but not present
// in the spec)
func (dr *DropletReconciler) ObjectsToDelete() {
	dropletsToDelete := make([]int, 0)

	for _, activeDroplet := range dr.activeDroplets {
		activeDropletInSpec := false
		for _, localDropletCreateRequest := range dr.localDropletCreateRequests {
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
	dr.dropletsToDelete = dropletsToDelete
}

func getDropletActions(localDropletCreateRequest dolocal.LocalDropletCreateRequest, activeDroplet godo.Droplet) []action {
	var dropletActions []action
	if activeDroplet.Size != nil && activeDroplet.Size.Slug != localDropletCreateRequest.Size {
		log.Println("droplet (name)", activeDroplet.Name, " (ID)", activeDroplet.ID, " size has been updated in gitdrops.yaml")

		dropletAction := action{
			action: resize,
			value:  localDropletCreateRequest.Size,
		}
		dropletActions = append(dropletActions, dropletAction)

	}
	if activeDroplet.Image != nil && activeDroplet.Image.Slug != localDropletCreateRequest.Image {
		log.Println("droplet (name)", activeDroplet.Name, " (ID)", activeDroplet.ID, " image  has been updated in gitdrops.yaml")
		dropletAction := action{
			action: rebuild,
			value:  localDropletCreateRequest.Image,
		}
		dropletActions = append(dropletActions, dropletAction)
	}
	return dropletActions
}

func (dr *DropletReconciler) DeleteObjects(ctx context.Context) error {
	for _, id := range dr.dropletsToDelete {
		err := dolocal.DeleteDroplet(ctx, dr.client, id)
		if err != nil {
			log.Println("error during delete request for droplet ", id, " error: ", err)
			return err
		}
	}
	return nil
}

func (dr *DropletReconciler) CreateObjects(ctx context.Context) error {
	for _, dropletToCreate := range dr.dropletsToCreate {
		dropletCreateRequest, err := translateDropletCreateRequest(dropletToCreate)
		if err != nil {
			log.Println("error converting gitdrops.yaml to droplet create request:")
			return err
		}
		log.Println("dropletCreateRequest", dropletCreateRequest)
		err = dolocal.CreateDroplet(ctx, dr.client, dropletCreateRequest)
		if err != nil {
			log.Println("error creating droplet ", dropletToCreate.Name)
			return err
		}

	}
	return nil
}

func (dr *DropletReconciler) UpdateObjects(ctx context.Context) error {
	for id, dropletActions := range dr.dropletsToUpdate {
		for _, dropletAction := range dropletActions {
			err := dolocal.UpdateDroplet(ctx, dr.client, id.(int), dropletAction.action, dropletAction.value.(string))
			if err != nil {
				log.Println("error during action request for droplet ", id, " error: ", err)
				// we do not return here as there may be more actions to complete
				// for this droplet.
			}
		}
	}
	return nil
}
