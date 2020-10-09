package main

//go:generate errorgen

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var (
	VERSION = "custom"
)

var (
	fastOpen   = flag.Bool("fast-open", false, "Enable TCP fast open.")
	localAddr  = flag.String("localAddr", "127.0.0.1", "local address to listen on.")
	localPort  = flag.String("localPort", "1984", "local port to listen on.")
	remoteAddr = flag.String("remoteAddr", "127.0.0.1", "remote address to forward.")
	remotePort = flag.String("remotePort", "1080", "remote port to forward.")
	path       = flag.String("path", "/api", "URL path for communicate, in server & client this must equal.")
	host       = flag.String("host", "cloudfront.com", "Hostname for server.")
	tlsEnabled = flag.Bool("tls", true, "Enable TLS, client only.")
	serverMode = flag.Bool("server", false, "Run in server mode")
	logLevel   = flag.String("loglevel", "", "loglevel: debug, info, warning (default), error, none.")
	version    = flag.Bool("version", false, "Show version then exit")

	localEndpoint  *net.TCPAddr
	remoteEndpoint *net.TCPAddr
	remoteHostPort string
)

func parseOpts() error {
	opts, err := parseEnv()

	if err == nil {
		if _, b := opts.Get("tls"); b {
			*tlsEnabled = true
		}
		if c, b := opts.Get("host"); b {
			*host = c
		}
		if c, b := opts.Get("path"); b {
			*path = c
		}
		if c, b := opts.Get("loglevel"); b {
			*logLevel = c
		}
		if _, b := opts.Get("server"); b {
			*serverMode = true
		}
		if c, b := opts.Get("localAddr"); b {
			if *serverMode {
				*remoteAddr = c
			} else {
				*localAddr = c
			}
		}
		if c, b := opts.Get("localPort"); b {
			if *serverMode {
				*remotePort = c
			} else {
				*localPort = c
			}
		}
		if c, b := opts.Get("remoteAddr"); b {
			if *serverMode {
				*localAddr = c
			} else {
				*remoteAddr = c
			}
		}
		if c, b := opts.Get("remotePort"); b {
			if *serverMode {
				*localPort = c
			} else {
				*remotePort = c
			}
		}

		if _, b := opts.Get("fastOpen"); b {
			*fastOpen = true
		}

		localEndpoint, err = net.ResolveTCPAddr("tcp", *localAddr+":"+*localPort)
		if err != nil {
			log.Fatal("Resolve local address failed: ", *localAddr, ":", *localPort, err)
		}
		remoteHostPort = *remoteAddr + ":" + *remotePort
		remoteEndpoint, err = net.ResolveTCPAddr("tcp", remoteHostPort)
		if err != nil {
			log.Fatal("Resolve remote address failed: ", *remoteAddr, ":", *remotePort, err)
		}
	}

	return err
}

func printVersion() {
	fmt.Println("http-plugin", VERSION)
	fmt.Println("Go version", runtime.Version())
	fmt.Println("Yet another SIP003 plugin for shadowsocks")
}

func main() {
	flag.Parse()
	if err := parseOpts(); err != nil {
		log.Fatal("failed to parse opts:", err)
	}

	if *version {
		printVersion()
		return
	}

	logInit()

	link := makeLink()

	if *serverMode {
		server := HttpServer{
			LocalAddr: localEndpoint,
			Path:      path,
			Link:      link,
		}
		go server.Start()
		client := FreeClient{
			RemoteAddr: remoteEndpoint,
			Link:       link,
		}
		go client.Start()
		logWarn("Plugin(SERVER mode) start successfully, Listening on ", localEndpoint)
	} else {
		server := FreeServer{
			LocalAddr: localEndpoint,
			Link:      link,
		}
		go server.Start()
		client := HttpClient{
			RemoteHostPort: remoteHostPort,
			Path:           *path,
			Link:           link,
		}
		go client.Start()
		logWarn("Plugin(CLIENT mode) start successfully, Server address ", remoteHostPort)
	}

	{
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-osSignals
	}
}
