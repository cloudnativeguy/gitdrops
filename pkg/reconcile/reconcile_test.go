package reconcile

import (
	"github.com/nolancon/gitdrops/pkg/dolocal"

	"reflect"
	"testing"

	"github.com/digitalocean/godo"
)

func newTestReconcileDroplets(client *godo.Client, activeDroplets []godo.Droplet, localDropletCreateRequests []dolocal.LocalDropletCreateRequest) *ReconcileDroplets {
	return &ReconcileDroplets{
		client:                     client,
		localDropletCreateRequests: localDropletCreateRequests,
		activeDroplets:             activeDroplets,
	}
}

func TestDropletsToUpdateCreate(t *testing.T) {
	tcases := []struct {
		name                       string
		activeDroplets             []godo.Droplet
		localDropletCreateRequests []dolocal.LocalDropletCreateRequest
		dropletsToUpdate           dropletActionsByID
		dropletsToCreate           []dolocal.LocalDropletCreateRequest
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
			dropletsToUpdate: make(dropletActionsByID),
			dropletsToCreate: []dolocal.LocalDropletCreateRequest{
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-4",
				},
				dolocal.LocalDropletCreateRequest{
					Name: "droplet-5",
				},
			},
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
			dropletsToUpdate: make(dropletActionsByID),
			dropletsToCreate: []dolocal.LocalDropletCreateRequest{},
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
			dropletsToUpdate: make(dropletActionsByID),
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
			dropletsToUpdate: dropletActionsByID{
				1: []dropletAction{
					{
						action: "rebuild",
						value:  "ubuntu-16-04-x64",
					},
				},
				2: []dropletAction{
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
		},
	}
	for _, tc := range tcases {
		rd := newTestReconcileDroplets(nil, tc.activeDroplets, tc.localDropletCreateRequests)

		dropletsToUpdate, dropletsToCreate := rd.dropletsToUpdateCreate()
		if !reflect.DeepEqual(dropletsToUpdate, tc.dropletsToUpdate) {
			t.Errorf("DropletsToUpdate - Failed %v, expected: %v, got %v", tc.name, tc.dropletsToUpdate, dropletsToUpdate)
		}
		if !reflect.DeepEqual(dropletsToCreate, tc.dropletsToCreate) {
			t.Errorf("DropletsToCreate - Failed %v, expected: %v, got %v", tc.name, tc.dropletsToCreate, dropletsToCreate)
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
		rd := newTestReconcileDroplets(nil, tc.activeDroplets, tc.localDropletCreateRequests)

		dropletsToDelete := rd.activeDropletsToDelete()
		if !reflect.DeepEqual(dropletsToDelete, tc.dropletsToDelete) {
			t.Errorf("Failed %v, expected: %v, got %v", tc.name, tc.dropletsToDelete, dropletsToDelete)
		}

	}
}
