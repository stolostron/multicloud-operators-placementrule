#!/bin/bash
echo "BUILD GOES HERE!"

# Run our build target and set IMAGE_NAME_AND_VERSION
export IMAGE_NAME_AND_VERSION=${1}
make build
make images
