package main

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

type fakeHTTPResponseWriter struct {
	http.Hijacker
	http.ResponseWriter
	ReadWriter *bufio.ReadWriter
	Conn       net.Conn
}

func (w *fakeHTTPResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.Conn, w.ReadWriter, nil
}

func serveWebSocket(conn net.Conn) {

	conn.SetReadDeadline(time.Now().Add(readTimeout))
	rewindConn := newRewindConn(conn, 2048)
	rw := bufio.NewReadWriter(bufio.NewReader(rewindConn), bufio.NewWriter(rewindConn))
	req, err := http.ReadRequest(rw.Reader)
	rewindConn.StopBuffering()
	if err != nil {
		info("[websocket] parse http request error:", err)
		fallback(rewindConn)
		return
	}
	if strings.ToLower(req.Header.Get("Upgrade")) != "websocket" || req.URL.Path != config.Websocket.Path {
		info("[websocket] url is not match:", err)
		fallback(rewindConn)
		return
	}

	url := "wss://" + config.Websocket.Host + config.Websocket.Host
	origin := "https://" + config.Websocket.Host
	wsConfig, err := websocket.NewConfig(url, origin)

	wsServer := websocket.Server{
		Config: *wsConfig,
		Handler: func(conn *websocket.Conn) {
			serveTrojan(conn)
		},
	}

	respWriter := &fakeHTTPResponseWriter{
		Conn:       conn,
		ReadWriter: rw,
	}
	wsServer.ServeHTTP(respWriter, req)
}
