package usb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"go.uber.org/zap"
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

type Frame interface {
	ChannelID() uint16
	Tag() uint8
	PacketIndex() uint16
	DataLength() uint16
	Data() []byte

	Type() int
	validate() error
}

const (
	typeFrame     = 1
	typeFrameNext = 2
)

// HIDFrame represents a HID frame compatible with LedgerJS implementation.
type HIDFrame struct {
	ChannelIDInner   uint16   // 2 bytes
	TagInner         uint8    // 1 byte
	PacketIndexInner uint16   // 2 bytes
	DataLengthInner  uint16   // 2 bytes
	DataInner        [57]byte // 57 bytes
}

func (hf HIDFrame) Type() int {
	return typeFrame
}

func (hf HIDFrame) ChannelID() uint16 {
	return hf.ChannelIDInner
}

func (hf HIDFrame) Tag() uint8 {
	return hf.TagInner
}

func (hf HIDFrame) PacketIndex() uint16 {
	return hf.PacketIndexInner
}

func (hf HIDFrame) DataLength() uint16 {
	return hf.DataLengthInner
}

func (hf HIDFrame) Data() []byte {
	return hf.DataInner[:]
}

// HIDFrameNext represents a HID frame that comes with sequence number > 0
type HIDFrameNext struct {
	ChannelIDInner   uint16   // 2 bytes
	TagInner         uint8    // 1 byte
	PacketIndexInner uint16   // 2 bytes
	DataInner        [59]byte // 57 bytes
}

func (hf HIDFrameNext) Type() int {
	return typeFrameNext
}

func (hf HIDFrameNext) ChannelID() uint16 {
	return hf.ChannelIDInner
}

func (hf HIDFrameNext) Tag() uint8 {
	return hf.TagInner
}

func (hf HIDFrameNext) PacketIndex() uint16 {
	return hf.PacketIndexInner
}

func (hf HIDFrameNext) DataLength() uint16 {
	return 0
}

func (hf HIDFrameNext) Data() []byte {
	return hf.DataInner[:]
}

// validate performs some basic validation on h.
func (h HIDFrame) validate() error {
	if h.TagInner != hidFrameTag {
		return fmt.Errorf("invalid frame tag")
	}

	if h.ChannelIDInner == 0 {
		return fmt.Errorf("channel id cannot be zero")
	}

	return nil
}

// validate performs some basic validation on h.
func (h HIDFrameNext) validate() error {
	if h.TagInner != hidFrameTag {
		return fmt.Errorf("invalid frame tag")
	}

	if h.ChannelIDInner == 0 {
		return fmt.Errorf("channel id cannot be zero")
	}

	return nil
}

// readFrame reads data into dest, returning an error if any.
func readFrame(data []byte, dest Frame) error {
	if len(data) > 64 || len(data) == 0 {
		return fmt.Errorf("data must be exactly 64 bytes long")
	}

	if err := binary.Read(bytes.NewReader(data), binary.BigEndian, dest); err != nil {
		return fmt.Errorf("invalid data, %w", err)
	}

	return nil
}

// Session represents a single data transmission session, identified by its channel ID.
type Session struct {
	channelID          uint16
	lastReadFrameIndex uint16
	data               *bytes.Buffer
	ShouldReadMore     bool
	amountToRead       uint16

	l *zap.SugaredLogger
}

// ReadData reads data into s.
// If this method is called on a fresh instance of Session, data will be handled as a HIDFrame,
// otherwise as a HIDFrameNext.
func (s *Session) ReadData(data []byte) error {
	var dest Frame

	dest = &HIDFrame{}
	if s.channelID != 0 {
		dest = &HIDFrameNext{}
	}

	err := readFrame(data, dest)
	if err != nil {
		return err
	}

	if err := dest.validate(); err != nil {
		return err
	}

	return s.ReadFrame(dest)
}

// ReadFrame does some basic checks on frame, and if positive will read frame data into s.
func (s *Session) ReadFrame(frame Frame) error {
	if s.channelID == 0 { // we're reading first frame
		return s.readFrame(frame)
	}

	if !s.ShouldReadMore {
		return fmt.Errorf("cannot read any more data in this session")
	}

	if s.channelID != frame.ChannelID() {
		return fmt.Errorf("different channel ID: expecting %v, received %v", s.channelID, frame.ChannelID())
	}

	if frame.PacketIndex()-s.lastReadFrameIndex != 1 {
		return fmt.Errorf("received out-of-order packet: expecting %v, received %v", s.lastReadFrameIndex+1, frame.PacketIndex())
	}

	s.readFrame(frame)

	return nil
}

func (s *Session) readFrame(frame Frame) error {
	s.channelID = frame.ChannelID()

	if s.channelID != 0 && frame.PacketIndex() == 0 {
		// we're trying to read a init packet in an already initialized session
		return fmt.Errorf("cannot read init packet in already initialized session")
	}

	s.lastReadFrameIndex = frame.PacketIndex()

	s.data.Write(frame.Data()[:])

	if frame.Type() == typeFrame {
		// we check against hidFrameMaxDataSize because if we're here
		// we're reading a HIDFrame, hence it might be all done in
		// a single frame.
		s.ShouldReadMore = !(frame.DataLength() <= hidFrameMaxDataSize)
		s.amountToRead = frame.DataLength()
		s.l.Debugw("read from initFrame", "amount", s.amountToRead)
	}

	if s.data.Len() >= int(s.amountToRead) {
		// since we read the entirety of the data field in s.data,
		// when this condition is true it means we finished writing, but
		// we must trim the data to s.amountToRead.
		s.ShouldReadMore = false

		lenBefTrunc := s.data.Len()
		s.data.Truncate(int(s.amountToRead))
		lenAftTrunc := s.data.Len()
		s.l.Debugw("data truncation", "before", lenBefTrunc, "after", lenAftTrunc)
	}

	return nil
}

const (
	defaultChunkSize = 57
	nextChunkSize    = 59
)

func chunkFunc(data []byte, size int) [][]byte {
	if len(data) <= size {
		return [][]byte{data}
	}

	chunks := int(math.Ceil(float64(len(data)) / float64(size)))

	ret := make([][]byte, 0, chunks)

	start := 0
	finish := size

	for idx := 0; idx < chunks; idx++ {
		if idx+1 == chunks {
			finish = len(data)
		}

		ret = append(ret, data[start:finish])
		start = start + size
		finish = finish + size

	}

	return ret
}

func chunk(data []byte) [][]byte {
	return chunkFunc(data, nextChunkSize)
}

func (s *Session) FormatResponse(data []byte) [][]byte {
	if len(data) <= defaultChunkSize {
		hf := HIDFrame{
			ChannelIDInner:   s.channelID,
			TagInner:         hidFrameTag,
			PacketIndexInner: uint16(0),
			// DataLength is only present in the first HID response frame, and is composed by the
			// first frame length along with the remaining data length.
			DataLengthInner: uint16(len(data)),
			DataInner:       [57]byte{},
		}

		copy(hf.DataInner[:], data)

		resp := &bytes.Buffer{}
		binary.Write(resp, binary.BigEndian, hf)

		return [][]byte{
			resp.Bytes(),
		}
	}

	firstChunk := data[0:defaultChunkSize]
	data = data[defaultChunkSize:]

	chunks := chunk(data)

	ret := make([][]byte, 0, len(chunks))

	// serialize first 57 bytes
	hf := HIDFrame{
		ChannelIDInner:   s.channelID,
		TagInner:         hidFrameTag,
		PacketIndexInner: uint16(0),
		// DataLength is only present in the first HID response frame, and is composed by the
		// first frame length along with the remaining data length.
		DataLengthInner: uint16(len(data) + len(firstChunk)),
		DataInner:       [57]byte{},
	}

	copy(hf.DataInner[:], firstChunk)

	resp := &bytes.Buffer{}
	binary.Write(resp, binary.BigEndian, hf)
	ret = append(ret, resp.Bytes())

	for i, chunk := range chunks {
		hf := HIDFrameNext{
			ChannelIDInner:   s.channelID,
			TagInner:         hidFrameTag,
			PacketIndexInner: uint16(i + 1),
			DataInner:        [59]byte{},
		}

		copy(hf.DataInner[:], chunk)

		resp := &bytes.Buffer{}
		binary.Write(resp, binary.BigEndian, hf)
		ret = append(ret, resp.Bytes())
	}

	return ret
}

func (s *Session) Data() []byte {
	return s.data.Bytes()
}

// NewSession returns a Session initialized with whatever data frame contains.
func NewSession(data []byte, l *zap.SugaredLogger) (Session, error) {
	s := Session{
		data:           &bytes.Buffer{},
		ShouldReadMore: true,

		l: l,
	}

	err := s.ReadData(data)
	if err != nil {
		return Session{}, err
	}

	return s, nil
}
