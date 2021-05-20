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
	attach            = "attach"
	detach            = "detach"
	digitaloceanToken = "DIGITALOCEAN_TOKEN"
)

type ObjectReconciler interface {
	Populate(context.Context) error
	Reconcile(context.Context) error
	ReconcilePeripherals(context.Context, actionsByID) error
	SetObjectsToUpdateAndCreate()
	SetObjectsToDelete()
	GetObjectsToUpdate() actionsByID
	DeleteObjects(context.Context) error
	UpdateObjects(context.Context) error
	CreateObjects(context.Context) error
}

type Reconciler struct {
	volumeReconciler  ObjectReconciler
	dropletReconciler ObjectReconciler
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
	gitDrops, err := dolocal.ReadGitDrops()
	if err != nil {
		log.Println(err)
		return Reconciler{}, err
	}

	client := godo.NewFromToken(os.Getenv(digitaloceanToken))

	volumeReconciler := &VolumeReconciler{
		privileges:                gitDrops.Privileges,
		client:                    client,
		localVolumeCreateRequests: gitDrops.Volumes,
	}
	err = volumeReconciler.Populate(ctx)
	if err != nil {
		log.Println(err)
		return Reconciler{}, err
	}

	dropletReconciler := &DropletReconciler{
		privileges:                 gitDrops.Privileges,
		client:                     client,
		localDropletCreateRequests: gitDrops.Droplets,
	}
	err = dropletReconciler.Populate(ctx)
	if err != nil {
		log.Println(err)
		return Reconciler{}, err
	}

	return Reconciler{
		volumeReconciler:  volumeReconciler,
		dropletReconciler: dropletReconciler,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.volumeReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	err = r.dropletReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	objectsToUpdate := r.dropletReconciler.GetObjectsToUpdate()
	err = r.volumeReconciler.ReconcilePeripherals(ctx, objectsToUpdate)
	if err != nil {
		return err
	}
	return nil
}
