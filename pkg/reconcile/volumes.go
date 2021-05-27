package reconcile

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/nolancon/gitdrops/pkg/gitdrops"

	"github.com/digitalocean/godo"
)

const (
	volumeNameErr          = "translateVolumeCreateRequest: volume name not specified"
	volumeRegionErr        = "translateVolumeCreateRequest: volume region not specified"
	volumeSizeGigaBytesErr = "translateVolumeCreateRequest: volume sizeGigaBytes not specified"
)

type volumeReconciler struct {
	privileges      gitdrops.Privileges
	client          *godo.Client
	activeVolumes   []godo.Volume
	gitdropsVolumes []gitdrops.Volume
	volumesToCreate []gitdrops.Volume
	volumesToUpdate actionsByID
	volumesToDelete []string
}

var _ objectReconciler = &volumeReconciler{}

func (vr *volumeReconciler) reconcile(ctx context.Context) error {
	if len(vr.volumesToCreate) != 0 {
		if vr.privileges.Create {
			err := vr.createObjects(ctx)
			if err != nil {
				return fmt.Errorf("volumeReconciler.reconcile: %v", err)
			}
		} else {
			log.Println("gitdrops discovered volumes to create, but does not have create privileges")
		}
	}
	if len(vr.volumesToUpdate) != 0 {
		if vr.privileges.Update {
			err := vr.updateObjects(ctx)
			if err != nil {
				return fmt.Errorf("volumeReconciler.reconcile: %v", err)
			}
		} else {
			log.Println("gitdrops discovered volumes to update, but does not have update privileges")
		}
	}
	return nil
}

func (vr *volumeReconciler) setActiveObjects(ctx context.Context) error {
	activeVolumes, err := gitdrops.ListVolumes(ctx, vr.client)
	if err != nil {
		return fmt.Errorf("volumeReconciler.setActiveObjects: %v", err)
	}
	vr.activeVolumes = activeVolumes
	return nil
}

func (vr *volumeReconciler) secondaryReconcile(ctx context.Context, objectsToUpdate actionsByID) error {
	if len(vr.volumesToDelete) != 0 {
		if vr.privileges.Delete {
			err := vr.deleteObjects(ctx)
			if err != nil {
				return fmt.Errorf("volumeReconciler.secondaryReconcile: %v", err)
			}
		} else {
			log.Println("gitdrops discovered volumes to delete, but does not have delete privileges")
		}
	}
	err := vr.setActiveObjects(ctx)
	if err != nil {
		return fmt.Errorf("volumeReconciler.secondaryReconcile: %v", err)
	}

	vr.volumesToUpdate = objectsToUpdate
	err = vr.updateObjects(ctx)
	if err != nil {
		return fmt.Errorf("volumeReconciler.secondaryReconcile: %v", err)
	}

	return nil
}

func translateVolumeCreateRequest(gitdropsVolume gitdrops.Volume) (*godo.VolumeCreateRequest, error) {
	createRequest := &godo.VolumeCreateRequest{}
	if gitdropsVolume.Name == "" {
		return createRequest, errors.New(volumeNameErr)
	}
	if gitdropsVolume.Region == "" {
		return createRequest, errors.New(volumeRegionErr)
	}
	if gitdropsVolume.SizeGigaBytes == 0 {
		return createRequest, errors.New(volumeSizeGigaBytesErr)
	}
	createRequest.Name = gitdropsVolume.Name
	createRequest.Region = gitdropsVolume.Region
	createRequest.SizeGigaBytes = gitdropsVolume.SizeGigaBytes
	createRequest.SnapshotID = gitdropsVolume.SnapshotID
	createRequest.FilesystemType = gitdropsVolume.FilesystemType
	createRequest.FilesystemLabel = gitdropsVolume.FilesystemLabel

	if gitdropsVolume.Tags != nil {
		createRequest.Tags = gitdropsVolume.Tags
	}

	return createRequest, nil
}

// SetObjectsToUpdateCreate populates VolumeReconciler with two lists:
// * volumesToUpdate: volumeActionsByID of volumes that are active on DO and are defined in
// gitdrops.yaml, but the active volumes are no longer in sync with the local gitdrops version.
// * volumesToCreate: Volumes of volumes defined in gitdrops.yaml that are NOT
// active on DO and therefore should be created.
func (vr *volumeReconciler) setObjectsToUpdateAndCreate() {
	volumesToCreate := make([]gitdrops.Volume, 0)
	volumeActionsByID := make(actionsByID)
	for _, gitdropsVolume := range vr.gitdropsVolumes {
		volumeIsActive := false
		for _, activeVolume := range vr.activeVolumes {
			if gitdropsVolume.Name == activeVolume.Name {
				//volume already exists, check for change in request
				volumeActions := getVolumeActions(gitdropsVolume, activeVolume)
				if len(volumeActions) != 0 {
					volumeActionsByID[activeVolume.ID] = volumeActions
				}
				volumeIsActive = true
				continue
			}
		}
		if !volumeIsActive {
			//create volume from local request
			volumesToCreate = append(volumesToCreate, gitdropsVolume)
		}
	}
	vr.volumesToUpdate = volumeActionsByID
	vr.volumesToCreate = volumesToCreate
}

// SetObjectToDelete populates VolumeReconciler with  a list of IDs for volumes that need
// to be deleted upon reconciliation of gitdrops.yaml (ie these volumes are active but not present
// in the spec)
func (vr *volumeReconciler) setObjectsToDelete() {
	volumesToDelete := make([]string, 0)

	for _, activeVolume := range vr.activeVolumes {
		activeVolumeInSpec := false
		for _, gitdropsVolume := range vr.gitdropsVolumes {
			if gitdropsVolume.Name == activeVolume.Name {
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

func (vr *volumeReconciler) getObjectsToUpdate() actionsByID {
	return vr.volumesToUpdate
}

func getVolumeActions(gitdropsVolume gitdrops.Volume, activeVolume godo.Volume) []action {
	var volumeActions []action
	if activeVolume.SizeGigaBytes != 0 && activeVolume.SizeGigaBytes != gitdropsVolume.SizeGigaBytes {
		log.Println("volume", activeVolume.Name, "size has been updated in gitdrops.yaml")
		volumeAction := action{
			action: resize,
			value:  gitdropsVolume.SizeGigaBytes,
		}
		volumeActions = append(volumeActions, volumeAction)
	}
	return volumeActions
}

func (vr *volumeReconciler) deleteObjects(ctx context.Context) error {
	for _, id := range vr.volumesToDelete {
		err := gitdrops.DeleteVolume(ctx, vr.client, id)
		if err != nil {
			return fmt.Errorf("volumeReconciler.deleteObjects: %v", err)
		}
	}
	return nil
}

func (vr *volumeReconciler) createObjects(ctx context.Context) error {
	for _, volumeToCreate := range vr.volumesToCreate {
		volumeCreateRequest, err := translateVolumeCreateRequest(volumeToCreate)
		if err != nil {
			return fmt.Errorf("volumeReconciler.createObjects: %v", err)
		}
		err = gitdrops.CreateVolume(ctx, vr.client, volumeCreateRequest)
		if err != nil {
			return fmt.Errorf("volumeReconciler.createObjects: %v", err)
		}
	}
	return nil
}

func (vr *volumeReconciler) updateObjects(ctx context.Context) error {
	for id, volumeActions := range vr.volumesToUpdate {
		for _, volumeAction := range volumeActions {
			switch volumeAction.action {
			case resize:
				err := gitdrops.ResizeVolume(ctx, vr.client, id.(string), vr.findVolumeRegion(id.(string)), volumeAction.value)
				if err != nil {
					return fmt.Errorf("volumeReconciler.updateObjects (resize): %v", err)
				}
			case attach:
				// in this case, 'id' is that of the droplet and 'value' is the volume
				// name. This is because this action was detected and created by the
				// droplet reconciler.
				err := gitdrops.AttachVolume(ctx, vr.client, volumeAction.value.(string), id.(int))
				if err != nil {
					return fmt.Errorf("volumeReconciler.updateObjects (attach): %v", err)
				}
			case detach:
				// in this case, 'id' is that of the droplet and 'value' is the volume
				// id. This is because this action was detected and created by the
				// droplet reconciler.
				err := gitdrops.DetachVolume(ctx, vr.client, volumeAction.value.(string), id.(int))
				if err != nil {
					return fmt.Errorf("volumeReconciler.updateObjects (detach): %v", err)
				}
			}
		}
	}
	return nil
}

func (vr *volumeReconciler) findVolumeRegion(volID string) string {
	for _, vol := range vr.activeVolumes {
		if vol.ID == volID {
			return vol.Region.Slug
		}
	}
	return ""
}
