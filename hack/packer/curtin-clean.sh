#!/bin/bash
# Removes files leftover from the cloud-config install
# If these are not removed, cloud-init cannot be run again (for example by Openstack or VMware customization)
FILES="/etc/cloud/cloud.cfg.d/50-curtin-networking.cfg
/etc/cloud/cloud.cfg.d/curtin-preserve-sources.cfg
/etc/cloud/cloud.cfg.d/subiquity-disable-cloudinit-networking.cfg"
for FILE in $FILES
do
  if test -f "$FILE"; then
    rm "$FILE"
  fi
done