// +build linux

package ceph

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/graphdriver"
	"github.com/docker/docker/pkg/containerfs"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/mount"
)

type CephDriver struct {
	home string
	*RbdSet
}

func Init(home string, options []string, uidMaps, gidMaps []idtools.IDMap) (graphdriver.Driver, error) {
	log.Debug("Init called, home = %s", home)
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
	}
	return graphdriver.NewNaiveDiffDriver(d, uidMaps, gidMaps), nil
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

	rootFs := path.Join(mp, "rootfs")
	if err := os.MkdirAll(rootFs, 0755); err != nil && !os.IsExist(err) {
		d.RbdSet.UnmountDevice(id)
		return nil, err
	}

	idFile := path.Join(mp, "id")
	if _, err := os.Stat(idFile); err != nil && os.IsNotExist(err) {
		// Create an "id" file with the container/image id in it to help reconstruct this in case
		// of later problems
		if err := ioutil.WriteFile(idFile, []byte(id), 0600); err != nil {
			d.RbdSet.UnmountDevice(id)
			return nil, err
		}
	}

	return containerfs.NewLocalContainerFS(rootFs), nil
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
