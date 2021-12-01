# WallERA

## `wallera-linux`

You will need a Linux kernel with `libcomposite`, `dummy_hcd` and `configfs`.

Either compile your own kernel or use `linux-zen`.

0. `modprobe libcomposite`, `modprobe dummy_hcd`, `modprobe configfs`
1. `make wallera-linux`
2. `sudo ./wallera-linux`
