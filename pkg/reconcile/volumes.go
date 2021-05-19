package reconcile

import (
	"context"
	"errors"
	"log"

	"github.com/nolancon/gitdrops/pkg/dolocal"

	"github.com/digitalocean/godo"
)

type VolumeReconciler struct {
	client                    *godo.Client
	activeVolumes             []godo.Volume
	localVolumeCreateRequests []dolocal.LocalVolumeCreateRequest
	volumesToCreate           []dolocal.LocalVolumeCreateRequest
	volumesToUpdate           actionsByID
	volumesToDelete           []string
}

func (vr *VolumeReconciler) Populate(ctx context.Context) error {
	activeVolumes, err := dolocal.ListVolumes(ctx, vr.client)
	if err != nil {
		log.Println("Error while listing volumes", err)
		return err
	}

	vr.activeVolumes = activeVolumes
	vr.ObjectsToUpdateAndCreate()
	vr.ObjectsToDelete()

	log.Println("active volumes:", len(activeVolumes))
	log.Println("active volumes to delete:", vr.volumesToDelete)
	log.Println("gitdrops volumes to update:", vr.volumesToUpdate)
	log.Println("gitdrops volumes to create:", vr.volumesToCreate)

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

// volumesToUpdateCreate poulates VolumeReconciler with two lists:
// * volumesToUpdate: volumeActionsByID of volumes that are active on DO and are defined in
// gitdrops.yaml, but the active volumes are no longer in sync with the local gitdrops version.
// * volumesToCreate: LocalVolumeCreateRequests of volumes defined in gitdrops.yaml that are NOT
// active on DO and therefore should be created.
func (vr *VolumeReconciler) ObjectsToUpdateAndCreate() {
	volumesToCreate := make([]dolocal.LocalVolumeCreateRequest, 0)
	volumeActionsByID := make(actionsByID)
	for _, localVolumeCreateRequest := range vr.localVolumeCreateRequests {
		volumeIsActive := false
		for _, activeVolume := range vr.activeVolumes {
			if localVolumeCreateRequest.Name == activeVolume.Name {
				//volume already exists, check for change in request
				log.Println("volume found check for change")
				// only do below check if delete privileges are granted
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

// ObjectToDelete populates VolumeReconciler with  a list of IDs for volumes that need
// to be deleted upon reconciliation of gitdrops.yaml (ie these volumes are active but not present
// in the spec)
func (vr *VolumeReconciler) ObjectsToDelete() {
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

func getVolumeActions(localVolumeCreateRequest dolocal.LocalVolumeCreateRequest, activeVolume godo.Volume) []action {
	var volumeActions []action
	if activeVolume.SizeGigaBytes != 0 && activeVolume.SizeGigaBytes != localVolumeCreateRequest.SizeGigaBytes {
		log.Println("volume (name)", activeVolume.Name, " (ID)", activeVolume.ID, " size has been updated in gitdrops.yaml")

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
		log.Println("volumeCreateRequest", volumeCreateRequest)
		err = dolocal.CreateVolume(ctx, vr.client, volumeCreateRequest)
		if err != nil {
			log.Println("error creating volume ", volumeToCreate.Name)
			return err
		}

	}
	return nil
}

func (vr *VolumeReconciler) UpdateObjects(ctx context.Context) error {
	for volumeID, volumeActions := range vr.volumesToUpdate {
		for _, volumeAction := range volumeActions {
			switch volumeAction.action {
			case resize:
				err := dolocal.ResizeVolume(ctx, vr.client, volumeID.(string), vr.findVolumeRegion(volumeID.(string)), volumeAction.value)
				if err != nil {
					log.Println("error during action request for volume ", volumeID, " error: ", err)
					// we do not return here as there may be more actions to complete
					// for this volume.
				}
			}
		}
	}
	return nil
}

func (vr *VolumeReconciler) findVolumeRegion(volID string) string {
	for _, vol := range vr.activeVolumes {
		if vol.ID == volID {
			return vol.Region.Slug
		}
	}
	return ""
}
