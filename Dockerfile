# use this dockefile to build plugin container
FROM golang:1.11.2.ceph AS base

# copy ceph cluster certification file to container, default cluster name=ceph
COPY ceph-key/ceph.client.admin.keyring /etc/ceph/ceph.client.admin.keyring
COPY ceph-key/ceph.conf /etc/ceph/ceph.conf


RUN mkdir -p $GOPATH/src/golang.org/x \
    && cd $GOPATH/src/golang.org/x \
    && git clone https://github.com/golang/sys.git \
    && git clone https://github.com/golang/crypto.git \
    && git clone https://github.com/golang/net.git \
    && go get github.com/ZYecho/docker-graph-driver \
    && cd $GOPATH/src/github.com/ZYecho/docker-graph-driver \
    && go build -v \
    && cp docker-graph-driver /bin

WORKDIR $GOPATH/src/github.com/ZYecho/docker-graph-driver
