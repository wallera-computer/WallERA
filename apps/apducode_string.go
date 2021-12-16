// Code generated by "stringer -type APDUCode"; DO NOT EDIT.

package apps

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[APDUExecutionError-25600]
	_ = x[APDUEmptyBuffer-27010]
	_ = x[APDUOutputBufferTooSmall-27011]
	_ = x[APDUCommandNotAllowed-27014]
	_ = x[APDUINSNotSupported-27904]
	_ = x[APDUCLANotSupported-28160]
	_ = x[APDUUnknown-28416]
	_ = x[APDUSuccess-36864]
	_ = x[APDUWrongLength-26368]
	_ = x[APDUDataInvalid-27012]
}

const (
	_APDUCode_name_0 = "APDUExecutionError"
	_APDUCode_name_1 = "APDUWrongLength"
	_APDUCode_name_2 = "APDUEmptyBufferAPDUOutputBufferTooSmallAPDUDataInvalid"
	_APDUCode_name_3 = "APDUCommandNotAllowed"
	_APDUCode_name_4 = "APDUINSNotSupported"
	_APDUCode_name_5 = "APDUCLANotSupported"
	_APDUCode_name_6 = "APDUUnknown"
	_APDUCode_name_7 = "APDUSuccess"
)

var (
	_APDUCode_index_2 = [...]uint8{0, 15, 39, 54}
)

func (i APDUCode) String() string {
	switch {
	case i == 25600:
		return _APDUCode_name_0
	case i == 26368:
		return _APDUCode_name_1
	case 27010 <= i && i <= 27012:
		i -= 27010
		return _APDUCode_name_2[_APDUCode_index_2[i]:_APDUCode_index_2[i+1]]
	case i == 27014:
		return _APDUCode_name_3
	case i == 27904:
		return _APDUCode_name_4
	case i == 28160:
		return _APDUCode_name_5
	case i == 28416:
		return _APDUCode_name_6
	case i == 36864:
		return _APDUCode_name_7
	default:
		return "APDUCode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}