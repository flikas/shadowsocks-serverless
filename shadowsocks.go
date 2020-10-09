package main

import (
	"bytes"
	"log"
	"net"
)

type FreeServer struct {
	LocalAddr *net.TCPAddr
	Link      *Link
}

func (ctx *FreeServer) Start() error {
	listener, err := net.ListenTCP("tcp", ctx.LocalAddr)
	if err != nil {
		return err
	}
	defer listener.Close()
	logInfo("Listening on ", ctx.LocalAddr)

	var conn *net.TCPConn
	for {
		conn, err = listener.AcceptTCP()
		if err != nil {
			logWarn("Error Accepting: ", err)
			continue
		}
		break
	}
	logAccess(&AccessLog{
		From: conn.RemoteAddr().String(),
		To:   conn.LocalAddr().String(),
	})

	defer conn.Close()
	for {
		//Read from tcp socket
		buf, err := readAll(conn)
		if err != nil {
			logWarn("Read from ss-client failed", err)
			log.Fatal(err)
		}
		logRequest(&AccessLog{
			From:    "(ss-client) " + conn.RemoteAddr().String(),
			To:      conn.LocalAddr().String(),
			Payload: buf.String(),
		})
		//Forward request to internal link
		if err = ctx.Link.Write(buf.Bytes()); err != nil {
			logWarn("Write to internal link failed", err)
			continue
		}
		//Read response from internal link
		rep, err := ctx.Link.Read()
		if err != nil {
			logWarn("Read to internal link failed", err)
			ctx.Link.SetError(err)
			continue
		}
		logResponse(&AccessLog{
			From:    "(internal)",
			To:      conn.LocalAddr().String(),
			Payload: bytes.NewBuffer(rep).String(),
		})
		//Forward response to tcp socket
		_, err = conn.Write(rep)
		if err != nil {
			logWarn("Write to ss-client failed", err)
			ctx.Link.SetError(err)
			continue
		}
	}
}

type FreeClient struct {
	RemoteAddr *net.TCPAddr
	Link       *Link
}

func (ctx *FreeClient) Start() {
	var conn *net.TCPConn

	for {
		//Read from internal link
		rawBytes, err := ctx.Link.Read()
		if err != nil {
			logWarn("(Free Client) Read from internal link failed", err)
			continue
		}
		if conn == nil {
			conn, err = net.DialTCP("tcp", nil, ctx.RemoteAddr)
			if err != nil {
				log.Fatal("Connect to shadowsocks failed: ", err)
			}
			logAccess(&AccessLog{
				From: conn.LocalAddr().String(),
				To:   conn.RemoteAddr().String(),
			})
			defer conn.Close()
		}
		logRequest(&AccessLog{
			From:    "(internal)",
			To:      "(ss-server) " + ctx.RemoteAddr.String(),
			Payload: bytes.NewBuffer(rawBytes).String(),
		})
		//Forward request to ss-server
		_, err = conn.Write(rawBytes)
		if err != nil {
			logWarn("Forward request to ss-server failed: ", err)
			ctx.Link.SetError(err)
			continue
		}
		//Read response from ss-server
		rep, err := readAll(conn)
		if err != nil {
			logWarn("Receive ss-server response failed: ", err)
			ctx.Link.SetError(err)
			continue
		}
		logResponse(&AccessLog{
			From:    "(ss-server) " + ctx.RemoteAddr.String(),
			To:      "(internal)",
			Payload: rep.String(),
		})
		//Write response to internal link
		err = ctx.Link.Write(rep.Bytes())
		if err != nil {
			logWarn("(Free Client) Write response to internal link failed: ", err)
			ctx.Link.SetError(err)
			continue
		}
	}
}

func readAll(conn *net.TCPConn) (*bytes.Buffer, error) {
	buf := make([]byte, 30*1024)
	l, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(buf[:l]), nil
}
