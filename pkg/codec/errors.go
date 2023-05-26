package codec

import "errors"

var (
	// ErrInvalidData represents general invalid data.
	ErrInvalidData = errors.New("invalid data")
	// ErrOutOfRange represents logic accessing data in out of range.
	ErrOutOfRange = errors.New("out of range")
	// ErrNoTerminate represents invalid protobuf format.
	ErrNoTerminate = errors.New("no terminationg bit found")
	// ErrUnexpectedFieldNumber represents protobuf field number not matching expected value.
	ErrUnexpectedFieldNumber = errors.New("unexpected field number found")
	// ErrFieldNumberNotFound represents protobuf field number not matching expected value.
	ErrFieldNumberNotFound = errors.New("expected field number does not exist")
	// ErrUnreadBytes represents extra bytes not read.
	ErrUnreadBytes = errors.New("unread bytes exist")
	// ErrUnnecessaryLeadingBytes represents error for which varint has unnecessary 0x80 byte at the beginning.
	ErrUnnecessaryLeadingBytes = errors.New("unnecessary leading bytes for varint found")
)
