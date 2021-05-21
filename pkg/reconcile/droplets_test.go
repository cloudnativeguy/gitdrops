package reconcile

import (
	"github.com/nolancon/gitdrops/pkg/dolocal"

	"reflect"
	"testing"

	"github.com/digitalocean/godo"
)

func newTestDropletReconciler(privileges dolocal.Privileges, client *godo.Client, activeDroplets []godo.Droplet, localDropletCreateRequests []dolocal.LocalDropletCreateRequest, volumeNameToID map[string]string) *DropletReconciler {
	return &DropletReconciler{
		privileges:                 privileges,
		client:                     client,
		activeDroplets:             activeDroplets,
		localDropletCreateRequests: localDropletCreateRequests,
		volumeNameToID:             volumeNameToID,
	}
}

func TestSetDropletsToUpdateCreate(t *testing.T) {
	tcases := []struct {
		name                       string
		activeDroplets             []godo.Droplet
		localDropletCreateRequests []dolocal.LocalDropletCreateRequest
		dropletsToCreate           []dolocal.LocalDropletCreateRequest
		dropletsToUpdate           actionsByID
		volumeNameToID             map[string]string
	}{
		{
			name: "test case 1",
			activeDroplets: []godo.Droplet{
				godo.Droplet{
					ID:   1,
					Name: "droplet-1",
				},
				godo.Droplet{
					ID:   2,
					Name: "droplet-2",
				},
				godo.Droplet{
					ID:   3,
					Name: "droplet-3",
				},
			},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-3",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-4",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-5",
				},
			},
			dropletsToUpdate: make(actionsByID),
			dropletsToCreate: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-4",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-5",
				},
			},
			volumeNameToID: make(map[string]string),
		},
		{
			name: "test case 2",
			activeDroplets: []godo.Droplet{
				godo.Droplet{
					ID:   1,
					Name: "droplet-1",
				},
				godo.Droplet{
					ID:   2,
					Name: "droplet-2",
				},
				godo.Droplet{
					ID:   3,
					Name: "droplet-3",
				},
			},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{

				dolocal.LocalDropletCreateRequest{
					Name: "droplet-1",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-2",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-3",
				},
			},

			dropletsToUpdate: make(actionsByID),
			dropletsToCreate: []dolocal.LocalDropletCreateRequest{},
			volumeNameToID:   make(map[string]string),
		},
		{
			name:           "test case 3",
			activeDroplets: []godo.Droplet{},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-1",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-2",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-3",
				},
			},
			dropletsToUpdate: make(actionsByID),
			dropletsToCreate: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-1",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-2",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-3",
				},
			},
			volumeNameToID: make(map[string]string),
		},
		{
			name: "test case 4",
			activeDroplets: []godo.Droplet{
				godo.Droplet{
					ID:   1,
					Name: "droplet-1",
					Image: &godo.Image{
						Slug: "centos-8-x64",
					},
				},
				godo.Droplet{
					ID:   2,
					Name: "droplet-2",
					Image: &godo.Image{
						Slug: "centos-8-x64",
					},
					Size: &godo.Size{
						Slug: "s-1vcpu-1gb",
					},
				},
				godo.Droplet{
					ID:   3,
					Name: "droplet-3",
				},
			},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name:  "droplet-1",
					Image: "ubuntu-16-04-x64",
				},
				dolocal.LocalDropletCreateRequest{
					Name:  "droplet-2",
					Image: "ubuntu-16-04-x64",
					Size:  "s-1vcpu-2gb",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-4",
				},
			},
			dropletsToUpdate: actionsByID{
				1: []action{
					{
						action: "rebuild",
						value:  "ubuntu-16-04-x64",
					},
				},
				2: []action{
					{
						action: "resize",
						value:  "s-1vcpu-2gb",
					},
					{
						action: "rebuild",
						value:  "ubuntu-16-04-x64",
					},
				},
			},
			dropletsToCreate: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-4",
				},
			},
			volumeNameToID: make(map[string]string),
		},
		{
			name: "test case 5 - attach volume",
			activeDroplets: []godo.Droplet{
				godo.Droplet{
					ID:   2,
					Name: "droplet-2",
					Image: &godo.Image{
						Slug: "centos-8-x64",
					},
					Size: &godo.Size{
						Slug: "s-1vcpu-1gb",
					},
				},
			},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name:    "droplet-2",
					Image:   "ubuntu-16-04-x64",
					Size:    "s-1vcpu-2gb",
					Volumes: []string{"volume-1"},
				},
			},
			dropletsToUpdate: actionsByID{
				2: []action{
					{
						action: "resize",
						value:  "s-1vcpu-2gb",
					},
					{
						action: "rebuild",
						value:  "ubuntu-16-04-x64",
					},
					{
						action: "attach",
						value:  "abc",
					},
				},
			},
			dropletsToCreate: []dolocal.LocalDropletCreateRequest{},
			volumeNameToID: map[string]string{
				"volume-1": "abc",
				"volume-2": "def",
			},
		},
		{
			name: "test case 6 - attach volume and detach volume",
			activeDroplets: []godo.Droplet{
				godo.Droplet{
					ID:   1,
					Name: "droplet-1",
					Image: &godo.Image{
						Slug: "ubuntu-16-04-x64",
					},
					Size: &godo.Size{
						Slug: "s-1vcpu-2gb",
					},
					VolumeIDs: []string{
						"def",
					},
				},
				godo.Droplet{
					ID:   2,
					Name: "droplet-2",
					Image: &godo.Image{
						Slug: "centos-8-x64",
					},
					Size: &godo.Size{
						Slug: "s-1vcpu-2gb",
					},
					VolumeIDs: []string{
						"abc",
					},
				},
			},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name:    "droplet-1",
					Image:   "ubuntu-16-04-x64",
					Size:    "s-1vcpu-2gb",
					Volumes: []string{"volume-1"},
				},
				dolocal.LocalDropletCreateRequest{
					Name:    "droplet-2",
					Image:   "centos-8-x64",
					Size:    "s-1vcpu-2gb",
					Volumes: []string{"volume-2"},
				},
			},
			dropletsToUpdate: actionsByID{
				1: []action{
					{
						action: "detach",
						value:  "def",
					},

					{
						action: "attach",
						value:  "abc",
					},
				},

				2: []action{
					{
						action: "detach",
						value:  "abc",
					},
					{
						action: "attach",
						value:  "def",
					},
				},
			},
			dropletsToCreate: []dolocal.LocalDropletCreateRequest{},
			volumeNameToID: map[string]string{
				"volume-1": "abc",
				"volume-2": "def",
			},
		},
		{
			name: "test case 7 - no action on attach/detach",
			activeDroplets: []godo.Droplet{
				godo.Droplet{
					ID:   2,
					Name: "droplet-2",
					Image: &godo.Image{
						Slug: "centos-8-x64",
					},
					Size: &godo.Size{
						Slug: "s-1vcpu-1gb",
					},
					VolumeIDs: []string{
						"abc",
					},
				},
			},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name:    "droplet-2",
					Image:   "centos-8-x64",
					Size:    "s-1vcpu-1gb",
					Volumes: []string{"volume-1"},
				},
			},
			dropletsToUpdate: make(actionsByID),
			dropletsToCreate: []dolocal.LocalDropletCreateRequest{},
			volumeNameToID: map[string]string{
				"volume-1": "abc",
			},
		},
	}
	for _, tc := range tcases {
		dr := newTestDropletReconciler(dolocal.Privileges{}, nil, tc.activeDroplets, tc.localDropletCreateRequests, tc.volumeNameToID)

		dr.SetObjectsToUpdateAndCreate()
		if !reflect.DeepEqual(dr.dropletsToUpdate, tc.dropletsToUpdate) {
			t.Errorf("DropletsToUpdate - Failed %v, expected: %v, got %v", tc.name, tc.dropletsToUpdate, dr.dropletsToUpdate)
		}
		if !reflect.DeepEqual(dr.dropletsToCreate, tc.dropletsToCreate) {
			t.Errorf("DropletsToCreate - Failed %v, expected: %v, got %v", tc.name, tc.dropletsToCreate, dr.dropletsToCreate)
		}

	}
}

func TestActiveDropletsToDelete(t *testing.T) {
	tcases := []struct {
		name                       string
		activeDroplets             []godo.Droplet
		localDropletCreateRequests []dolocal.LocalDropletCreateRequest
		dropletsToDelete           []int
	}{
		{
			name: "test case 1",
			activeDroplets: []godo.Droplet{
				godo.Droplet{
					ID:   1,
					Name: "droplet-1",
				},
				godo.Droplet{
					ID:   2,
					Name: "droplet-2",
				},
				godo.Droplet{
					ID:   3,
					Name: "droplet-3",
				},
			},

			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{

				dolocal.LocalDropletCreateRequest{
					Name: "droplet-3",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-4",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-5",
				},
			},
			dropletsToDelete: []int{1, 2},
		},
		{
			name: "test case 2",
			activeDroplets: []godo.Droplet{
				godo.Droplet{
					ID:   1,
					Name: "droplet-1",
				},
				godo.Droplet{
					ID:   2,
					Name: "droplet-2",
				},
				godo.Droplet{
					ID:   3,
					Name: "droplet-3",
				},
			},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-1",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-2",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-3",
				},
			},
			dropletsToDelete: []int{},
		},
		{
			name:           "test case 3",
			activeDroplets: []godo.Droplet{},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-1",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-2",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-3",
				},
			},
			dropletsToDelete: []int{},
		},
		{
			name: "test case 4",
			activeDroplets: []godo.Droplet{
				godo.Droplet{
					ID:   1,
					Name: "droplet-1",
					Region: &godo.Region{
						Name: "london",
					},
				},
				godo.Droplet{
					ID:   2,
					Name: "droplet-2",
				},
				godo.Droplet{
					ID:   3,
					Name: "droplet-3",
				},
			},
			localDropletCreateRequests: []dolocal.LocalDropletCreateRequest{

				dolocal.LocalDropletCreateRequest{
					Name:   "droplet-1",
					Region: "nyc3",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-2",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-4",
				},
			},
			dropletsToDelete: []int{3},
		},
	}
	for _, tc := range tcases {
		dr := newTestDropletReconciler(dolocal.Privileges{}, nil, tc.activeDroplets, tc.localDropletCreateRequests, nil)

		dr.SetObjectsToDelete()
		if !reflect.DeepEqual(dr.dropletsToDelete, tc.dropletsToDelete) {
			t.Errorf("Failed %v, expected: %v, got %v", tc.name, tc.dropletsToDelete, dr.dropletsToDelete)
		}

	}
}
