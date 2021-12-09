# Wallera vagrant dev box

Build it with Packer (tested with packer 1.7.8)

```console
packer build wallera-dev.json
vagrant box add --name wallera packer_vmware-iso_vmware.box
```
