package reconcile

import (
	"context"
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

type ObjectReconciler interface {
}

type Reconciler struct {
	privileges        dolocal.Privileges
	volumeReconciler  *VolumeReconciler
	dropletReconciler *DropletReconciler
}

type VolumeReconciler struct {
	client                    *godo.Client
	activeVolumes             []godo.Volume
	localVolumeCreateRequests []dolocal.LocalVolumeCreateRequest
	volumesToCreate           []dolocal.LocalVolumeCreateRequest
	volumesToUpdate           actionsByID
	volumesToDelete           []int
}

type actionsByID map[int][]action

type action struct {
	object string
	action string
	value  string
}

func NewReconciler(ctx context.Context) (Reconciler, error) {
	gitDrops, err := dolocal.ReadGitDrops()
	if err != nil {
		log.Println(err)
		return Reconciler{}, err
	}

	client := godo.NewFromToken(os.Getenv(digitaloceanToken))

	//	activeVolumes, err := dolocal.ListVolumes(ctx, client)
	//	if err != nil {
	//		log.Println("Error while listing volumes", err)
	//		return Reconciler{}, err
	//	}

	//	log.Println("active volumes on digitalocean:", len(activeVolumes))
	//	volumeReconciler := &VolumeReconciler{
	//		client:                    client,
	//		activeVolumes:             activeVolumes,
	//		localVolumeCreateRequests: gitDrops.Volumes,
	//	}

	dropletReconciler := &DropletReconciler{
		client:                     client,
		localDropletCreateRequests: gitDrops.Droplets,
	}
	err = dropletReconciler.Populate(ctx)

	return Reconciler{
		privileges: gitDrops.Privileges,
		//		volumeReconciler:  volumeReconciler,
		dropletReconciler: dropletReconciler,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.privileges.Delete {
		err := r.dropletReconciler.DeleteObjects(ctx)
		if err != nil {
			log.Println("error deleting droplet")
			return err
		}
	} else {
		log.Println("gitdrops.yaml does not have delete privileges")
	}

	if r.privileges.Create {
		err := r.dropletReconciler.CreateObjects(ctx)
		if err != nil {
			log.Println("error creating droplet")
			return err
		}
	} else {
		log.Println("gitdrops.yaml does not have create privileges")
	}
	if r.privileges.Update {
		err := r.dropletReconciler.UpdateObjects(ctx)
		if err != nil {
			log.Println("error updating droplet")
			return err
		}
	} else {
		log.Println("gitdrops.yaml does not have update privileges")
	}

	return nil
}
