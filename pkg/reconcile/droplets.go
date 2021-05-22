package reconcile

import (
	"context"
	"errors"
	"log"

	"github.com/nolancon/gitdrops/pkg/gitdrops"

	"github.com/digitalocean/godo"
)

type DropletReconciler struct {
	privileges       gitdrops.Privileges
	client           *godo.Client
	activeDroplets   []godo.Droplet
	gitdropsDroplets []gitdrops.Droplet
	dropletsToCreate []gitdrops.Droplet
	dropletsToUpdate actionsByID
	dropletsToDelete []int
	volumeNameToID   map[string]string
}

var _ ObjectReconciler = &DropletReconciler{}

func (dr *DropletReconciler) SetActiveObjects(ctx context.Context) error {
	activeDroplets, err := gitdrops.ListDroplets(ctx, dr.client)
	if err != nil {
		log.Println("Error while listing droplets", err)
		return err
	}
	dr.activeDroplets = activeDroplets
	log.Println("active droplets:", len(activeDroplets))

	activeVolumes, err := gitdrops.ListVolumes(ctx, dr.client)
	if err != nil {
		log.Println("Error while listing volumes", err)
		return err
	}

	volumeNameToID := make(map[string]string)
	for _, activeVolume := range activeVolumes {
		volumeNameToID[activeVolume.Name] = activeVolume.ID
	}
	dr.volumeNameToID = volumeNameToID

	return nil
}

func (dr *DropletReconciler) Reconcile(ctx context.Context) error {
	if dr.privileges.Delete {
		err := dr.DeleteObjects(ctx)
		if err != nil {
			log.Println("error deleting droplet")
			return err
		}
	} else {
		log.Println("gitdrops.yaml does not have delete privileges")
	}
	if dr.privileges.Create {
		err := dr.CreateObjects(ctx)
		if err != nil {
			log.Println("error creating droplet")
			return err
		}
	} else {
		log.Println("gitdrops.yaml does not have create privileges")
	}

	if dr.privileges.Update {
		err := dr.UpdateObjects(ctx)
		if err != nil {
			log.Println("error updating droplet")
			return err
		}
	} else {
		log.Println("gitdrops.yaml does not have update privileges")
	}

	return nil
}

func (dr *DropletReconciler) SecondaryReconcile(context.Context, actionsByID) error {
	return nil
}

func (dr *DropletReconciler) translateDropletCreateRequest(gitdropsDroplet gitdrops.Droplet) (*godo.DropletCreateRequest, error) {
	createRequest := &godo.DropletCreateRequest{}
	if gitdropsDroplet.Name == "" {
		return createRequest, errors.New("droplet name not specified")
	}
	if gitdropsDroplet.Region == "" {
		return createRequest, errors.New("droplet region not specified")
	}
	if gitdropsDroplet.Size == "" {
		return createRequest, errors.New("droplet size not specified")
	}
	if gitdropsDroplet.Image == "" {
		return createRequest, errors.New("droplet image not specified")
	}
	createRequest.Name = gitdropsDroplet.Name
	createRequest.Region = gitdropsDroplet.Region
	createRequest.Size = gitdropsDroplet.Size
	dropletImage := godo.DropletCreateImage{}
	dropletImage.Slug = gitdropsDroplet.Image
	createRequest.Image = dropletImage

	if len(gitdropsDroplet.SSHKeyFingerprint) != 0 {
		dropletCreateSSHKey := godo.DropletCreateSSHKey{Fingerprint: gitdropsDroplet.SSHKeyFingerprint}
		dropletCreateSSHKeys := make([]godo.DropletCreateSSHKey, 0)
		dropletCreateSSHKeys = append(dropletCreateSSHKeys, dropletCreateSSHKey)
		createRequest.SSHKeys = dropletCreateSSHKeys
	}

	if gitdropsDroplet.VPCUUID != "" {
		createRequest.VPCUUID = gitdropsDroplet.VPCUUID
	}

	if len(gitdropsDroplet.Volumes) != 0 {
		dropletCreateVolumes := make([]godo.DropletCreateVolume, 0)
		for _, vol := range gitdropsDroplet.Volumes {
			dropletCreateVolume := godo.DropletCreateVolume{ID: dr.volumeNameToID[vol]}
			dropletCreateVolumes = append(dropletCreateVolumes, dropletCreateVolume)
		}
		createRequest.Volumes = dropletCreateVolumes
	}
	if len(gitdropsDroplet.Tags) != 0 {
		createRequest.Tags = gitdropsDroplet.Tags
	}
	if gitdropsDroplet.VPCUUID != "" {
		createRequest.VPCUUID = gitdropsDroplet.VPCUUID
	}
	if gitdropsDroplet.UserData.Data != "" {
		createRequest.UserData = gitdropsDroplet.UserData.Data
	}
	return createRequest, nil

}

// dropletsToUpdateCreate poulates DropletReconciler with two lists:
// * dropletsToUpdate: dropletActionsByID of droplets that are active on DO and are defined in
// gitdrops.yaml, but the active droplets are no longer in sync with the local gitdrops version.
// * dropletsToCreate: Droplets of droplets defined in gitdrops.yaml that are NOT
// active on DO and therefore should be created.
func (dr *DropletReconciler) SetObjectsToUpdateAndCreate() {
	dropletsToCreate := make([]gitdrops.Droplet, 0)
	dropletActionsByID := make(actionsByID)
	for _, gitdropsDroplet := range dr.gitdropsDroplets {
		dropletIsActive := false
		for _, activeDroplet := range dr.activeDroplets {
			if gitdropsDroplet.Name == activeDroplet.Name {
				// droplet already exists, check for change in request
				dropletActions := getDropletActions(gitdropsDroplet, activeDroplet)
				dropletActions = append(dropletActions, dr.volumesToDetach(activeDroplet, gitdropsDroplet)...)
				dropletActions = append(dropletActions, dr.volumesToAttach(activeDroplet, gitdropsDroplet)...)
				if len(dropletActions) != 0 {
					dropletActionsByID[activeDroplet.ID] = dropletActions
				}
				dropletIsActive = true
				continue
			}
		}
		if !dropletIsActive {
			//create droplet from local request
			log.Println("droplet not active, create droplet ", gitdropsDroplet)
			dropletsToCreate = append(dropletsToCreate, gitdropsDroplet)
		}
	}
	dr.dropletsToUpdate = dropletActionsByID
	dr.dropletsToCreate = dropletsToCreate
}

// ObjectToDelete populates DropletReconciler with a list of IDs for droplets that need
// to be deleted upon reconciliation of gitdrops.yaml (ie these droplets are active but not present
// in the spec)
func (dr *DropletReconciler) SetObjectsToDelete() {
	dropletsToDelete := make([]int, 0)

	for _, activeDroplet := range dr.activeDroplets {
		activeDropletInSpec := false
		for _, gitdropsDroplet := range dr.gitdropsDroplets {
			if gitdropsDroplet.Name == activeDroplet.Name {
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

func getDropletActions(gitdropsDroplet gitdrops.Droplet, activeDroplet godo.Droplet) []action {
	var dropletActions []action
	if activeDroplet.Size != nil && activeDroplet.Size.Slug != gitdropsDroplet.Size {
		log.Println("droplet", activeDroplet.Name, " size has been updated in gitdrops.yaml")

		dropletAction := action{
			action: resize,
			value:  gitdropsDroplet.Size,
		}
		dropletActions = append(dropletActions, dropletAction)

	}
	if activeDroplet.Image != nil && activeDroplet.Image.Slug != gitdropsDroplet.Image {
		log.Println("droplet", activeDroplet.Name, "image  has been updated in gitdrops.yaml")
		dropletAction := action{
			action: rebuild,
			value:  gitdropsDroplet.Image,
		}
		dropletActions = append(dropletActions, dropletAction)
	}

	return dropletActions
}

// volumesToDetach returns a slice of actions{action: detach, value: <volume-id>}
func (dr *DropletReconciler) volumesToDetach(activeDroplet godo.Droplet, gitdropsDroplet gitdrops.Droplet) []action {
	actions := make([]action, 0)
	for _, activeDropletVolumeID := range activeDroplet.VolumeIDs {
		volumeFound := false
		for _, gitdropsDropletVolume := range gitdropsDroplet.Volumes {
			if dr.volumeNameToID[gitdropsDropletVolume] == activeDropletVolumeID {
				volumeFound = true
				continue
			}
		}
		if !volumeFound {
			log.Println("volume", activeDropletVolumeID, "to be detached from droplet")
			action := action{
				action: detach,
				value:  activeDropletVolumeID,
			}
			actions = append(actions, action)
		}
	}
	return actions
}

// volumesToAttach returns a slice of actions{action: attach, value: <volume-id>}
func (dr *DropletReconciler) volumesToAttach(activeDroplet godo.Droplet, gitdropsDroplet gitdrops.Droplet) []action {
	actions := make([]action, 0)
	for _, gitdropsDropletVolume := range gitdropsDroplet.Volumes {
		volumeFound := false
		for _, activeDropletVolumeID := range activeDroplet.VolumeIDs {
			if dr.volumeNameToID[gitdropsDropletVolume] == activeDropletVolumeID {
				volumeFound = true
				continue
			}

		}
		if !volumeFound {
			// create attach action for volume
			log.Println("volume", gitdropsDropletVolume, "not attached, attach to droplet")
			action := action{
				action: attach,
				value:  dr.volumeNameToID[gitdropsDropletVolume],
			}
			actions = append(actions, action)
		}
	}
	return actions
}

func (dr *DropletReconciler) GetObjectsToUpdate() actionsByID {
	return dr.dropletsToUpdate
}

func (dr *DropletReconciler) DeleteObjects(ctx context.Context) error {
	for _, id := range dr.dropletsToDelete {
		err := gitdrops.DeleteDroplet(ctx, dr.client, id)
		if err != nil {
			log.Println("error during delete request for droplet ", id, " error: ", err)
			return err
		}
	}
	return nil
}

func (dr *DropletReconciler) CreateObjects(ctx context.Context) error {
	for _, dropletToCreate := range dr.dropletsToCreate {
		dropletCreateRequest, err := dr.translateDropletCreateRequest(dropletToCreate)
		if err != nil {
			log.Println("error converting gitdrops.yaml to droplet create request:")
			return err
		}
		err = gitdrops.CreateDroplet(ctx, dr.client, dropletCreateRequest)
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
			err := gitdrops.UpdateDroplet(ctx, dr.client, id.(int), dropletAction.action, dropletAction.value.(string))
			if err != nil {
				log.Println("error during action request for droplet ", id, " error: ", err)
				// we do not return here as there may be more actions to complete
				// for this droplet.
			}
		}
	}
	return nil
}
