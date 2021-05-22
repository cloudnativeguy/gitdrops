package reconcile

import (
	"context"
	"log"
	"os"

	"github.com/nolancon/gitdrops/pkg/gitdrops"

	"github.com/digitalocean/godo"
)

const (
	resize            = "resize"
	rebuild           = "rebuild"
	attach            = "attach"
	detach            = "detach"
	digitaloceanToken = "DIGITALOCEAN_TOKEN"
)

type objectReconciler interface {
	reconcile(context.Context) error
	secondaryReconcile(context.Context, actionsByID) error
	setActiveObjects(context.Context) error
	setObjectsToUpdateAndCreate()
	setObjectsToDelete()
	getObjectsToUpdate() actionsByID
	deleteObjects(context.Context) error
	updateObjects(context.Context) error
	createObjects(context.Context) error
}

type Reconciler struct {
	volumeReconciler  objectReconciler
	dropletReconciler objectReconciler
}

// actionsByID is a slice of actions to be taken on the object. The ID is that of the object
// itself, we use an interface as the type (string/int) varies per object.
type actionsByID map[interface{}][]action

type action struct {
	// action is the name of the action to be taken eg resize, rebuild, attach, detach etc
	action string
	// value is eg '10GB' for resize or 'ubuntu-x-x' for rebuild etc
	value interface{}
}

func NewReconciler(ctx context.Context) (Reconciler, error) {
	gitDrops, err := gitdrops.ReadGitDrops()
	if err != nil {
		log.Println(err)
		return Reconciler{}, err
	}

	client := godo.NewFromToken(os.Getenv(digitaloceanToken))

	volumeReconciler := &volumeReconciler{
		privileges:      gitDrops.Privileges,
		client:          client,
		gitdropsVolumes: gitDrops.Volumes,
	}
	activeVolumes, err := gitdrops.ListVolumes(ctx, volumeReconciler.client)
	if err != nil {
		log.Println("Error while listing volumes", err)
		return Reconciler{}, err
	}

	volumeReconciler.activeVolumes = activeVolumes
	volumeReconciler.setObjectsToUpdateAndCreate()
	volumeReconciler.setObjectsToDelete()

	log.Println("active volumes:", len(activeVolumes))
	log.Println("active volumes to delete:", volumeReconciler.volumesToDelete)
	log.Println("gitdrops volumes to update:", volumeReconciler.volumesToUpdate)
	log.Println("gitdrops volumes to create:", volumeReconciler.volumesToCreate)

	dropletReconciler := &dropletReconciler{
		privileges:       gitDrops.Privileges,
		client:           client,
		gitdropsDroplets: gitDrops.Droplets,
	}
	err = dropletReconciler.setActiveObjects(ctx)
	if err != nil {
		return Reconciler{}, err
	}

	dropletReconciler.setObjectsToUpdateAndCreate()
	dropletReconciler.setObjectsToDelete()

	log.Println("active droplets:", len(activeVolumes))
	log.Println("active droplets to delete:", dropletReconciler.dropletsToDelete)
	log.Println("gitdrops droplets to update:", dropletReconciler.dropletsToUpdate)
	log.Println("gitdrops droplets to create:", dropletReconciler.dropletsToCreate)

	return Reconciler{
		volumeReconciler:  volumeReconciler,
		dropletReconciler: dropletReconciler,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	log.Println("begin initial volume reconciliation(create, resize)...")
	err := r.volumeReconciler.reconcile(ctx)
	if err != nil {
		return err
	}
	log.Println("initial volume reconciliation complete")
	log.Println("begin droplet reconciliation(create, delete, resize, rebuild)...")
	err = r.dropletReconciler.reconcile(ctx)
	if err != nil {
		return err
	}
	log.Println("droplet reconciliation complete")
	log.Println("begin secondary volume reconciliation (delete, attach, detach)...")

	// re-set active objects in the droplet reconciler because the volumes have been
	// reconciled and we need to search again for volume attach/detach actions.
	err = r.dropletReconciler.setActiveObjects(ctx)
	if err != nil {
		return err
	}
	// pass objects to update from droplet reconciler to volume reconciler
	// as they now contain actions for volumes to attac/detach from droplets
	// based on droplet reconciliation
	r.dropletReconciler.setObjectsToUpdateAndCreate()
	objectsToUpdate := r.dropletReconciler.getObjectsToUpdate()
	err = r.volumeReconciler.secondaryReconcile(ctx, objectsToUpdate)
	if err != nil {
		return err
	}
	log.Println("secondary volume reconciliation complete")
	return nil
}
