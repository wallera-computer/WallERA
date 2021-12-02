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
   |    |  |   |	|-> Upper-level data packet
   |	|  |   |-> Upper-level data packet length
   |	|  |-> Packet index
   |	|-> Tag
   |-> Channel ID
```

The maximum amount of data each upper-level data packet can hold is 64-7 = 57 bytes.

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

**PSA:** hypothesis ahead!

Each packet can hold maximum of 57 bytes, times `max(uint16)` = 57 * 65535 = 3735495 bytes is the maximum amount of bytes we can send over this framing.

AFAIS Ledger add a channel identification number _but_ doesn't really like being connected to multiple parties at the same time, AKA only one program running on the host can access the Ledger.

This greatly simplifies how we reason about the system

If we assume 57 bytes per packet received, `DataLength` should decrease for each packet we receive. 

When `DataLength =< 57` it means the packet we just received is the last one and we can now pass the resulting data slice to the upper-level layer.
