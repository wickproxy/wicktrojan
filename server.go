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
		msg = fmt.Sprintf("[udp] %v->%v:%v (%v)", from, ctx.Host, ctx.Port, ctx.Username)
	} else {
		msg = fmt.Sprintf("[tcp] %v->%v:%v (%v)", from, ctx.Host, ctx.Port, ctx.Username)
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

	debug("[open]",ctx.String(conn.RemoteAddr().String()))

	if ctx.Host == config.PanelHost {
		handlePanel(conn, ctx)
		return
	}
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

	udp, err := bufConn.ReadByte()
	if err != nil {
		return ctx, errors.New("request too short")
	}
	if udp == 0x03 {
		ctx.UDP = true
	}
	atype, err := bufConn.ReadByte()
	if err != nil {
		return ctx, errors.New("request too short")
	}

	var buf [300]byte
	switch atype {
	case 0x01:
		n, err := bufConn.Read(buf[:8])
		if err != nil || n != 8 {
			return ctx, errors.New("parse IPv4 error")
		}
		ctx.Host = net.IPv4(buf[0], buf[1], buf[2], buf[3]).String()
		ctx.Port = strconv.Itoa(int(buf[4])*256 + int(buf[5]))
	case 0x03:
		nd, err := bufConn.ReadByte()
		if err != nil {
			return ctx, errors.New("parse domain error")
		}
		ndl := int(nd)
		n, err := bufConn.Read(buf[:ndl+4])
		if err != nil || n != ndl+4 {
			return ctx, errors.New("parse domain error")
		}
		ctx.Host = string(buf[:ndl])
		ctx.Port = strconv.Itoa(int(buf[ndl])*256 + int(buf[ndl+1]))
	case 0x04:
		n, err := bufConn.Read(buf[:20])
		if err != nil || n != 20 {
			return ctx, errors.New("parse IPv6 error")
		}
		ctx.Host = net.IP(buf[:16]).String()
		ctx.Port = strconv.Itoa(int(buf[16])*256 + int(buf[17]))
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

func handlePanel(inbound net.Conn, ctx requestCTX) {
	msg := fmt.Sprintf("Welcome to wicktrojan panel! User [%v] ", ctx.Username)
	if u, ok := users[ctx.Hex]; ok {
		usageLock.RLock()
		usage := formatUsage(u.Usage)
		usageLock.RUnlock()
		var quota string
		if u.Quota > 0 {
			quota = formatUsage(u.Quota)
		} else {
			quota = "INF"
		}
		msg += fmt.Sprintf("(%v/%v)", usage, quota)
	}
	msg = "HTTP/1.1 200 OK\r\nContent-Length:" + strconv.Itoa(len(msg)) + "\r\n\r\n" + msg
	inbound.Write([]byte(msg))
}

func formatUsage(usage int64) string {
	list := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	idx := 0
	usageF := float64(usage)
	for usageF > 1024.0 && idx <= 5 {
		usageF = usageF / 1024.0
		idx = idx + 1
	}
	return fmt.Sprintf("%.2f %v", usageF, list[idx])
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
