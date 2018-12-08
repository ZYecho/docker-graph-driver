// +build linux

package ceph

import (
	"io"
	"os"
	"path"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/graphdriver"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/chrootarchive"
	"github.com/docker/docker/pkg/containerfs"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/mount"
)

type CephDriver struct {
	home string
	*RbdSet
	uidMaps []idtools.IDMap
	gidMaps []idtools.IDMap
}

func Init(home string, options []string, uidMaps, gidMaps []idtools.IDMap) (graphdriver.Driver, error) {
	log.Debugf("Init called, home = %s", home)
	if err := os.MkdirAll(home, 0700); err != nil && !os.IsExist(err) {
		log.Errorf("Rbd create home dir %s failed: %v", err)
		return nil, err
	}

	rbdSet, err := NewRbdSet(home, true, options)
	if err != nil {
		return nil, err
	}

	if err := mount.MakePrivate(home); err != nil {
		return nil, err
	}

	d := &CephDriver{
		RbdSet: rbdSet,
		home:   home,
		uidMaps: uidMaps,
		gidMaps: gidMaps,
	}
	return d, nil
}

func (d *CephDriver) String() string {
	log.Debug("String called")
	return "ceph"
}

func (d *CephDriver) Status() [][2]string {
	log.Debug("Status called")
	status := [][2]string{
		{"Pool Objects", ""},
	}
	return status
}

func (d *CephDriver) GetMetadata(id string) (map[string]string, error) {
	log.Debugf("GetMata called, id = %s", id)
	info := d.RbdSet.Devices[id]

	metadata := make(map[string]string)
	metadata["BaseHash"] = info.BaseHash
	metadata["DeviceSize"] = strconv.FormatUint(info.Size, 10)
	metadata["DeviceName"] = info.Device
	return metadata, nil
}

func (d *CephDriver) Cleanup() error {
	log.Debug("cleanup called")
	err := d.RbdSet.Shutdown()

	if err2 := mount.Unmount(d.home); err2 == nil {
		err = err2
	}

	return err
}

func (d *CephDriver) CreateReadWrite(id, parent string, opts *graphdriver.CreateOpts) error {
	log.Debugf("CreateReadWrite called, id = %s", id)
	return d.Create(id, parent, opts)
}

func (d *CephDriver) Create(id, parent string, opts *graphdriver.CreateOpts) error {
	log.Debugf("Create called, id = %s", id)
	if err := d.RbdSet.AddDevice(id, parent); err != nil {
		return err
	}
	return nil
}

func (d *CephDriver) Remove(id string) error {
	log.Debugf("Remove called, id = %s", id)
	if !d.RbdSet.HasDevice(id) {
		return nil
	}

	if err := d.RbdSet.DeleteDevice(id); err != nil {
		return err
	}

	mountPoint := path.Join(d.home, "mnt", id)
	if err := os.RemoveAll(mountPoint); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (d *CephDriver) Get(id, mountLabel string) (containerfs.ContainerFS, error) {
	log.Debugf("Get called, id = %s", id)
	mp := path.Join(d.home, "mnt", id)

	if err := os.MkdirAll(mp, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}

	if err := d.RbdSet.MountDevice(id, mp, mountLabel); err != nil {
		return nil, err
	}
	return containerfs.NewLocalContainerFS(mp), nil
}

func (d *CephDriver) Put(id string) error {
	log.Debugf("Put called, id = %d", id)
	if err := d.RbdSet.UnmountDevice(id); err != nil {
		log.Errorf("Warning: error unmounting device %s: %s", id, err)
		return err
	}
	return nil
}

func (d *CephDriver) Exists(id string) bool {
	log.Debugf("Exists called, id = %s", id)
	return d.RbdSet.HasDevice(id)
}

// ApplyDiff extracts the changeset from the given diff into the
// layer with the specified id and parent, returning the size of the
// new layer in bytes.
func (d *CephDriver) ApplyDiff(id, parent string, diff io.Reader) (size int64, err error) {
	log.Debugf("ApplyDiff called, id = %s", id)
	// Mount the root filesystem so we can apply the diff/layer.
	layerRootFs, err := d.Get(id, "")
	if err != nil {
		return
	}
	defer d.Put(id)

	layerFs := layerRootFs.Path()
	options := &archive.TarOptions{UIDMaps: d.uidMaps,
		GIDMaps: d.gidMaps}
	start := time.Now().UTC()
	log.WithField("id", id).Debug("Start untar layer")
	if size, err = chrootarchive.ApplyUncompressedLayer(layerFs, diff, options); err != nil {
		return
	}
	log.WithField("id", id).Debugf("Untar time: %vs", time.Now().UTC().Sub(start).Seconds())

	return
}

// Changes produces a list of changes between the specified layer
// and its parent layer. If parent is "", then all changes will be ADD changes.
func (d *CephDriver) Changes(id, parent string) ([]archive.Change, error) {
	log.Debugf("Changes called, id = %s", id)
	layerRootFs, err := d.Get(id, "")
	if err != nil {
		return nil, err
	}
	defer d.Put(id)

	layerFs := layerRootFs.Path()
	parentFs := ""

	if parent != "" {
		parentRootFs, err := d.Get(parent, "")
		if err != nil {
			return nil, err
		}
		defer d.Put(parent)
		parentFs = parentRootFs.Path()
	}

	return archive.ChangesDirs(layerFs, parentFs)
}

// DiffSize calculates the changes between the specified layer
// and its parent and returns the size in bytes of the changes
// relative to its base filesystem directory.
func (d *CephDriver) DiffSize(id, parent string) (size int64, err error) {
	log.Debugf("DiffSize called, id = %s", id)
	changes, err := d.Changes(id, parent)
	if err != nil {
		return
	}

	layerFs, err := d.Get(id, "")
	if err != nil {
		return
	}
	defer d.Put(id)

	return archive.ChangesSize(layerFs.Path(), changes), nil
}

// Diff produces an archive of the changes between the specified
// layer and its parent layer which may be "".
func (d *CephDriver) Diff(id, parent string) (arch io.ReadCloser, err error) {
	startTime := time.Now()
	log.Debugf("Diff called, id = %s", id)
	layerRootFs, err := d.Get(id, "")
	if err != nil {
		return nil, err
	}
	layerFs := layerRootFs.Path()

	defer func() {
		if err != nil {
			d.Put(id)
		}
	}()

	if parent == "" {
		archive, err := archive.Tar(layerFs, archive.Uncompressed)
		if err != nil {
			return nil, err
		}
		return ioutils.NewReadCloserWrapper(archive, func() error {
			err := archive.Close()
			d.Put(id)
			return err
		}), nil
	}

	parentRootFs, err := d.Get(parent, "")
	if err != nil {
		return nil, err
	}
	defer d.Put(parent)

	parentFs := parentRootFs.Path()

	changes, err := archive.ChangesDirs(layerFs, parentFs)
	if err != nil {
		return nil, err
	}

	archive, err := archive.ExportChanges(layerFs, changes, d.uidMaps, d.gidMaps)
	if err != nil {
		return nil, err
	}

	return ioutils.NewReadCloserWrapper(archive, func() error {
		err := archive.Close()
		d.Put(id)

		// NaiveDiffDriver compares file metadata with parent layers. Parent layers
		// are extracted from tar's with full second precision on modified time.
		// We need this hack here to make sure calls within same second receive
		// correct result.
		time.Sleep(time.Until(startTime.Truncate(time.Second).Add(time.Second)))
		return err
	}), nil
}
