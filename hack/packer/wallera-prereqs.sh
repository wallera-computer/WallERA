#!/bin/bash
echo "===> Installing Wallera Prereqs"

echo "===> Setting MOTD"
rm /etc/update-motd.d/*
cp /tmp/motd /etc/update-motd.d/motd
chmod +x /etc/update-motd.d/motd

echo "===> Installing prereqs"
apt update -y -q
apt install -y -q asciidoc-base autoconf bc bison build-essential cmake fakeroot flex g++ gcc git libconfig-dev libelf-dev libglib2.0-dev libncurses-dev libssl-dev libsystemd-dev libtool usbutils zstd linux-source

echo "===> Installing TAMAGO"
cd /tmp || exit
wget -nv https://github.com/f-secure-foundry/tamago-go/releases/download/tamago-go1.17.4/tamago-go1.17.4.linux-amd64.tar.gz
tar -xf tamago-go1.17.4.linux-amd64.tar.gz -C /

echo "===> Installing Libusbgx"
git clone https://github.com/libusbgx/libusbgx
cd libusbgx || exit
# git checkout -b libusbgx-v0.2.0
autoreconf -i
./configure --prefix=/usr
make -j16
make install
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
make -j16 bindeb-pkg
cd .. || exit
dpkg -i linux-*.deb
update-grub

echo "===> Set modules to load at boot time"
{
  echo "libcomposite"
  echo "dummy_hcd"
  echo "configfs"
} >> /etc/modules

echo "===> Configure shell env"
{
  echo 'export GOROOT=/usr/local/tamago-go'
  echo 'export PATH=$PATH:$GOROOT/bin'
  echo 'export TAMAGO=$GOROOT/bin/go'
} | tee -a /home/vagrant/.bashrc /root/.bashrc