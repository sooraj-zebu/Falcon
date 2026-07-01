package storage

type Health struct {
	MountPath string

	Mounted   bool
	Writable  bool
	Readable  bool

	Filesystem string

	Total uint64
	Used  uint64
	Free  uint64

	Healthy bool
	Message string
}
