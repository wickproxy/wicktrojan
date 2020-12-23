package main

import (
	"io"
	"net"
)

type rewinder struct {
	raw        io.Reader
	buf        []byte
	bufReadIdx int
	rewound    bool
	buffering  bool
	bufferSize int
}

func (r *rewinder) Read(p []byte) (int, error) {
	if r.rewound {
		if len(r.buf) > r.bufReadIdx {
			n := copy(p, r.buf[r.bufReadIdx:])
			r.bufReadIdx += n
			return n, nil
		}
		r.rewound = false
	}
	n, err := r.raw.Read(p)
	if r.buffering {
		r.buf = append(r.buf, p[:n]...)
		if len(r.buf) > r.bufferSize*2 {
			debug("[rewinder] read too many bytes!")
		}
	}
	return n, err
}

func (r *rewinder) ReadByte() (byte, error) {
	buf := [1]byte{}
	_, err := r.Read(buf[:])
	return buf[0], err
}

func (r *rewinder) Discard(n int) (int, error) {
	buf := [128]byte{}
	if n < 128 {
		return r.Read(buf[:n])
	}
	for discarded := 0; discarded+128 < n; discarded += 128 {
		_, err := r.Read(buf[:])
		if err != nil {
			return discarded, err
		}
	}
	if rest := n % 128; rest != 0 {
		return r.Read(buf[:rest])
	}
	return n, nil
}

func (r *rewinder) Rewind() {
	if r.bufferSize == 0 {
		panic("[rewinder] no buffer")
	}
	r.rewound = true
	r.bufReadIdx = 0
}

func (r *rewinder) StopBuffering() {
	r.buffering = false
}

func (r *rewinder) SetBufferSize(size int) {
	if size == 0 {
		if !r.buffering {
			panic("[rewinder] reader is disabled")
		}
		r.buffering = false
		r.buf = nil
		r.bufReadIdx = 0
		r.bufferSize = 0
	} else {
		if r.buffering {
			panic("[rewinder] reader is buffering")
		}
		r.buffering = true
		r.bufReadIdx = 0
		r.bufferSize = size
		r.buf = make([]byte, 0, size)
	}
}

type rewindConn struct {
	net.Conn
	*rewinder
}

func (c *rewindConn) Read(p []byte) (n int, err error) {
	return c.rewinder.Read(p)
}

func newRewindConn(conn net.Conn, bufferSize int) *rewindConn {
	ret := &rewindConn{
		Conn: conn,
		rewinder: &rewinder{
			raw: conn,
		},
	}
	if bufferSize == 0 {
		bufferSize = 2048
	}
	ret.SetBufferSize(bufferSize)
	return ret
}
