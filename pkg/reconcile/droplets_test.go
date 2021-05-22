package reconcile

import (
	"github.com/nolancon/gitdrops/pkg/gitdrops"

	"reflect"
	"testing"

	"github.com/digitalocean/godo"
)

func newTestDropletReconciler(privileges gitdrops.Privileges, client *godo.Client, activeDroplets []godo.Droplet, gitdropsDroplets []gitdrops.Droplet, volumeNameToID map[string]string) *dropletReconciler {
	return &dropletReconciler{
		privileges:       privileges,
		client:           client,
		activeDroplets:   activeDroplets,
		gitdropsDroplets: gitdropsDroplets,
		volumeNameToID:   volumeNameToID,
	}
}

func TestSetDropletsToUpdateCreate(t *testing.T) {
	tcases := []struct {
		name             string
		activeDroplets   []godo.Droplet
		gitdropsDroplets []gitdrops.Droplet
		dropletsToCreate []gitdrops.Droplet
		dropletsToUpdate actionsByID
		volumeNameToID   map[string]string
	}{
		{
			name: "test case 1",
			activeDroplets: []godo.Droplet{
				{
					ID:   1,
					Name: "droplet-1",
				},
				{
					ID:   2,
					Name: "droplet-2",
				},
				{
					ID:   3,
					Name: "droplet-3",
				},
			},
			gitdropsDroplets: []gitdrops.Droplet{
				{
					Name: "droplet-3",
				},
				{
					Name: "droplet-4",
				},
				{
					Name: "droplet-5",
				},
			},
			dropletsToUpdate: make(actionsByID),
			dropletsToCreate: []gitdrops.Droplet{
				{
					Name: "droplet-4",
				},
				{
					Name: "droplet-5",
				},
			},
			volumeNameToID: make(map[string]string),
		},
		{
			name: "test case 2",
			activeDroplets: []godo.Droplet{
				{
					ID:   1,
					Name: "droplet-1",
				},
				{
					ID:   2,
					Name: "droplet-2",
				},
				{
					ID:   3,
					Name: "droplet-3",
				},
			},
			gitdropsDroplets: []gitdrops.Droplet{

				{
					Name: "droplet-1",
				},
				{
					Name: "droplet-2",
				},
				{
					Name: "droplet-3",
				},
			},

			dropletsToUpdate: make(actionsByID),
			dropletsToCreate: []gitdrops.Droplet{},
			volumeNameToID:   make(map[string]string),
		},
		{
			name:           "test case 3",
			activeDroplets: []godo.Droplet{},
			gitdropsDroplets: []gitdrops.Droplet{
				{
					Name: "droplet-1",
				},
				{
					Name: "droplet-2",
				},
				{
					Name: "droplet-3",
				},
			},
			dropletsToUpdate: make(actionsByID),
			dropletsToCreate: []gitdrops.Droplet{
				{
					Name: "droplet-1",
				},
				{
					Name: "droplet-2",
				},
				{
					Name: "droplet-3",
				},
			},
			volumeNameToID: make(map[string]string),
		},
		{
			name: "test case 4",
			activeDroplets: []godo.Droplet{
				{
					ID:   1,
					Name: "droplet-1",
					Image: &godo.Image{
						Slug: "centos-8-x64",
					},
				},
				{
					ID:   2,
					Name: "droplet-2",
					Image: &godo.Image{
						Slug: "centos-8-x64",
					},
					Size: &godo.Size{
						Slug: "s-1vcpu-1gb",
					},
				},
				{
					ID:   3,
					Name: "droplet-3",
				},
			},
			gitdropsDroplets: []gitdrops.Droplet{
				{
					Name:  "droplet-1",
					Image: "ubuntu-16-04-x64",
				},
				{
					Name:  "droplet-2",
					Image: "ubuntu-16-04-x64",
					Size:  "s-1vcpu-2gb",
				},
				{
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
			dropletsToCreate: []gitdrops.Droplet{
				{
					Name: "droplet-4",
				},
			},
			volumeNameToID: make(map[string]string),
		},
		{
			name: "test case 5 - attach volume",
			activeDroplets: []godo.Droplet{
				{
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
			gitdropsDroplets: []gitdrops.Droplet{
				{
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
			dropletsToCreate: []gitdrops.Droplet{},
			volumeNameToID: map[string]string{
				"volume-1": "abc",
				"volume-2": "def",
			},
		},
		{
			name: "test case 6 - attach volume and detach volume",
			activeDroplets: []godo.Droplet{
				{
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
				{
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
			gitdropsDroplets: []gitdrops.Droplet{
				{
					Name:    "droplet-1",
					Image:   "ubuntu-16-04-x64",
					Size:    "s-1vcpu-2gb",
					Volumes: []string{"volume-1"},
				},
				{
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
			dropletsToCreate: []gitdrops.Droplet{},
			volumeNameToID: map[string]string{
				"volume-1": "abc",
				"volume-2": "def",
			},
		},
		{
			name: "test case 7 - no action on attach/detach",
			activeDroplets: []godo.Droplet{
				{
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
			gitdropsDroplets: []gitdrops.Droplet{
				{
					Name:    "droplet-2",
					Image:   "centos-8-x64",
					Size:    "s-1vcpu-1gb",
					Volumes: []string{"volume-1"},
				},
			},
			dropletsToUpdate: make(actionsByID),
			dropletsToCreate: []gitdrops.Droplet{},
			volumeNameToID: map[string]string{
				"volume-1": "abc",
			},
		},
	}
	for _, tc := range tcases {
		dr := newTestDropletReconciler(gitdrops.Privileges{}, nil, tc.activeDroplets, tc.gitdropsDroplets, tc.volumeNameToID)

		dr.setObjectsToUpdateAndCreate()
		if !reflect.DeepEqual(dr.dropletsToUpdate, tc.dropletsToUpdate) {
			t.Errorf("DropletsToUpdate - Failed %v, expected: %v, got %v", tc.name, tc.dropletsToUpdate, dr.dropletsToUpdate)
		}
		if !reflect.DeepEqual(dr.dropletsToCreate, tc.dropletsToCreate) {
			t.Errorf("DropletsToCreate - Failed %v, expected: %v, got %v", tc.name, tc.dropletsToCreate, dr.dropletsToCreate)
		}
	}
}

func TestSetDropletsToDelete(t *testing.T) {
	tcases := []struct {
		name             string
		activeDroplets   []godo.Droplet
		gitdropsDroplets []gitdrops.Droplet
		dropletsToDelete []int
	}{
		{
			name: "test case 1",
			activeDroplets: []godo.Droplet{
				{
					ID:   1,
					Name: "droplet-1",
				},
				{
					ID:   2,
					Name: "droplet-2",
				},
				{
					ID:   3,
					Name: "droplet-3",
				},
			},

			gitdropsDroplets: []gitdrops.Droplet{

				{
					Name: "droplet-3",
				},
				{
					Name: "droplet-4",
				},
				{
					Name: "droplet-5",
				},
			},
			dropletsToDelete: []int{1, 2},
		},
		{
			name: "test case 2",
			activeDroplets: []godo.Droplet{
				{
					ID:   1,
					Name: "droplet-1",
				},
				{
					ID:   2,
					Name: "droplet-2",
				},
				{
					ID:   3,
					Name: "droplet-3",
				},
			},
			gitdropsDroplets: []gitdrops.Droplet{
				{
					Name: "droplet-1",
				},
				{
					Name: "droplet-2",
				},
				{
					Name: "droplet-3",
				},
			},
			dropletsToDelete: []int{},
		},
		{
			name:           "test case 3",
			activeDroplets: []godo.Droplet{},
			gitdropsDroplets: []gitdrops.Droplet{
				{
					Name: "droplet-1",
				},
				{
					Name: "droplet-2",
				},
				{
					Name: "droplet-3",
				},
			},
			dropletsToDelete: []int{},
		},
		{
			name: "test case 4",
			activeDroplets: []godo.Droplet{
				{
					ID:   1,
					Name: "droplet-1",
					Region: &godo.Region{
						Name: "london",
					},
				},
				{
					ID:   2,
					Name: "droplet-2",
				},
				{
					ID:   3,
					Name: "droplet-3",
				},
			},
			gitdropsDroplets: []gitdrops.Droplet{

				{
					Name:   "droplet-1",
					Region: "nyc3",
				},
				{
					Name: "droplet-2",
				},
				{
					Name: "droplet-4",
				},
			},
			dropletsToDelete: []int{3},
		},
	}
	for _, tc := range tcases {
		dr := newTestDropletReconciler(gitdrops.Privileges{}, nil, tc.activeDroplets, tc.gitdropsDroplets, nil)

		dr.setObjectsToDelete()
		if !reflect.DeepEqual(dr.dropletsToDelete, tc.dropletsToDelete) {
			t.Errorf("Failed %v, expected: %v, got %v", tc.name, tc.dropletsToDelete, dr.dropletsToDelete)
		}
	}
}
