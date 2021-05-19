package dolocal

type GitDrops struct {
	Privileges Privileges                  `yaml:"privileges"`
	Droplets   []LocalDropletCreateRequest `yaml:"droplets"`
	Volumes    []LocalVolumeCreateRequest  `yaml:"volumes"`
}

type Privileges struct {
	Create bool `yaml:"create"`
	Update bool `yaml:"update"`
	Delete bool `yaml:"delete"`
}

// LocalDropletCreateRequest is a simplified representation of godo.DropletCreateRequest.
// It is only a single level deep to enable unmarshalling from gitdrops.yaml.
type LocalDropletCreateRequest struct {
	Name              string   `yaml:"name"`
	Region            string   `yaml:"region"`
	Size              string   `yaml:"size"`
	Image             string   `yaml:"image"`
	SSHKeyFingerprint string   `yaml:"sshKeyFingerprint"`
	Backups           bool     `yaml:"backups"`
	IPv6              bool     `yaml:"ipv6"`
	Monitoring        bool     `yaml:"monitoring"`
	UserData          UserData `yaml:"userData,omitempty"`
	Volumes           []string `yaml:"volumes,omitempty"`
	Tags              []string `yaml:"tags"`
	VPCUUID           string   `yaml:"vpcuuid,omitempty"`
}

// TODO
type LocalVolumeCreateRequest struct {
	Name          string `yaml:"name"`
	Region        string `yaml:"region"`
	SizeGigaBytes int64  `yaml:"sizeGigaBytes"`
}

// UserData stores the Path of a userdata file and/or the Data itself. In the event that path is
// defined, Data is populated with contents of the file at Path. Thus Path 'data' takes precedence
// over Data.
type UserData struct {
	Path string `yaml:"path,omitempty"`
	Data string `yaml:"data,omitempty"`
}
