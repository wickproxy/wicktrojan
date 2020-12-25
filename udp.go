package main

import (
	"bufio"
	"errors"
	"net"
	"strconv"
	"time"
)

func handleUDP(inbound net.Conn, bufConn *bufio.Reader, ctx *requestCTX) {

	outbound, err := net.ListenUDP("udp", &net.UDPAddr{})
	if err != nil {
		info("[udp] can not dail to outbound", err)
		return
	}
	defer outbound.Close()

	go func() {
		var usage int64
		buf := BufferPool.Get().([]byte)
		buf = buf[0:cap(buf)]
		for {
			outbound.SetReadDeadline(time.Now().Add(udpTimeout))
			n, remoteAddr, rerr := outbound.ReadFromUDP(buf)
			outbound.SetReadDeadline(time.Time{})
			hostport := remoteAddr.String()
			host, port, aerr := net.SplitHostPort(hostport)
			if aerr != nil {
				info("[udp]", aerr)
				break
			}
			if n > 0 {
				data, werr := packageUDP(buf[:n], host, port)
				if werr != nil {
					break
				}
				nw, werr := inbound.Write(data)
				if werr != nil {
					break
				}
				usage += int64(nw)
			}
			if rerr != nil {
				break
			}
		}
		BufferPool.Put(buf)
		inbound.SetReadDeadline(time.Now())
		ctx.RUsage = usage
	}()

	var usage int64
	buf := BufferPool.Get().([]byte)
	buf = buf[0:cap(buf)]
	for {
		inbound.SetReadDeadline(time.Now().Add(udpTimeout))
		payload, remoteAddr, rerr := unpackageUDP(bufConn, *ctx)
		inbound.SetReadDeadline(time.Time{})
		if rerr != nil {
			info("[udp]", rerr)
			break
		}
		nw, err := outbound.WriteToUDP(payload, remoteAddr)
		if err != nil {
			break
		}
		usage += int64(nw)
	}
	BufferPool.Put(buf)
	outbound.SetReadDeadline(time.Now())
	ctx.SUsage = usage
	return
}

func packageUDP(payload []byte, host, port string) ([]byte, error) {
	data := make([]byte, 1, 256)
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.To4() != nil {
			data[0] = 0x01
			data = append(data, ip.To4()...)
		} else if ip.To16() != nil {
			data[0] = 0x04
			data = append(data, ip.To16()...)
		} else {
			return []byte{}, errors.New("host package error")
		}
	} else {
		data[0] = 0x03
		data = append(data, byte(len(host)))
		data = append(data, []byte(host)...)
	}

	l := len(payload)
	iport, err := strconv.Atoi(port)
	if err != nil {
		return []byte{}, errors.New("port package error")
	}
	data = append(data, byte(iport/256), byte(iport%256), byte(l/256), byte(l%256), '\r', '\n')
	return append(data, payload...), nil
}

func unpackageUDP(bufConn *bufio.Reader, ctx requestCTX) (payload []byte, remoteUDPAddr *net.UDPAddr, err error) {
	var reqhost, reqport string
	var reqlen int
	var buf [300]byte

	atype, err := bufConn.ReadByte()
	if err != nil {
		return payload, nil, err
	}
	switch atype {
	case 0x01:
		n, err := bufConn.Read(buf[:10])
		if err != nil || n != 10 {
			return payload, nil, errors.New("parse IPv4 error")
		}
		reqhost = net.IPv4(buf[0], buf[1], buf[2], buf[3]).String()
		reqport = strconv.Itoa(int(buf[4])*256 + int(buf[5]))
		reqlen = int(buf[6])*256 + int(buf[7])
	case 0x03:
		nd, err := bufConn.ReadByte()
		if err != nil {
			return payload, nil, errors.New("parse domain error")
		}
		ndl := int(nd)
		n, err := bufConn.Read(buf[:ndl+6])
		if err != nil || n != ndl+6 {
			return payload, nil, errors.New("parse domain error")
		}
		reqhost = string(buf[:ndl])
		reqport = strconv.Itoa(int(buf[ndl])*256 + int(buf[ndl+1]))
		reqlen = int(buf[ndl+2])*256 + int(buf[ndl+3])
	case 0x04:
		n, err := bufConn.Read(buf[:22])
		if err != nil || n != 22 {
			return payload, nil, errors.New("parse IPv6 error")
		}
		reqhost = net.IP(buf[:16]).String()
		reqport = strconv.Itoa(int(buf[16])*256 + int(buf[17]))
		reqlen = int(buf[18])*256 + int(buf[19])
	default:
		return payload, nil, errors.New("udp address type invalid")
	}

	tmpCTX := requestCTX{
		Username: ctx.Username,
		Host:     reqhost,
		Port:     reqport,
		UDP:      true,
	}
	if !checkRules(tmpCTX) {
		return payload, nil, errors.New("request is not allowed")
	}

	var data [65536]byte
	nr, err := bufConn.Read(data[:reqlen])
	if err != nil {
		return payload, nil, err
	}
	if nr != reqlen {
		return payload, nil, errors.New("unmatch payload size")
	}

	remoteUDPAddr, err = net.ResolveUDPAddr("udp", net.JoinHostPort(reqhost, reqport))
	return data[:nr], remoteUDPAddr, err
}
