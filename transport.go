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

func relay(leftConn, rightConn net.Conn, ctx *requestCTX) int64 {
	ch := make(chan error)

	go func() {
		buf := BufferPool.Get().([]byte)
		buf = buf[0:cap(buf)]
		nr, err := io.CopyBuffer(rightConn, leftConn, buf)
		BufferPool.Put(buf)
		rightConn.SetReadDeadline(time.Now())
		ctx.SUsage = nr
		ch <- err
	}()

	buf := BufferPool.Get().([]byte)
	buf = buf[0:cap(buf)]
	n, _ := io.CopyBuffer(leftConn, rightConn, buf)
	BufferPool.Put(buf)
	leftConn.SetReadDeadline(time.Now())
	ctx.RUsage = n
	<-ch
	return n
}