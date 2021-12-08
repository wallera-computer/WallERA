# WallERA

## `wallera-linux`

You will need a Linux kernel with `libcomposite`, `dummy_hcd` and `configfs`.

Either compile your own kernel or use `linux-zen`.

0. `modprobe libcomposite`, `modprobe dummy_hcd`, `modprobe configfs`
1. `make wallera-linux`
2. `sudo ./wallera-linux`

[Here](https://github.com/wallera-computer/ledgerjs-examples) are some LedgerJS-based examples one can use to fiddle with the implementation.

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

A follow-up packet is defined as follows:

```go
type HIDFrameNext struct {
	ChannelID   uint16
	Tag         uint8
	PacketIndex uint16
	Data        [59]byte
}
```

The upper-level data packet framing is implementation-dependent, typically APDU is used.

**PSA:** hypothesis ahead!

Each packet can hold maximum of 57 bytes, times `max(uint16)` = 57 * 65535 = 3735495 bytes is the maximum amount of bytes we can send over this framing.

AFAIS Ledger add a channel identification number _but_ doesn't really like being connected to multiple parties at the same time, AKA only one program running on the host can access the Ledger.

This greatly simplifies how we reason about the system

If we assume 57 bytes per packet received, `DataLength` should decrease for each packet we receive. 

When `DataLength =< 57` it means the packet we just received is the last one and we can now pass the resulting data slice to the upper-level layer.

## TODO
 - handle error codes correctly when commands do not return properly (check what the client code expects)

### Cosmos App

APDU packet schema is [here](https://github.com/LedgerHQ/app-cosmos/blob/master/docs/APDUSPEC.md)

The signature session is weird.

They can create three kinds of session:
 - `init`: contains derivation path used to sign the blob, only one packet of this kind will ever be observed
 - `add`: contains beginning of data blob to sign, multiple packets of this kind can be observed
 - `last`: contains remaining part of the data to be signed

Instead of using a single HID session and send all the data over, the Cosmos app needs this three-session approach because potentially a Cosmos signature client could beam over megabytes of data to be signed.

Signature payload is chunked in chunks of 255 bytes in size, so excluding `init` sessions which are always less than 57 bytes in length:
 - `add` sessions beam always 255 bytes of data
 - `last` sessions beam always less than 255 bytes of data

For each session, we have to pass data to the Cosmos app from the `usb` layer, let it process and respond back with OK/Error, otherwise the signature flow fails.

This means we have to build some sort of `SignatureSession` object which should persist across session handling calls: a state machine on top of another state machine.

An example signature request, with a payload of 355 bytes in length will work as follows:
 1. a `init` session is created, which consists of 1 `HIDFrame` frame; Cosmos app will elaborate this by initializing a `SignatureSession` internally, storing the derivation path in it
 2. a `add` session is created, which consists in 1 `HIDFrame` frame, 4 `HIDFrameNext` frames; Cosmos app will add the resulting data bytes in a `bytes.Buffer`
 3. a `last` session is created, which consists in 1 `HIDFrame` frame,  1 `HIDFrameNext` frame; Cosmos app will append the resulting data bytes in the `bytes.Buffer` created previously, sign the 355 bytes and send back a signature