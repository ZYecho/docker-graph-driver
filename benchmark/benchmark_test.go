package benchmark

import (
	"fmt"
	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
	"testing"
	"time"
)

// This file used to test rbd snapshot vs rbd cp

const (
	DefaultRadosConfigFile = "/etc/ceph/ceph.conf"
	DefaultPoolName        = "rbd"
)

var (
	conn  *rados.Conn
	ioctx *rados.IOContext
	err   error
)

func init() {
	fmt.Println("do in init")
	conn, _ = rados.NewConn()
	if err = conn.ReadConfigFile(DefaultRadosConfigFile); err != nil {
		fmt.Printf("Rdb read config file failed: %v", err)
		return
	}

	if err := conn.Connect(); err != nil {
		fmt.Printf("Rbd connect failed: %v", err)
		return
	}

	ioctx, err = conn.OpenIOContext(DefaultPoolName)
	if err != nil {
		fmt.Printf("Rbd open pool %s failed: %v", DefaultPoolName, err)
		conn.Shutdown()
		return
	}
}

func BenchmarkSnapshotAndClone(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := createImage("docker_image_59d5ac28d116df91d70cac25a428a1f55373bf57933c5045a8a369ab66f7f83c", fmt.Sprintf("test_snapshot_clone_%d", genRandomHash())); err != nil {
			b.Fatal("failed to create image using snapshot and clone")
		}
	}
}

func createImage(baseImgName, imgName string) error {
	var snapshot *rbd.Snapshot

	img := rbd.GetImage(ioctx, baseImgName)

	// create snapshot for hash
	snapName := "snapshot_" + imgName

	if err := img.Open(snapName); err != nil {
		if err != rbd.RbdErrorNotFound {
			//fmt.Printf("Rbd open image %s with snap %s failed: %v", baseImgName, snapName, err)
			return err
		}

		// open image without snapshot name
		if err = img.Open(); err != nil {
			//fmt.Printf("Rbd open image %s failed: %v", baseImgName, err)
			return err
		}

		// create snapshot
		if snapshot, err = img.CreateSnapshot(snapName); err != nil {
			//fmt.Printf("Rbd create snapshot %s failed: %v", snapName, err)
			img.Close()
			return err
		}

	} else {
		snapshot = img.GetSnapshot(snapName)
	}

	// open snapshot success
	defer img.Close()

	// protect snapshot
	if err := snapshot.Protect(); err != nil {
		//fmt.Printf("Rbd protect snapshot %s failed: %v", snapName, err)
		//return err
	}

	// clone image
	_, err := img.Clone(snapName, ioctx, imgName, rbd.RbdFeatureLayering, 0)
	if err != nil {
		fmt.Printf("Rbd clone snapshot %s@%s to %s failed: %v", baseImgName, snapName, imgName, err)
		return err
	}

	return nil
}

func BenchmarkCopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := createImageByCopy("docker_image_59d5ac28d116df91d70cac25a428a1f55373bf57933c5045a8a369ab66f7f83c", fmt.Sprintf("test_copy_%d", genRandomHash())); err != nil {
			b.Fatal("failed to create image using copy")
		}
	}
}

func createImageByCopy(baseImgName, imgName string) error {
	img := rbd.GetImage(ioctx, baseImgName)
	if err := img.Open(true); err != nil {
		fmt.Printf("Rbd failed to open image")
		return err
	}
	if err := img.Copy(*ioctx, imgName); err != nil {
		fmt.Printf("Rbd copy image failed: %v", err)
		//return err
	}
	return nil
}

func genRandomHash() int64 {
	return time.Now().Unix()
}
