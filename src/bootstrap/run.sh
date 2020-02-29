#!/bin/bash

# TODO: complete this script

# re-build the protos
cd ../common/protos/; bash make_protos.sh; protos_build_status=$?; cd -
if [ $protos_build_status -eq 1 ]; then
    echo "Error building protos"
    exit $protos_build_status
fi
echo "Successfully build protos..."

