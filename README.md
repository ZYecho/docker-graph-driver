Docker external graph driver
============================

# Why external graph driver
Long times ago, @dachary proposed implement [Ceph storage driver](https://github.com/docker/docker/issues/8854), but still not achieved now. The main reason is that it is not possible to statically compile docker with the ceph graph driver enabled because some static libraries are missing (Ubuntu 14.04) at the moment. 

For support rbd storage driver, must use dynamic compile.This is not accepted by docker community.

- [#9146](https://github.com/docker/docker/pull/9146)
- [#14800](https://github.com/docker/docker/pull/14800/)

Now docker community plan to implement out-of-process graph driver [#13777](https://github.com/docker/docker/pull/13777). It is a good tradeoff between Docker and Ceph.

On the other hand, companies like EMC, NetApp and others will most likely not be sending pull requests to add their product specific graph driver to the Docker repository. That can be because of a multitude of reasons: they want to keep it closed source, they want to put some proprietary stuff in there, they'd want to be able to change it, update it separately from Docker releases and so on.

# How to use

## How to compile

```bash
go build -v
```

## Run docker graph driver

```bash
# ./docker-graph-driver
DEBU[0000] Rbd setup base image                         
INFO[0000] listening on /run/docker/plugins/ceph.sock
   
```

## Run docker daemon

```bash
# dockerd --experimental -s ceph --log-level debug 
```

## Pull images

```bash
# docker pull centos:latest
Pulling repository centos
7322fbe74aa5: Download complete 
f1b10cd84249: Download complete 
c852f6d61e65: Download complete 
Status: Downloaded newer image for centos:latest
```

## List rbd image

```bash
# rbd list
docker_image_7322fbe74aa5632b33a400959867c8ac4290e9c5112877a7754be70cfe5d66e9
docker_image_base_image
docker_image_c852f6d61e65cddf1e8af1f6cd7db78543bfb83cdcd36845541cf6d9dfef20a0
docker_image_f1b10cd842498c23d206ee0cbeaa9de8d2ae09ff3c7af2723a9e337a6965d639
```
## Run container

```bash
# docker run -it --rm centos:latest /bin/bash
[root@290238155b54 /]#
```

```bash
# rbd list
docker_image_290238155b547852916b732e38bc4494375e1ed2837272e2940dfccc62691f6c
docker_image_290238155b547852916b732e38bc4494375e1ed2837272e2940dfccc62691f6c-init
docker_image_7322fbe74aa5632b33a400959867c8ac4290e9c5112877a7754be70cfe5d66e9
docker_image_base_image
docker_image_c852f6d61e65cddf1e8af1f6cd7db78543bfb83cdcd36845541cf6d9dfef20a0
docker_image_f1b10cd842498c23d206ee0cbeaa9de8d2ae09ff3c7af2723a9e337a6965d639
```

# Integration with Docker Plugin System
1. make sure that you have install a ceph cluster
2. put the cluster certification file into dir ```ceph-key```
3. ```sudo sh build_plugin.sh``` to create docker plugin, after that you could use ```docker plugin``` to see new plugin.
4. use ```docker plugin enable pluginName``` to enable your plugin
5. stop and start dockerd use ```dockerd --experimental -s pluginName &```

## Others

- [Docker Graph Driver Plugin](https://docs.docker.com/engine/extend/plugins_graphdriver/#graph-driver-plugin-protocol)
- [Docker Plugin Config](https://docs.docker.com/engine/extend/config/#config-field-descriptions)
- [How to use Docker Plugin](https://docs.docker.com/engine/extend/#installing-and-using-a-plugin)
