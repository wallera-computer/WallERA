#!/bin/bash

if [[ $UID != 0 ]]; then
    echo this script needs root privileges
    exit 1
fi

set -e

MODULES=(libcomposite configfs dummy_hcd)

for i in ${MODULES[@]}; do
    echo loading $i
    modprobe $i
done
