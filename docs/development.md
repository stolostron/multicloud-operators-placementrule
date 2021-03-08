# Development guide

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Development guide](#development-guide)
    - [Launch Dev mode](#launch-dev-mode)
    - [Build a local image](#build-a-local-image)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Launch Dev mode

Run the following command to launch developer mode:

```shell
git clone git@github.com:open-cluster-management/multicloud-operators-placementrule.git
cd multicloud-operators-placementrule
export GITHUB_USER=<github_user>
export GITHUB_TOKEN=<github_token>
make
make build
./build/_output/bin/multicluster-operators-placementrule
```

## Build a local image

Run the following command to build an image:

```shell
git clone git@github.com:open-cluster-management/multicloud-operators-placementrule.git
cd multicloud-operators-placementrule
export GITHUB_USER=<github_user>
export GITHUB_TOKEN=<github_token>
make
make build-images
```
