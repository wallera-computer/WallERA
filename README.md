# WallERA

## `wallera-linux`

You will need a Linux kernel with `libcomposite`, `dummy_hcd` and `configfs`.

Either compile your own kernel or use `linux-zen`.

0. `modprobe libcomposite`, `modprobe dummy_hcd`, `modprobe configfs`
1. `make wallera-linux`
2. `sudo ./wallera-linux`

# HID transport packet format

Each packet is 64 bytes in size, as specified by the HID report descriptor.

Example packet:

```
[38 190 5 0 0 0 5 85 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
 |___|  | |_| |_|  |________________________________________________________________________________________________________________|
   |    |  |   |												|
   |	|  |   |-> Upper-level data packet length								|-> Upper-level data packet
   |	|  |-> Packet index
   |	|-> Tag
   |-> Channel ID
```

The maximum amount of data each upper-level data packet can hold is 64-5 = 59 bytes.

An hypothetical Go struct representing a HID frame is the following:

```go
type HIDFrame struct {
	ChannelID   uint16
	Tag         uint8
	PacketIndex uint16
	DataLength  uint16
	Data        [57]byte
}
```

The upper-level data packet framing is implementation-dependent, typically APDU is used.
