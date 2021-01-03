package main

import (
	"io"
	"net"
	"sync"
	"time"
)

// BufferPool is for data copy
var BufferPool sync.Pool

// PoolInit to initialize BufferPool
func init() {
	makeBuffer := func() interface{} { return make([]byte, 0, 65536) }
	BufferPool = sync.Pool{New: makeBuffer}
}

type closeWriter interface {
	CloseWrite() error
}

func relay(inbound, outbound net.Conn, ctx *requestCTX) int64 {
	ch := make(chan error)

	go func() {
		buf := BufferPool.Get().([]byte)
		buf = buf[0:cap(buf)]
		nr, err := io.CopyBuffer(outbound, inbound, buf)
		BufferPool.Put(buf)
		outbound.SetReadDeadline(time.Now())
		ctx.SUsage = nr
		ch <- err
	}()

	buf := BufferPool.Get().([]byte)
	buf = buf[0:cap(buf)]
	var n int64
	if config.Reshape {
		sw := shapeWriter{}
		sw.init(inbound)
		n, _ = io.CopyBuffer(sw, outbound, buf)
	} else {
		n, _ = io.CopyBuffer(inbound, outbound, buf)
	}
	BufferPool.Put(buf)
	inbound.SetReadDeadline(time.Now())
	ctx.RUsage = n
	<-ch
	return n
}