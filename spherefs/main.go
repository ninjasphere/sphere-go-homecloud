package spherefs

import (
	"encoding/json"
	"os"

	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/sphere-go-homecloud/models"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
)

var enableSphereFS = config.Bool(false, "homecloud.sphereFS.enabled")

var log = logger.GetLogger("SphereFS")

type SphereFS struct {
	ThingModel *models.ThingModel `inject:""`
}

func NewSphereFS() *SphereFS {
	return &SphereFS{}
}

func (s *SphereFS) PostConstruct() error {
	if !enableSphereFS {
		return nil
	}

	var mountPoint = config.String("/sphere", "homecloud.sphereFS.mountPoint")

	c, err := fuse.Mount(
		mountPoint,
		fuse.FSName("SphereFS"),
		fuse.Subtype("sphere"),
		fuse.LocalVolume(),
		fuse.VolumeName("SphereFS"),
	)
	if err != nil {
		return err
	}

	things, err := s.ThingModel.FetchAll()

	if err != nil {
		return err
	}
	t := *things

	go func() {

		defer c.Close()

		err = fs.Serve(c, &RootFS{t})
		if err != nil {
			log.Fatalf("Failed to serve SphereFS: %s", err)
		}

		// check if the mount process has an error to report
		<-c.Ready
		if err := c.MountError; err != nil {
			log.Fatalf("Failed to mount SphereFS at %s: %s", mountPoint, err)
		}
	}()
	return nil
}

type RootFS struct {
	things []*model.Thing
}

func (r *RootFS) Root() (fs.Node, fuse.Error) {
	return &RootDir{r.things}, nil
}

type RootDir struct {
	things []*model.Thing
}

func (d *RootDir) Attr() fuse.Attr {
	return fuse.Attr{Inode: 1, Mode: os.ModeDir | 0555}
}

func (d *RootDir) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	for i, thing := range d.things {
		if thing.ID == name {
			return &ThingFile{uint64(i), thing}, nil
		}
	}
	return nil, fuse.ENOENT
}

func (d *RootDir) ReadDir(intr fs.Intr) ([]fuse.Dirent, fuse.Error) {

	thingDirs := make([]fuse.Dirent, len(d.things))

	for i, thing := range d.things {
		thingDirs[i] = fuse.Dirent{Inode: uint64(i), Name: thing.ID, Type: fuse.DT_Dir}
	}

	return thingDirs, nil
}

type ThingFile struct {
	inode uint64
	thing *model.Thing
}

func (f *ThingFile) Attr() fuse.Attr {
	s, _ := json.Marshal(f.thing)
	return fuse.Attr{Inode: f.inode, Mode: 0444, Size: uint64(len(s))}
}

func (f *ThingFile) ReadAll(intr fs.Intr) ([]byte, fuse.Error) {
	s, _ := json.Marshal(f.thing)
	return []byte(s), nil
}
