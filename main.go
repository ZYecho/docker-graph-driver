package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/ZYecho/docker-graph-driver/ceph"
	"github.com/docker/go-plugins-helpers/graphdriver/shim"
)

const (
	socketAddress = "/run/docker/plugins/ceph.sock"
)

func main() {
	h := shim.NewHandlerFromGraphDriver(ceph.Init)
	//logrus.SetLevel(logrus.DebugLevel)
	logrus.Infof("listening on %s\n", socketAddress)
	fmt.Println(h.ServeUnix(socketAddress, 0))
}
