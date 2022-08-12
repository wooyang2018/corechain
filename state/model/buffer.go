package model

import (
	"errors"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoiface"
)

// Buffer is a buffer for encoding and decoding the protobuf wire format.
// It may be reused between invocations to reduce memory usage.
type Buffer struct {
	buf           []byte
	idx           int
	deterministic bool
}

// NewBuffer allocates a new Buffer initialized with buf,
// where the contents of buf are considered the unread portion of the buffer.
func NewBuffer(buf []byte) *Buffer {
	return &Buffer{buf: buf}
}

// SetBuf sets buf as the internal buffer,
// where the contents of buf are considered the unread portion of the buffer.
func (b *Buffer) SetBuf(buf []byte) {
	b.buf = buf
	b.idx = 0
}

// Reset clears the internal buffer of all written and unread data.
func (b *Buffer) Reset() {
	b.buf = b.buf[:0]
	b.idx = 0
}

// Bytes returns the internal buffer.
func (b *Buffer) Bytes() []byte {
	return b.buf
}

// Unread returns the unread portion of the buffer.
func (b *Buffer) Unread() []byte {
	return b.buf[b.idx:]
}

func marshalAppend(buf []byte, m proto.Message, deterministic bool) ([]byte, error) {
	var ErrNil = errors.New("proto: Marshal called with nil")
	if m == nil {
		return nil, ErrNil
	}
	nbuf, err := proto.MarshalOptions{
		Deterministic: deterministic,
		AllowPartial:  true,
	}.MarshalAppend(buf, m)
	if err != nil {
		return buf, err
	}
	if len(buf) == len(nbuf) {
		if !m.ProtoReflect().IsValid() {
			return buf, ErrNil
		}
	}
	return nbuf, proto.CheckInitialized(m)
}

// UnmarshalMerge parses a wire-format message in b and places the decoded results in m.
func UnmarshalMerge(b []byte, m proto.Message) error {
	out, err := proto.UnmarshalOptions{
		AllowPartial: true,
		Merge:        true,
	}.UnmarshalState(protoiface.UnmarshalInput{
		Buf:     b,
		Message: m.ProtoReflect(),
	})
	if err != nil {
		return err
	}
	if out.Flags&protoiface.UnmarshalInitialized > 0 {
		return nil
	}
	return proto.CheckInitialized(m)
}

// EncodeVarint appends an unsigned varint encoding to the buffer.
func (b *Buffer) EncodeVarint(v uint64) error {
	b.buf = protowire.AppendVarint(b.buf, v)
	return nil
}

// EncodeMessage appends a length-prefixed encoded message to the buffer.
func (b *Buffer) EncodeMessage(m proto.Message) error {
	var err error
	b.buf = protowire.AppendVarint(b.buf, uint64(proto.Size(m)))
	b.buf, err = marshalAppend(b.buf, m, b.deterministic)
	return err
}

// DecodeVarint consumes an encoded unsigned varint from the buffer.
func (b *Buffer) DecodeVarint() (uint64, error) {
	v, n := protowire.ConsumeVarint(b.buf[b.idx:])
	if n < 0 {
		return 0, protowire.ParseError(n)
	}
	b.idx += n
	return uint64(v), nil
}

// DecodeRawBytes consumes a length-prefixed raw bytes from the buffer.
// If alloc is specified, it returns a copy the raw bytes
// rather than a sub-slice of the buffer.
func (b *Buffer) DecodeRawBytes(alloc bool) ([]byte, error) {
	v, n := protowire.ConsumeBytes(b.buf[b.idx:])
	if n < 0 {
		return nil, protowire.ParseError(n)
	}
	b.idx += n
	if alloc {
		v = append([]byte(nil), v...)
	}
	return v, nil
}

// DecodeMessage consumes a length-prefixed message from the buffer.
// It does not reset m before unmarshaling.
func (b *Buffer) DecodeMessage(m proto.Message) error {
	v, err := b.DecodeRawBytes(false)
	if err != nil {
		return err
	}
	return UnmarshalMerge(v, m)
}
