package reconcile

import (
	"context"
	"errors"
	"log"

	"github.com/nolancon/gitdrops/pkg/dolocal"

	"github.com/digitalocean/godo"
)

type VolumeReconciler struct {
	privileges                dolocal.Privileges
	client                    *godo.Client
	activeVolumes             []godo.Volume
	localVolumeCreateRequests []dolocal.LocalVolumeCreateRequest
	volumesToCreate           []dolocal.LocalVolumeCreateRequest
	volumesToUpdate           actionsByID
	volumesToDelete           []string
}

var _ ObjectReconciler = &VolumeReconciler{}

func (vr *VolumeReconciler) Populate(ctx context.Context) error {
	activeVolumes, err := dolocal.ListVolumes(ctx, vr.client)
	if err != nil {
		log.Println("Error while listing volumes", err)
		return err
	}

	vr.activeVolumes = activeVolumes
	vr.SetObjectsToUpdateAndCreate()
	vr.SetObjectsToDelete()

	log.Println("active volumes:", len(activeVolumes))
	log.Println("active volumes to delete:", vr.volumesToDelete)
	log.Println("gitdrops volumes to update:", vr.volumesToUpdate)
	log.Println("gitdrops volumes to create:", vr.volumesToCreate)

	return nil
}

func (vr *VolumeReconciler) Reconcile(ctx context.Context) error {
	if vr.privileges.Create {
		err := vr.CreateObjects(ctx)
		if err != nil {
			log.Println("error creating volume")
			return err
		}
	} else {
		log.Println("gitdrops.yaml does not have create privileges")
	}
	if vr.privileges.Update {
		err := vr.UpdateObjects(ctx)
		if err != nil {
			log.Println("error updating volume")
			return err
		}
	} else {
		log.Println("gitdrops.yaml does not have update privileges")
	}

	return nil
}

func (vr *VolumeReconciler) SetActiveObjects(ctx context.Context) error {
	activeVolumes, err := dolocal.ListVolumes(ctx, vr.client)
	if err != nil {
		log.Println("Error while listing volumes", err)
		return err
	}
	vr.activeVolumes = activeVolumes
	return nil
}

func (vr *VolumeReconciler) SecondaryReconcile(ctx context.Context, objectsToUpdate actionsByID) error {
	if vr.privileges.Delete {
		err := vr.DeleteObjects(ctx)
		if err != nil {
			log.Println("error deleting droplet")
			return err
		}
	} else {
		log.Println("gitdrops.yaml does not have delete privileges")
	}

	err := vr.SetActiveObjects(ctx)
	if err != nil {
		return err
	}

	vr.volumesToUpdate = objectsToUpdate
	err = vr.UpdateObjects(ctx)
	if err != nil {
		return err
	}

	return nil
}

func translateVolumeCreateRequest(localVolumeCreateRequest dolocal.LocalVolumeCreateRequest) (*godo.VolumeCreateRequest, error) {
	createRequest := &godo.VolumeCreateRequest{}
	if localVolumeCreateRequest.Name == "" {
		return createRequest, errors.New("volume name not specified")
	}
	if localVolumeCreateRequest.Region == "" {
		return createRequest, errors.New("volume region not specified")
	}
	if localVolumeCreateRequest.SizeGigaBytes == 0 {
		return createRequest, errors.New("volume sizeGigaBytes not specified")
	}
	createRequest.Name = localVolumeCreateRequest.Name
	createRequest.Region = localVolumeCreateRequest.Region
	createRequest.SizeGigaBytes = localVolumeCreateRequest.SizeGigaBytes
	return createRequest, nil

}

// SetObjectsToUpdateCreate populates VolumeReconciler with two lists:
// * volumesToUpdate: volumeActionsByID of volumes that are active on DO and are defined in
// gitdrops.yaml, but the active volumes are no longer in sync with the local gitdrops version.
// * volumesToCreate: LocalVolumeCreateRequests of volumes defined in gitdrops.yaml that are NOT
// active on DO and therefore should be created.
func (vr *VolumeReconciler) SetObjectsToUpdateAndCreate() {
	volumesToCreate := make([]dolocal.LocalVolumeCreateRequest, 0)
	volumeActionsByID := make(actionsByID)
	for _, localVolumeCreateRequest := range vr.localVolumeCreateRequests {
		volumeIsActive := false
		for _, activeVolume := range vr.activeVolumes {
			if localVolumeCreateRequest.Name == activeVolume.Name {
				//volume already exists, check for change in request
				volumeActions := getVolumeActions(localVolumeCreateRequest, activeVolume)
				if len(volumeActions) != 0 {
					volumeActionsByID[activeVolume.ID] = volumeActions
				}
				volumeIsActive = true
				continue
			}
		}
		if !volumeIsActive {
			//create volume from local request
			log.Println("volume not active, create volume ", localVolumeCreateRequest)
			volumesToCreate = append(volumesToCreate, localVolumeCreateRequest)
		}
	}
	vr.volumesToUpdate = volumeActionsByID
	vr.volumesToCreate = volumesToCreate
}

// SetObjectToDelete populates VolumeReconciler with  a list of IDs for volumes that need
// to be deleted upon reconciliation of gitdrops.yaml (ie these volumes are active but not present
// in the spec)
func (vr *VolumeReconciler) SetObjectsToDelete() {
	volumesToDelete := make([]string, 0)

	for _, activeVolume := range vr.activeVolumes {
		activeVolumeInSpec := false
		for _, localVolumeCreateRequest := range vr.localVolumeCreateRequests {
			if localVolumeCreateRequest.Name == activeVolume.Name {
				activeVolumeInSpec = true
				continue
			}
		}
		if !activeVolumeInSpec {
			//create volume from local request
			volumesToDelete = append(volumesToDelete, activeVolume.ID)
		}
	}
	vr.volumesToDelete = volumesToDelete
}

func (vr *VolumeReconciler) GetObjectsToUpdate() actionsByID {
	return vr.volumesToUpdate
}

func getVolumeActions(localVolumeCreateRequest dolocal.LocalVolumeCreateRequest, activeVolume godo.Volume) []action {
	var volumeActions []action
	if activeVolume.SizeGigaBytes != 0 && activeVolume.SizeGigaBytes != localVolumeCreateRequest.SizeGigaBytes {
		log.Println("volume", activeVolume.Name, "size has been updated in gitdrops.yaml")

		volumeAction := action{
			action: resize,
			value:  localVolumeCreateRequest.SizeGigaBytes,
		}
		volumeActions = append(volumeActions, volumeAction)

	}
	return volumeActions
}

func (vr *VolumeReconciler) DeleteObjects(ctx context.Context) error {
	for _, id := range vr.volumesToDelete {
		err := dolocal.DeleteVolume(ctx, vr.client, id)
		if err != nil {
			log.Println("error during delete request for volume ", id, " error: ", err)
			return err
		}
	}
	return nil
}

func (vr *VolumeReconciler) CreateObjects(ctx context.Context) error {
	for _, volumeToCreate := range vr.volumesToCreate {
		volumeCreateRequest, err := translateVolumeCreateRequest(volumeToCreate)
		if err != nil {
			log.Println("error converting gitdrops.yaml to volume create request:")
			return err
		}
		err = dolocal.CreateVolume(ctx, vr.client, volumeCreateRequest)
		if err != nil {
			log.Println("error creating volume ", volumeToCreate.Name)
			return err
		}

	}
	return nil
}

func (vr *VolumeReconciler) UpdateObjects(ctx context.Context) error {
	for id, volumeActions := range vr.volumesToUpdate {
		for _, volumeAction := range volumeActions {
			switch volumeAction.action {
			case resize:
				err := dolocal.ResizeVolume(ctx, vr.client, id.(string), vr.findVolumeRegion(id.(string)), volumeAction.value)
				if err != nil {
					log.Println("error during resize action request for volume ", id, " error: ", err)
					// we do not return here as there may be more actions to complete
					// for this volume.
				}
			case attach:
				// in this case, 'id' is that of the droplet and 'value' is the volume
				// name. This is because this action was detected and created by the
				// droplet reconciler.
				err := dolocal.AttachVolume(ctx, vr.client, volumeAction.value.(string), id.(int))
				if err != nil {
					log.Println("error during attach action request for volume ", volumeAction.value.(string), " error: ", err)

				}
			case detach:
				// in this case, 'id' is that of the droplet and 'value' is the volume
				// id. This is because this action was detected and created by the
				// droplet reconciler.
				err := dolocal.DetachVolume(ctx, vr.client, volumeAction.value.(string), id.(int))
				if err != nil {
					log.Println("error during detach action request for volume ", volumeAction.value.(string), " error: ", err)

				}

			}
		}
	}
	return nil
}

func (vr *VolumeReconciler) findVolumeIDByName(volName string) string {
	for _, vol := range vr.activeVolumes {
		if vol.Name == volName {
			return vol.ID
		}
	}
	return ""
}

func (vr *VolumeReconciler) findVolumeRegion(volID string) string {
	for _, vol := range vr.activeVolumes {
		if vol.ID == volID {
			return vol.Region.Slug
		}
	}
	return ""
}
