#!/bin/bash

# Allows VMware customization of the VM using cloud-init, see https://kb.vmware.com/s/article/54986
# We need to clear out everything produced by the previous cloud-init run from packer that created the template
cloud-init clean --logs
rm -rf /var/lib/cloud/*
rm -f /etc/cloud/cloud.cfg
rm -f /etc/netplan/*
truncate -s 0 /etc/machine-id
echo "disable_vmware_customization: false" | tee -a /etc/cloud/cloud.cfg

echo "==> Cleaning up temporary files"
rm -rf /tmp/*
rm -rf /usr/src/linux-source-*

# Cleanup apt cache
apt-get -y autoremove --purge
apt-get -y clean
apt-get -y autoclean

echo "==> Installed packages"
dpkg --get-selections | grep -v deinstall

# Remove Bash history
unset HISTFILE
rm -f /root/.bash_history
rm -f /home/vagrant/.bash_history

# Clean up log files
find /var/log -type f | while read f; do echo -ne '' > "$f"; done;

export PREFIX="/sbin"

FileSystem=$(grep ext /etc/mtab| awk -F" " '{ print $2 }')

for i in $FileSystem
do
        echo "$i"
        number=$(df -B 512 "$i" | awk -F" " '{print $3}' | grep -v Used)
        echo "$number"
        percent=$(echo "scale=0; $number * 98 / 100" | bc )
        echo "$percent"
        dd count=$(echo $percent) if=/dev/zero of=$(echo $i)/zf
        /bin/sync
        sleep 15
        rm -f "$i"/zf
done

VolumeGroup=$($PREFIX/vgdisplay | grep Name | awk -F" " '{ print $3 }')

for j in $VolumeGroup
do
        echo "$j"
        $PREFIX/lvcreate -l $($PREFIX/vgdisplay $j | grep Free | awk -F" " '{ print $5 }') -n zero "$j"
        if [ -e /dev/"$j"/zero ]; then
                cat /dev/zero > /dev/"$j"/zero
                /bin/sync
                sleep 15
                $PREFIX/lvremove -f /dev/"$j"/zero
        fi
done

# Make sure we wait until all the data is written to disk, otherwise
# Packer might quit too early before the large files are deleted
sync