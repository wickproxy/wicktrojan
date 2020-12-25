package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

var (
	version     string
	buildTime   string
	debugFlag   = flag.Bool("debug", false, "enable debug mode")
	usageFile   = flag.String("usage", "", "usage database, left empty and usage information will not be stored")
	loggingFile = flag.String("log", "", "log file, left empty and use standard output")
	configFile  = flag.String("config", "config.toml", "config file")
	versionFlag = flag.Bool("version", false, "show version")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println(getVersion())
		flag.PrintDefaults()
	}
	flag.Parse()
}

func main() {
	if *versionFlag {
		fmt.Println(getVersion())
		os.Exit(0)
	}
	if *loggingFile != "" {
		logWriter, err := os.Create(*loggingFile)
		if err != nil {
			fatal("[log] can not open (or create) logging file:", err)
		}
		log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
		log.SetOutput(logWriter)
	}
	loadConfig()
	listenAndServe()
}

func getVersion() string {
	return "Version: " + version + " build: " + buildTime + " (platform: " + runtime.GOOS + "-" + runtime.GOARCH + ")."
}

func debug(args ...interface{}) {
	if *debugFlag {
		log.Println(args...)
	}
}

func info(args ...interface{}) {
	log.Println(args...)
}

func fatal(args ...interface{}) {
	log.Fatalln(args...)
}

func listenAndServe() {
	var ln net.Listener
	var err error

	var tlsUsed bool
	var tlsConfig *tls.Config

	if (config.TLS.Certificate != "" && config.TLS.PrivateKey != "") || config.TLS.IssueHost != "" {
		tlsUsed = true
		if len(config.TLS.NextProtos) == 0 {
			config.TLS.NextProtos = []string{"http/1.1"}
		}
		tlsConfig = &tls.Config{
			NextProtos: config.TLS.NextProtos,
			MinVersion: tls.VersionTLS12,
		}

		if config.TLS.IssueHost != "" {
			m := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(config.TLS.IssueHost),
			}
			if config.TLS.IssueStore != "" {
				m.Cache = autocert.DirCache(config.TLS.IssueStore)
			}
			tlsConfig.GetCertificate = m.GetCertificate
			tlsConfig.NextProtos = append(config.TLS.NextProtos, acme.ALPNProto)
		} else {
			cert, err := tls.LoadX509KeyPair(config.TLS.Certificate, config.TLS.PrivateKey)
			if err != nil {
				fatal("[tls] read certificate error:", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}

	ln, err = net.Listen("tcp", config.Listen)
	if err != nil {
		fatal("[server] server listen failed:", err)
	}
	info("[server] start to serve at", config.Listen)
	var tempDelay time.Duration
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				debug("temporary error delay for ", tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			info("accept connect error: ", err)
			continue
		}
		tempDelay = 0

		if tlsUsed {
			tlsConn := tls.Server(conn, tlsConfig)
			err = tlsConn.Handshake()
			if err != nil {
				info("[tls] tls handshake error:", err)
				tlsConn.Close()
				conn.Close()
				continue
			}
			go serve(tlsConn)
		} else {
			go serve(conn)
		}
	}
}
