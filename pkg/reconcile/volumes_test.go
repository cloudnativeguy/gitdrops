package reconcile

import (
	"errors"
	"reflect"
	"testing"

	"github.com/nolancon/gitdrops/pkg/gitdrops"

	"github.com/digitalocean/godo"
)

func newTestVolumeReconciler(privileges gitdrops.Privileges, client *godo.Client, activeVolumes []godo.Volume, gitdropsVolumes []gitdrops.Volume) *volumeReconciler {
	return &volumeReconciler{
		privileges:      privileges,
		client:          client,
		activeVolumes:   activeVolumes,
		gitdropsVolumes: gitdropsVolumes,
	}
}

func TestSetVolumesToUpdateCreate(t *testing.T) {
	tcases := []struct {
		name            string
		activeVolumes   []godo.Volume
		gitdropsVolumes []gitdrops.Volume
		volumesToCreate []gitdrops.Volume
		volumesToUpdate actionsByID
		volumeNameToID  map[string]string
	}{
		{
			name: "test case 1",
			activeVolumes: []godo.Volume{
				{
					ID:   "abc",
					Name: "volume-1",
				},
				{
					ID:   "def",
					Name: "volume-2",
				},
				{
					ID:   "ghi",
					Name: "volume-3",
				},
			},
			gitdropsVolumes: []gitdrops.Volume{
				{
					Name: "volume-3",
				},
				{
					Name: "volume-4",
				},
				{
					Name: "volume-5",
				},
			},
			volumesToUpdate: make(actionsByID),
			volumesToCreate: []gitdrops.Volume{
				{
					Name: "volume-4",
				},
				{
					Name: "volume-5",
				},
			},
			volumeNameToID: make(map[string]string),
		},
		{
			name: "test case 2",
			activeVolumes: []godo.Volume{
				{
					ID:   "abc",
					Name: "volume-1",
				},
				{
					ID:   "def",
					Name: "volume-2",
				},
				{
					ID:   "ghi",
					Name: "volume-3",
				},
			},
			gitdropsVolumes: []gitdrops.Volume{

				{
					Name: "volume-1",
				},
				{
					Name: "volume-2",
				},
				{
					Name: "volume-3",
				},
			},

			volumesToUpdate: make(actionsByID),
			volumesToCreate: []gitdrops.Volume{},
			volumeNameToID:  make(map[string]string),
		},
		{
			name:          "test case 3",
			activeVolumes: []godo.Volume{},
			gitdropsVolumes: []gitdrops.Volume{
				{
					Name: "volume-1",
				},
				{
					Name: "volume-2",
				},
				{
					Name: "volume-3",
				},
			},
			volumesToUpdate: make(actionsByID),
			volumesToCreate: []gitdrops.Volume{
				{
					Name: "volume-1",
				},
				{
					Name: "volume-2",
				},
				{
					Name: "volume-3",
				},
			},
			volumeNameToID: make(map[string]string),
		},
		{
			name: "test case 4",
			activeVolumes: []godo.Volume{
				{
					ID:   "abc",
					Name: "volume-1",
				},
				{
					ID:            "def",
					Name:          "volume-2",
					SizeGigaBytes: 100,
				},
				{
					ID:   "ghi",
					Name: "volume-3",
				},
			},
			gitdropsVolumes: []gitdrops.Volume{
				{
					Name: "volume-1",
				},
				{
					Name:          "volume-2",
					SizeGigaBytes: 200,
				},
				{
					Name: "volume-4",
				},
			},
			volumesToUpdate: actionsByID{
				string("def"): []action{
					{
						action: "resize",
						value:  int64(200),
					},
				},
			},
			volumesToCreate: []gitdrops.Volume{
				{
					Name: "volume-4",
				},
			},
			volumeNameToID: make(map[string]string),
		},
	}
	for _, tc := range tcases {
		vr := newTestVolumeReconciler(gitdrops.Privileges{}, nil, tc.activeVolumes, tc.gitdropsVolumes)

		vr.setObjectsToUpdateAndCreate()
		if !reflect.DeepEqual(vr.volumesToUpdate, tc.volumesToUpdate) {
			t.Errorf("VolumesToUpdate - Failed %v, expected: %v, got %v", tc.name, tc.volumesToUpdate, vr.volumesToUpdate)
		}

		if !reflect.DeepEqual(vr.volumesToCreate, tc.volumesToCreate) {
			t.Errorf("VolumesToCreate - Failed %v, expected: %v, got %v", tc.name, tc.volumesToCreate, vr.volumesToCreate)
		}
	}
}

func TestSetVolumesToDelete(t *testing.T) {
	tcases := []struct {
		name            string
		activeVolumes   []godo.Volume
		gitdropsVolumes []gitdrops.Volume
		volumesToDelete []string
	}{
		{
			name: "test case 1",
			activeVolumes: []godo.Volume{
				{
					ID:   "abc",
					Name: "volume-1",
				},
				{
					ID:   "def",
					Name: "volume-2",
				},
				{
					ID:   "ghi",
					Name: "volume-3",
				},
			},

			gitdropsVolumes: []gitdrops.Volume{

				{
					Name: "volume-3",
				},
				{
					Name: "volume-4",
				},
				{
					Name: "volume-5",
				},
			},
			volumesToDelete: []string{"abc", "def"},
		},
		{
			name: "test case 2",
			activeVolumes: []godo.Volume{
				{
					ID:   "abc",
					Name: "volume-1",
				},
				{
					ID:   "def",
					Name: "volume-2",
				},
				{
					ID:   "ghi",
					Name: "volume-3",
				},
			},
			gitdropsVolumes: []gitdrops.Volume{
				{
					Name: "volume-1",
				},
				{
					Name: "volume-2",
				},
				{
					Name: "volume-3",
				},
			},
			volumesToDelete: []string{},
		},
		{
			name:          "test case 3",
			activeVolumes: []godo.Volume{},
			gitdropsVolumes: []gitdrops.Volume{
				{
					Name: "volume-1",
				},
				{
					Name: "volume-2",
				},
				{
					Name: "volume-3",
				},
			},
			volumesToDelete: []string{},
		},
		{
			name: "test case 4",
			activeVolumes: []godo.Volume{
				{
					ID:   "abc",
					Name: "volume-1",
					Region: &godo.Region{
						Name: "london",
					},
				},
				{
					ID:   "def",
					Name: "volume-2",
				},
				{
					ID:   "ghi",
					Name: "volume-3",
				},
			},
			gitdropsVolumes: []gitdrops.Volume{

				{
					Name:   "volume-1",
					Region: "nyc3",
				},
				{
					Name: "volume-2",
				},
				{
					Name: "volume-4",
				},
			},
			volumesToDelete: []string{"ghi"},
		},
	}
	for _, tc := range tcases {
		vr := newTestVolumeReconciler(gitdrops.Privileges{}, nil, tc.activeVolumes, tc.gitdropsVolumes)

		vr.setObjectsToDelete()
		if !reflect.DeepEqual(vr.volumesToDelete, tc.volumesToDelete) {
			t.Errorf("Failed %v, expected: %v, got %v", tc.name, tc.volumesToDelete, vr.volumesToDelete)
		}
	}
}

func TestTranslateVolumesCreateRequest(t *testing.T) {
	tcases := []struct {
		name                   string
		gitdropsVolume         gitdrops.Volume
		expVolumeCreateRequest *godo.VolumeCreateRequest
		expError               error
	}{
		{
			name: "test case 1 - no name",
			gitdropsVolume: gitdrops.Volume{
				Region:        "nyc3",
				SizeGigaBytes: 200,
			},
			expVolumeCreateRequest: &godo.VolumeCreateRequest{},
			expError:               errors.New(volumeNameErr),
		},
		{
			name: "test case 2 - no region",
			gitdropsVolume: gitdrops.Volume{
				Name:          "volume-1",
				SizeGigaBytes: 200,
				SnapshotID:    "test-id",
			},
			expVolumeCreateRequest: &godo.VolumeCreateRequest{},
			expError:               errors.New(volumeRegionErr),
		},
		{
			name: "test case 3 - no size",
			gitdropsVolume: gitdrops.Volume{
				Name:   "volume-1",
				Region: "nyc3",
			},
			expVolumeCreateRequest: &godo.VolumeCreateRequest{},
			expError:               errors.New(volumeSizeGigaBytesErr),
		},
		{
			name: "test case 3 - no error",
			gitdropsVolume: gitdrops.Volume{
				Name:           "volume-1",
				Region:         "nyc3",
				SizeGigaBytes:  200,
				SnapshotID:     "test-id",
				FilesystemType: "ext",
				Tags:           []string{"tag-1", "tag-2"},
			},
			expVolumeCreateRequest: &godo.VolumeCreateRequest{
				Region:         "nyc3",
				Name:           "volume-1",
				SizeGigaBytes:  200,
				SnapshotID:     "test-id",
				FilesystemType: "ext",
				Tags:           []string{"tag-1", "tag-2"},
			},
			expError: nil,
		},
	}
	for _, tc := range tcases {
		volumeCreateRequest, err := translateVolumeCreateRequest(tc.gitdropsVolume)
		if !reflect.DeepEqual(volumeCreateRequest, tc.expVolumeCreateRequest) {
			t.Errorf("Failed %v, expected: %v, got %v", tc.name, tc.expVolumeCreateRequest, volumeCreateRequest)
		}
		if err != nil {
			if err.Error() != tc.expError.Error() {
				t.Errorf("Failed %v, expected error : %v, got error %v", tc.name, tc.expError, err)
			}
		}
	}
}
