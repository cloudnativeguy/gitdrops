package gitdrops

type GitDrops struct {
	Privileges Privileges `yaml:"privileges"`
	Droplets   []Droplet  `yaml:"droplets"`
	Volumes    []Volume   `yaml:"volumes"`
}

type Privileges struct {
	Create bool `yaml:"create"`
	Update bool `yaml:"update"`
	Delete bool `yaml:"delete"`
}

// Droplet is a simplified gitdrops representation of godo.DropletCreateRequest
type Droplet struct {
	Name   string `yaml:"name"`
	Region string `yaml:"region"`
	Size   string `yaml:"size"`
	// Image represents the image name for the droplet. It is the equivalient of
	// godo.DropletCreateRequest.Image.Slug
	Image string `yaml:"image"`
	// SSHKeyFingerprint represents the SSH key fingerprints for the droplet.
	// It is the equivalient of godo.DropletCreateRequest.[]SSHKeys.FingerPrint
	SSHKeyFingerprints []string `yaml:"sshKeyFingerprints"`
	Backups            bool     `yaml:"backups"`
	IPv6               bool     `yaml:"ipv6"`
	Monitoring         bool     `yaml:"monitoring"`
	// See type UserData
	UserData UserData `yaml:"userData,omitempty"`
	// Volumes is a []string of the volume names to be attached to the droplet.
	Volumes []string `yaml:"volumes,omitempty"`
	Tags    []string `yaml:"tags"`
	VPCUUID string   `yaml:"vpcuuid,omitempty"`
}

// Volume is a simplified gitdrops representation of godo.VolumeCreateRequest
type Volume struct {
	Name            string   `yaml:"name"`
	Region          string   `yaml:"region"`
	SizeGigaBytes   int64    `yaml:"sizeGigaBytes"`
	SnapshotID      string   `yaml:"snapShotID"`
	FilesystemType  string   `yaml:"filesystemType"`
	FilesystemLabel string   `yaml:"filesystemLabel"`
	Tags            []string `yaml:"tags"`
}

// UserData stores the Path of a userdata file and/or the Data itself. In the event that path is
// defined, Data is populated with contents of the file at Path. Thus Path takes precedence over Data.
type UserData struct {
	Path string `yaml:"path,omitempty"`
	Data string `yaml:"data,omitempty"`
}
