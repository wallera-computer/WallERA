# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "wallera"
  config.vm.provider "vmware_desktop" do |vm|
    vm.vmx["memsize"] = "8192"
    vm.vmx["numvcpus"] = "4"
    vm.vmx["ethernet0.pcislotnumber"] = "160"
  end
  config.vm.provider "virtualbox" do |vb|
    vb.memory = 1024
    vb.cpus = 2
  end
end
