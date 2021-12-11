#!/bin/bash
set -e

echo "===> Installing Wallera Prereqs"

echo "===> Setting MOTD"
rm /etc/update-motd.d/*
cp /tmp/motd /etc/update-motd.d/motd
chmod +x /etc/update-motd.d/motd

echo "===> Installing prereqs"
apt update -y -q
apt install -y -q asciidoc-base autoconf bc bison build-essential cmake fakeroot flex g++ gcc git libconfig-dev libelf-dev libglib2.0-dev libncurses-dev libssl-dev libsystemd-dev libtool usbutils zstd linux-source
add-apt-repository ppa:longsleep/golang-backports -y 
apt update -y -q
apt install golang-go -y -q

echo "===> Installing TAMAGO"
cd /opt || exit
wget -nv https://github.com/f-secure-foundry/tamago-go/archive/refs/tags/tamago-go1.17.5.zip
unzip tamago-go1.17.5.zip
cd tamago-go-tamago-go1.17.5/src
./make.bash

echo "===> Installing Libusbgx"
cd /tmp
git clone https://github.com/libusbgx/libusbgx
cd libusbgx || exit
# git checkout -b libusbgx-v0.2.0
autoreconf -i
./configure --prefix=/usr
make -j16
make install
ldconfig
cd - || exit

echo "===> Installing GT"
git clone https://github.com/kopasiak/gt.git
cd gt/source || exit
cmake -DCMAKE_INSTALL_PREFIX= .
make
make install
cd - || exit

echo "===> Sourcing and Compiling Kernel"
cd /usr/src || exit
tar xfj linux-source-*.tar.bz2
cd linux-source-*/ || exit
cp /boot/config-$(uname -r)* .config
sed -i "s/^CONFIG_LOCALVERSION=.*/CONFIG_LOCALVERSION=\"-wallera\"/" .config
echo "CONFIG_USB_DUMMY_HCD=m" >> .config
yes '' | make oldconfig
make -j$(nproc) bindeb-pkg
cd .. || exit
dpkg -i linux-*.deb
update-grub

echo "===> Set modules to load at boot time"
{
  echo "libcomposite"
#  echo "dummy_hcd"
  echo "configfs"
} >> /etc/modules

echo "===> Configure shell env"
{
  echo 'export GOROOT=/opt/tamago-go-tamago-go1.17.5/'
  echo 'export PATH=$PATH:$GOROOT/bin'
  echo 'export TAMAGO=$GOROOT/bin/go'
} | tee -a /home/vagrant/.bashrc /root/.bashrc
