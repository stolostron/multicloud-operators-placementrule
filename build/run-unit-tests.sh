#!/bin/bash
echo "UNIT TESTS GO HERE!"

# Run unit test
export IMAGE_NAME_AND_VERSION=${1}
make test
