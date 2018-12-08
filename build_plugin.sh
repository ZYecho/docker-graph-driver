#!/bin/bash

SUDO=sudo
# Setup archive directories
BASE_DIR=../
PLUGIN_BASE_DIR=$BASE_DIR/plugin/artifacts

DOCKER_HUB_REPO=ZYecho
DOCKER_HUB_CEPH_PLUGIN=ceph-storage-driver
DOCKER_HUB_CEPH_TAG=latest

mkdir -p $PLUGIN_BASE_DIR
cp config.json $PLUGIN_BASE_DIR/
mkdir -p $PLUGIN_BASE_DIR/rootfs

# Build the base rootfs image
$SUDO docker build -t rootfsimage .
# Create a container from the rootfs image
id=$($SUDO docker create rootfsimage)
# Export and untar the container into a rootfs directory
$SUDO docker export "$id" | tar -x -C $PLUGIN_BASE_DIR/rootfs
# Create a docker v2 plugin
$SUDO docker plugin create $DOCKER_HUB_REPO/$DOCKER_HUB_CEPH_PLUGIN:$DOCKER_HUB_LCFS_TAG $PLUGIN_BASE_DIR
# Remove the temporary container
$SUDO docker rm -vf "$id"
$SUDO docker rmi rootfsimage

# Remove the archive directory
$SUDO rm -rf $PLUGIN_BASE_DIR