package p2pv2

import (
	"bufio"
	"encoding/binary"
	"io"

	"google.golang.org/protobuf/proto"
)

type WriteCloser interface {
	WriteMsg(proto.Message) error
	io.Closer
}

type ReadCloser interface {
	ReadMsg(msg proto.Message) error
	io.Closer
}

func NewDelimitedWriter(w io.Writer) WriteCloser {
	return &varintWriter{w, make([]byte, binary.MaxVarintLen64), nil}
}

type varintWriter struct {
	w      io.Writer
	lenBuf []byte
	buffer []byte
}

func getSize(v interface{}) (int, bool) {
	if sz, ok := v.(interface {
		Size() (n int)
	}); ok {
		return sz.Size(), true
	} else if sz, ok := v.(interface {
		ProtoSize() (n int)
	}); ok {
		return sz.ProtoSize(), true
	} else {
		return 0, false
	}
}

func (this *varintWriter) WriteMsg(msg proto.Message) (err error) {
	var data []byte

	// fallback
	data, err = proto.Marshal(msg)
	if err != nil {
		return err
	}
	length := uint64(len(data))
	n := binary.PutUvarint(this.lenBuf, length)
	_, err = this.w.Write(this.lenBuf[:n])
	if err != nil {
		return err
	}
	_, err = this.w.Write(data)
	return err
}

func (this *varintWriter) Close() error {
	if closer, ok := this.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func NewDelimitedReader(r io.Reader, maxSize int) ReadCloser {
	var closer io.Closer
	if c, ok := r.(io.Closer); ok {
		closer = c
	}
	return &varintReader{bufio.NewReader(r), nil, maxSize, closer}
}

type varintReader struct {
	r       *bufio.Reader
	buf     []byte
	maxSize int
	closer  io.Closer
}

func (this *varintReader) ReadMsg(msg proto.Message) error {
	length64, err := binary.ReadUvarint(this.r)
	if err != nil {
		return err
	}
	length := int(length64)
	if length < 0 || length > this.maxSize {
		return io.ErrShortBuffer
	}
	if len(this.buf) < length {
		this.buf = make([]byte, length)
	}
	buf := this.buf[:length]
	if _, err := io.ReadFull(this.r, buf); err != nil {
		return err
	}
	return proto.Unmarshal(buf, msg)
}

func (this *varintReader) Close() error {
	if this.closer != nil {
		return this.closer.Close()
	}
	return nil
}
