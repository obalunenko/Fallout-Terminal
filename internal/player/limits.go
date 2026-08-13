package player

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

const (
	// MaxUncompressedMessageBytes is the effective protobuf message limit for
	// every public player request, including unknown fields.
	MaxUncompressedMessageBytes = 4 << 10
	// MaxEncodedBodyBytes permits bounded Connect framing overhead while
	// rejecting oversized HTTP input before application adapters run.
	MaxEncodedBodyBytes = 8 << 10
	// MaxDecompressedMessageBytes prevents compressed input from expanding past
	// the same effective player-controlled protobuf limit.
	MaxDecompressedMessageBytes = 4 << 10
)

// ErrResourceExhausted identifies a transport/message bound violation without
// retaining or exposing the rejected request bytes.
var ErrResourceExhausted = errors.New("public player request exceeds configured limit")

// ValidateMessageSize counts known and unknown protobuf fields and rejects the
// value before any canonical adapter or service invocation.
func ValidateMessageSize(message proto.Message) error {
	if message == nil {
		return nil
	}
	if size := proto.Size(message); size > MaxUncompressedMessageBytes {
		return fmt.Errorf("%w: protobuf message is %d bytes; maximum is %d", ErrResourceExhausted, size, MaxUncompressedMessageBytes)
	}
	return nil
}

// ReadEncodedBody reads one bounded encoded HTTP body without retaining a
// prefix when the body crosses the configured limit.
func ReadEncodedBody(reader io.Reader) ([]byte, error) {
	if reader == nil {
		return nil, nil
	}
	data, err := io.ReadAll(io.LimitReader(reader, MaxEncodedBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read public player request: %w", err)
	}
	if len(data) > MaxEncodedBodyBytes {
		return nil, fmt.Errorf("%w: encoded body maximum is %d bytes", ErrResourceExhausted, MaxEncodedBodyBytes)
	}
	return data, nil
}
