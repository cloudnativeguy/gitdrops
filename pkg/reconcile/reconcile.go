package reconcile

import (
	"context"
	"errors"
	"github.com/nolancon/gitdrops/pkg/dolocal"
	"log"
	"os"

	"github.com/digitalocean/godo"
)

const (
	update            = "update"
	create            = "create"
	digitaloceanToken = "DIGITALOCEAN_TOKEN"
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

	client := godo.NewFromToken(os.Getenv(digitaloceanToken))
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
	dropletsToUpdate, dropletsToCreate := rd.dropletsToUpdateCreate()
	log.Println("droplets to update", dropletsToUpdate)
	log.Println("droplets to create ", dropletsToCreate)
	dropletsToDelete := rd.activeDropletsToDelete()
	log.Println("droplets to delete ", dropletsToDelete)

	dropletsToDelete = append(dropletsToDelete, dropletsToUpdate...)
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

	return nil
}

func (rd *ReconcileDroplets) deleteDroplets(ctx context.Context, dropletsToDelete []int) error {
	for _, id := range dropletsToDelete {
		response, err := rd.client.Droplets.Delete(ctx, id)
		if err != nil {
			log.Println("error during delete request for droplet ", id, " error: ", err)
			return err
		}
		log.Println("delete request for droplet ", id, " returned: ", response.StatusCode)
	}
	return nil
}

func (rd *ReconcileDroplets) createDroplets(ctx context.Context, dropletsToCreate []dolocal.LocalDropletCreateRequest) error {
	for _, dropletToCreate := range dropletsToCreate {
		dropletCreateRequest, err := translateDropletCreateRequest(dropletToCreate)
		if err != nil {
			log.Println("error convering gitdrops.yaml to droplet create request:")
			return err
		}
		log.Println("dropletCreateRequest", dropletCreateRequest)
		_, response, err := rd.client.Droplets.Create(ctx, dropletCreateRequest)
		log.Println("create request for ", dropletToCreate.Name, "returned ", response.Status)
		if err != nil {
			log.Println("error creating droplet ", dropletToCreate.Name)
			return err
		}

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
// * dropletsToUpdate: IDs of droplets that are active on DO and are defined in gitdrops.yaml,
// but the active droplets are no longer in sync with the local gitdrops version.
// * dropletsToCreate: Encompasses (A) LocalDropletCreateRequests of droplets defined in gitdrops.yaml
// that are NOT active on DO and therefore should be created. (B) Full LocalDropletCreateRequests of
// list A - needed later for deletion as update requires delete + create.
func (rd *ReconcileDroplets) dropletsToUpdateCreate() ([]int, []dolocal.LocalDropletCreateRequest) {
	dropletsToCreate := make([]dolocal.LocalDropletCreateRequest, 0)
	dropletsToUpdate := make([]int, 0)

	for _, localDropletCreateRequest := range rd.localDropletCreateRequests {
		dropletIsActive := false
		for _, activeDroplet := range rd.activeDroplets {
			if localDropletCreateRequest.Name == activeDroplet.Name {
				//droplet already exists, check for change in request
				log.Println("droplet found check for change")
				// only do below check if delete privileges are granted
				if dropletNeedsUpdate(localDropletCreateRequest, activeDroplet) {
					log.Println("droplet requires update")
					dropletsToCreate = append(dropletsToCreate, localDropletCreateRequest)
					dropletsToUpdate = append(dropletsToUpdate, activeDroplet.ID)
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
	return dropletsToUpdate, dropletsToCreate
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

// dropletNeedsUpdate compares a droplet defined in gitdrops.yaml with the same doplet active on DO.
// If there is a difference in Region, Size or Image, the droplet needs an update.
func dropletNeedsUpdate(localDropletCreateRequest dolocal.LocalDropletCreateRequest, activeDroplet godo.Droplet) bool {
	if activeDroplet.Region != nil && activeDroplet.Region.Name != localDropletCreateRequest.Region {
		return true
	}
	if activeDroplet.Size != nil && activeDroplet.Size.String() != localDropletCreateRequest.Size {
		return true
	}
	if activeDroplet.Image != nil && activeDroplet.Image.String() != localDropletCreateRequest.Image {
		return true
	}
	return false
}
