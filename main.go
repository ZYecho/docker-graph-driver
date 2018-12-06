package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/ZYecho/docker-graph-driver/rbd"
	"github.com/docker/go-plugins-helpers/graphdriver/shim"
)

const (
	socketAddress = "/run/docker/plugins/rbd.sock"
)

func main() {
	h := shim.NewHandlerFromGraphDriver(rbd.Init)
	logrus.Infof("listening on %s\n", socketAddress)
	fmt.Println(h.ServeUnix(socketAddress, 0))
}
