package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

const (
	readTimeout time.Duration = 30 * time.Second
	udpTimeout                = 60 * time.Second
	tcpTimeout                = 120 * time.Second
)

type requestCTX struct {
	Username string
	Hex      string

	Host   string
	Port   string
	UDP    bool
	SUsage int64
	RUsage int64
}

func (ctx requestCTX) String(from string) (msg string) {
	if ctx.UDP {
		msg = fmt.Sprintf("[UDP] %v->%v:%v (%v)", from, ctx.Host, ctx.Port, ctx.Username)
	} else {
		msg = fmt.Sprintf("[TCP] %v->%v:%v (%v)", from, ctx.Host, ctx.Port, ctx.Username)
	}
	if ctx.SUsage > 0 {
		msg += fmt.Sprintf(" Send: %v byte", ctx.SUsage)
	}
	if ctx.RUsage > 0 {
		msg += fmt.Sprintf(" Recv: %v byte", ctx.RUsage)
	}
	return
}

func serve(conn net.Conn) {
	defer conn.Close()
	var ctx requestCTX
	var err error

	conn.SetReadDeadline(time.Now().Add(readTimeout))
	rewindConn := newRewindConn(conn, 2048)
	bufConn := bufio.NewReader(rewindConn)
	if err == nil {
		ctx, err = handshake(bufConn)
	}
	rewindConn.StopBuffering()
	if err != nil {
		info("[server] request error:", err)
		err = fallback(rewindConn)
		if err != nil {
			info("[fallback] ", err)
		}
		return
	}
	conn.SetReadDeadline(time.Time{})

	debug(ctx.String(conn.RemoteAddr().String()))

	if ctx.UDP {
		handleUDP(conn, bufConn, &ctx)
	} else {
		handleTCP(conn, bufConn, &ctx)
	}
	debug("[close]", ctx.String(conn.RemoteAddr().String()))
	conn.Close()
	updateUsage(ctx)
}

func handshake(bufConn *bufio.Reader) (ctx requestCTX, err error) {
	hex, err := bufConn.ReadString('\n')
	if err != nil {
		return ctx, err
	}
	if len(hex) < 56 {
		return ctx, errors.New("format error")
	}
	hex = hex[:56]
	username, ok := authenticate(hex)
	if username == "" {
		return ctx, errors.New("authenticate failed")
	} else if !ok {
		msg := fmt.Sprintf("error happened on user (%v), may be out of usage", username)
		return ctx, errors.New(msg)
	}
	ctx.Username = username
	ctx.Hex = hex

	request, err := readCRLF(bufConn)
	if err != nil {
		return ctx, err
	}
	if len(request) <= 2 {
		return ctx, errors.New("request too short")
	}

	if request[0] == 0x03 {
		ctx.UDP = true
	}

	switch request[1] {
	case 0x01:
		if len(request) < 8 {
			return ctx, errors.New("parse IPv4 error")
		}
		ctx.Host = net.IPv4(request[2], request[3], request[4], request[5]).String()
		ctx.Port = strconv.Itoa(int(request[6])*256 + int(request[7]))
	case 0x03:
		nd := int(request[2])
		if len(request) < 5+nd {
			return ctx, errors.New("parse domain error")
		}
		ctx.Host = string(request[3 : 3+nd])
		ctx.Port = strconv.Itoa(int(request[3+nd])*256 + int(request[4+nd]))
	case 0x04:
		if len(request) < 20 {
			return ctx, errors.New("parse IPv6 error")
		}
		ctx.Host = net.IP(request[2:18]).String()
		ctx.Port = strconv.Itoa(int(request[18])*256 + int(request[19]))
	default:
		return ctx, errors.New("invalid address type")
	}

	if !checkRules(ctx) {
		msg := fmt.Sprintf("[rule] ACL check not passed: [%v] %v:%v is not allowed", ctx.Username, ctx.Host, ctx.Port)
		return ctx, errors.New(msg)
	}
	return
}

func fallback(conn *rewindConn) error {
	if config.Fallback == "" {
		return errors.New("fallback url is empty")
	}

	outbound, err := net.Dial("tcp", config.Fallback)
	if err != nil {
		return errors.New("fallback error: " + err.Error())
	}
	defer outbound.Close()

	buf := BufferPool.Get().([]byte)
	defer BufferPool.Put(buf)

	conn.Rewind()
	n, err := conn.Read(buf)
	if err != nil {
		return errors.New("fallback error: " + err.Error())
	}
	outbound.Write(buf[:n])
	relay(outbound, conn, &requestCTX{})
	return nil
}

func handleTCP(inbound net.Conn, bufConn *bufio.Reader, ctx *requestCTX) {
	hostport := net.JoinHostPort(ctx.Host, ctx.Port)
	outbound, err := net.Dial("tcp", hostport)
	if err != nil {
		info("[outbound] connect to outbound error:", err)
		return
	}
	defer outbound.Close()

	buf := BufferPool.Get().([]byte)
	n := bufConn.Buffered()
	if n < cap(buf) {
		buf = buf[0:n]
	} else {
		buf = buf[0:cap(buf)]
	}
	tn, err := bufConn.Read(buf)
	outbound.Write(buf[:tn])
	if err != nil {
		info("[outbound] write to outbound error:", err)
		return
	}
	BufferPool.Put(buf)

	_ = relay(inbound, outbound, ctx)
}

func readCRLF(bufConn *bufio.Reader) ([]byte, error) {
	var request = make([]byte, 0, 32)
	for {
		by, err := bufConn.ReadByte()
		if err != nil {
			return request, err
		}
		if by != '\n' {
			request = append(request, by)
		} else {
			if len(request) > 0 && request[len(request)-1] == '\r' {
				request = request[:len(request)-1]
				break
			}
			request = append(request, by)
		}
	}
	return request, nil
}
