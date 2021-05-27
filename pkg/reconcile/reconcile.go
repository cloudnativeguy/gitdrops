package reconcile

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

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
	setActiveObjects(context.Context) error
	setObjectsToUpdateAndCreate()
	setObjectsToDelete()
	getActiveObjects() interface{}
	getObjectsToCreate() interface{}
	getObjectsToUpdate() actionsByID
	getObjectsToDelete() interface{}
	reconcileObjectsToCreate(context.Context) error
	// reconcileObjectsToUpdate can optionally be passed actions from another reconciler
	// eg attach/detach from dropletReconciler to volumeReconciler
	reconcileObjectsToUpdate(context.Context, actionsByID) error
	reconcileObjectsToDelete(context.Context) error
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
		return Reconciler{}, fmt.Errorf("NewReconciler: %v", err)
	}

	client := godo.NewFromToken(os.Getenv(digitaloceanToken))

	volumeReconciler := &volumeReconciler{
		privileges:      gitDrops.Privileges,
		client:          client,
		gitdropsVolumes: gitDrops.Volumes,
	}

	dropletReconciler := &dropletReconciler{
		privileges:       gitDrops.Privileges,
		client:           client,
		gitdropsDroplets: gitDrops.Droplets,
	}
	return Reconciler{
		volumeReconciler:  volumeReconciler,
		dropletReconciler: dropletReconciler,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.volumeReconciler.setActiveObjects(ctx)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}
	r.volumeReconciler.setObjectsToUpdateAndCreate()

	err = r.volumeReconciler.reconcileObjectsToCreate(ctx)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}
	err = r.volumeReconciler.reconcileObjectsToUpdate(ctx, nil)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}

	// wait 10 seconds to allow any volume creation before droplet creation,
	// also allow time for volumes to attach detach
	log.Println("Reconcile: waiting for volumes to update before reconciling droplets (10s)...")
	time.Sleep(10 * time.Second)

	// re-set active objects in the droplet reconciler because the volumes have been
	// reconciled and we need to search again for volume attach/detach actions.
	err = r.dropletReconciler.setActiveObjects(ctx)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}
	r.dropletReconciler.setObjectsToUpdateAndCreate()
	err = r.dropletReconciler.reconcileObjectsToCreate(ctx)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}
	err = r.dropletReconciler.reconcileObjectsToUpdate(ctx, nil)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}

	r.dropletReconciler.setObjectsToDelete()
	err = r.dropletReconciler.reconcileObjectsToDelete(ctx)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}

	// pass objects to update from droplet reconciler to volume reconciler
	// as they now contain actions for volumes to attac/detach from droplets
	// based on droplet reconciliation
	err = r.volumeReconciler.setActiveObjects(ctx)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}
	objectsToUpdate := r.dropletReconciler.getObjectsToUpdate()
	err = r.volumeReconciler.reconcileObjectsToUpdate(ctx, objectsToUpdate)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}
	r.volumeReconciler.setObjectsToDelete()
	err = r.volumeReconciler.reconcileObjectsToDelete(ctx)
	if err != nil {
		return fmt.Errorf("Reconcile: %v", err)
	}

	return nil
}
