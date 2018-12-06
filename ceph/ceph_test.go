package ceph

import (
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/docker/docker/daemon/graphdriver"
)

var (
	tmpOuter = path.Join(os.TempDir(), "ceph-tests")
	tmp      = path.Join(tmpOuter, "ceph")
)

func testInit(dir string, t testing.TB) graphdriver.Driver {
	d, err := Init(dir, nil, nil, nil)
	if err != nil {
		if err == graphdriver.ErrNotSupported {
			t.Skip(err)
		} else {
			t.Fatal(err)
		}
	}
	return d
}

func newDriver(t testing.TB) graphdriver.Driver {
	if err := os.MkdirAll(tmp, 0755); err != nil {
		t.Fatal(err)
	}

	d := testInit(tmp, t)
	return d
}

func cleanup(t *testing.T, d graphdriver.Driver) {
	if err := d.Cleanup(); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(tmp)
}

func PutDriver(t *testing.T, d graphdriver.Driver) {
	if d == nil {
		t.Skip("No driver to put!")
	}
	cleanup(t, d)
}

func verifyFile(t *testing.T, path string, mode os.FileMode, uid, gid uint32) {
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if fi.Mode()&os.ModeType != mode&os.ModeType {
		t.Fatalf("Expected %s type 0x%x, got 0x%x", path, mode&os.ModeType, fi.Mode()&os.ModeType)
	}

	if fi.Mode()&os.ModePerm != mode&os.ModePerm {
		t.Fatalf("Expected %s mode %o, got %o", path, mode&os.ModePerm, fi.Mode()&os.ModePerm)
	}

	if fi.Mode()&os.ModeSticky != mode&os.ModeSticky {
		t.Fatalf("Expected %s sticky 0x%x, got 0x%x", path, mode&os.ModeSticky, fi.Mode()&os.ModeSticky)
	}

	if fi.Mode()&os.ModeSetuid != mode&os.ModeSetuid {
		t.Fatalf("Expected %s setuid 0x%x, got 0x%x", path, mode&os.ModeSetuid, fi.Mode()&os.ModeSetuid)
	}

	if fi.Mode()&os.ModeSetgid != mode&os.ModeSetgid {
		t.Fatalf("Expected %s setgid 0x%x, got 0x%x", path, mode&os.ModeSetgid, fi.Mode()&os.ModeSetgid)
	}

	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		if stat.Uid != uid {
			t.Fatalf("%s no owned by uid %d", path, uid)
		}
		if stat.Gid != gid {
			t.Fatalf("%s not owned by gid %d", path, gid)
		}
	}

}

func createBase(t *testing.T, driver graphdriver.Driver, name string) {
	// We need to be able to set any perms
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	if err := driver.Create(name, "", nil); err != nil {
		t.Fatal(err)
	}

	dir, err := driver.Get(name, "")
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Put(name)

	subdir := path.Join(dir.Path(), "a subdir")
	if err := os.Mkdir(subdir, 0705|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(subdir, 1, 2); err != nil {
		t.Fatal(err)
	}

	file := path.Join(dir.Path(), "a file")
	if err := ioutil.WriteFile(file, []byte("Some data"), 0222|os.ModeSetuid); err != nil {
		t.Fatal(err)
	}
}

func verifyBase(t *testing.T, driver graphdriver.Driver, name string) {
	dir, err := driver.Get(name, "")
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Put(name)

	subdir := path.Join(dir.Path(), "a subdir")
	verifyFile(t, subdir, 0705|os.ModeDir|os.ModeSticky, 1, 2)

	file := path.Join(dir.Path(), "a file")
	verifyFile(t, file, 0222|os.ModeSetuid, 0, 0)

	fis, err := ioutil.ReadDir(dir.Path())
	if err != nil {
		t.Fatal(err)
	}

	if len(fis) != 2 {
		t.Fatal("Unexpected files in base image")
	}

}

// Creates an new image and verifies it is empty and the right metadata
func TestCreateEmpty(t *testing.T) {
	driver := newDriver(t)
	defer PutDriver(t, driver)

	if err := driver.Create("empty", "", nil); err != nil {
		t.Fatal(err)
	}

	if !driver.Exists("empty") {
		t.Fatal("Newly created image doesn't exist")
	}

	dir, err := driver.Get("empty", "")
	if err != nil {
		t.Fatal(err)
	}

	verifyFile(t, dir.Path(), 0755|os.ModeDir, 0, 0)

	// Verify that the directory is empty
	fis, err := ioutil.ReadDir(dir.Path())
	if err != nil {
		t.Fatal(err)
	}

	if len(fis) != 0 {
		t.Fatal("New directory not empty")
	}

	driver.Put("empty")

	if err := driver.Remove("empty"); err != nil {
		t.Fatal(err)
	}

}

func TestCreateBase(t *testing.T) {
	driver := newDriver(t)
	defer PutDriver(t, driver)

	createBase(t, driver, "Base")
	verifyBase(t, driver, "Base")

	if err := driver.Remove("Base"); err != nil {
		t.Fatal(err)
	}
}

func TestCreateSnap(t *testing.T) {
	driver := newDriver(t)
	defer PutDriver(t, driver)

	createBase(t, driver, "Base")

	if err := driver.Create("Snap", "Base", nil); err != nil {
		t.Fatal(err)
	}

	verifyBase(t, driver, "Snap")

	if err := driver.Remove("Snap"); err != nil {
		t.Fatal(err)
	}

	if err := driver.Remove("Base"); err != nil {
		t.Fatal(err)
	}
}
