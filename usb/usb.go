package usb

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/hsanjuan/go-nfctype4/apdu"
)

const (
	hidFrameMaxDataSize = 57
	hidFrameTag         = 0x05
)

// Interface represents a high-level USB implementation which provides means of communication with
// a USB host.
type Interface interface {
	// Read reads an amount of bytes from the underlying USB handlers, and returns
	// them for processing.
	Read() ([]byte, error)

	// Write writes data to the underlying USB handler, returns an error if any.
	Write(data []byte) error
}

// HIDFrame represents a HID frame compatible with LedgerJS implementation.
type HIDFrame struct {
	ChannelID   uint16   // 2 bytes
	Tag         uint8    // 1 byte
	PacketIndex uint16   // 2 bytes
	DataLength  uint16   // 2 bytes
	Data        [57]byte // 57 bytes
}

// validate performs some basic validation on h.
func (h HIDFrame) validate() error {
	if h.Tag != hidFrameTag {
		return fmt.Errorf("invalid frame tag")
	}

	if h.ChannelID == 0 {
		return fmt.Errorf("channel id cannot be zero")
	}

	return nil
}

// ParseHIDFrame returns a HIDFrame from data.
// If the frame is not encoded as a LedgerJS HID frame, this function will
// return an error.
func ParseHIDFrame(data []byte) (HIDFrame, error) {
	ret := HIDFrame{}

	if len(data) > 64 || len(data) == 0 {
		return HIDFrame{}, fmt.Errorf("data must be exactly 64 bytes long")
	}

	if err := binary.Read(bytes.NewReader(data), binary.BigEndian, &ret); err != nil {
		return HIDFrame{}, fmt.Errorf("invalid data, %w", err)
	}

	if err := ret.validate(); err != nil {
		return HIDFrame{}, fmt.Errorf("frame validation failed, %w", err)
	}

	return ret, nil
}

// Session represents a single data transmission session, identified by its channel ID.
type Session struct {
	channelID          uint16
	lastReadFrameIndex uint16
	data               *bytes.Buffer
	shouldReadMore     bool
}

// CAPDU returns a command APDU packet from s data.
func (s Session) CAPDU() (apdu.CAPDU, error) {
	if s.shouldReadMore {
		return apdu.CAPDU{}, fmt.Errorf("cannot build apdu packet from incomplete data")
	}

	packet := apdu.CAPDU{}
	_, err := packet.Unmarshal(s.data.Bytes())
	if err != nil {
		return apdu.CAPDU{}, nil
	}

	return packet, nil
}

// ReadFrame does some basic checks on frame, and if positive will read frame data into s.
func (s *Session) ReadFrame(frame HIDFrame) error {
	if !s.shouldReadMore {
		return fmt.Errorf("cannot read any more data in this session")
	}

	if s.channelID != frame.ChannelID {
		return fmt.Errorf("different channel ID: expecting %v, received %v", s.channelID, frame.ChannelID)
	}

	if frame.PacketIndex-s.lastReadFrameIndex != 1 {
		return fmt.Errorf("received out-of-order packet: expecting %v, received %v", s.lastReadFrameIndex+1, frame.PacketIndex)
	}

	s.readFrame(frame)

	return nil
}

func (s *Session) readFrame(frame HIDFrame) {
	s.lastReadFrameIndex = frame.PacketIndex

	s.data.Write(frame.Data[:])

	s.shouldReadMore = !(frame.DataLength <= hidFrameMaxDataSize)
}

// NewSession returns a Session initialized with whatever data frame contains.
func NewSession(frame HIDFrame) (Session, error) {
	if frame.PacketIndex != 0 {
		return Session{}, fmt.Errorf("cannot create Session for non-first packet")
	}

	s := Session{
		channelID: frame.ChannelID,
		data:      &bytes.Buffer{},
	}

	s.readFrame(frame)

	return s, nil
}
