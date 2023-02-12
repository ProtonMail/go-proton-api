package proton

// Volume is a Proton Drive volume.
type Volume struct {
	VolumeID string // Encrypted volume ID

	CreateTime int64 // Creation time of the volume in Unix time
	ModifyTime int64 // Last modification time of the volume in Unix time
	MaxSpace   int64 // Space limit for the volume in bytes
	UsedSpace  int64 // Space used by files in the volume in bytes

	State VolumeState // The state of the volume (active, locked, maybe more in the future)
	Share VolumeShare // The main share of the volume
}

// VolumeShare is the main share of a volume.
type VolumeShare struct {
	ShareID string // Encrypted share ID
	LinkID  string // Encrypted link ID
}

// VolumeState is the state of a volume.
type VolumeState int

const (
	ActiveVolumeState VolumeState = 1
	LockedVolumeState VolumeState = 3
)
