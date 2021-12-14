package apps

//go:generate stringer -type APDUCode
type APDUCode uint16

const (
	APDUExecutionError       APDUCode = 0x6400 // Execution Error
	APDUEmptyBuffer          APDUCode = 0x6982 // Empty buffer
	APDUOutputBufferTooSmall APDUCode = 0x6983 // Output buffer too small
	APDUCommandNotAllowed    APDUCode = 0x6986 // Command not allowed
	APDUINSNotSupported      APDUCode = 0x6D00 // INS not supported
	APDUCLANotSupported      APDUCode = 0x6E00 // CLA not supported
	APDUUnknown              APDUCode = 0x6F00 // Unknown
	APDUSuccess              APDUCode = 0x9000 // Success
	APDUWrongLength          APDUCode = 0x6700 // Wrong length
	APDUDataInvalid          APDUCode = 0x6984 // Data invalid
)
