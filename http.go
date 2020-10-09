package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

type HttpServer struct {
	LocalAddr *net.TCPAddr
	Path      *string
	Link      *Link
}

func (ctx *HttpServer) Start() {
	defaultPage := func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			_, _ = io.WriteString(w, "It works!\n")
		} else {
			http.Error(w, "Page Not Found", http.StatusNotFound)
		}
	}
	http.HandleFunc("/", defaultPage)
	http.HandleFunc(*path, ctx.httpHandler)
	log.Fatal(http.ListenAndServe(ctx.LocalAddr.String(), nil))
}

func (ctx *HttpServer) httpHandler(w http.ResponseWriter, req *http.Request) {
	//Read request body
	var buf bytes.Buffer
	_, err := io.Copy(&buf, req.Body)
	if err != nil {
		logWarn("Http get request body failed:", err)
		return
	}
	//Forward request to internal link
	logRequest(&AccessLog{
		From:    req.RemoteAddr,
		To:      req.URL.String(),
		Payload: buf.String(),
	})
	if buf.Len() == 0 {
		logWarn("Http server got empty payload")
		http.Error(w, "Empty Request", http.StatusBadRequest)
		return
	}
	err = ctx.Link.Write(buf.Bytes())
	if err != nil {
		logWarn("Write request to internal link failed", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "Write request to internal link failed: "+err.Error())
		return
	}
	//Read response from internal link
	rep, err := ctx.Link.Read()
	if err != nil {
		logWarn("Read response from internal link failed", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "Read response from internal link failed: "+err.Error())
		return
	}
	//Forward response to remote host
	logResponse(&AccessLog{
		From:    "(internal)",
		To:      req.RemoteAddr,
		Payload: bytes.NewBuffer(rep).String(),
	})
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(rep)
	if err != nil {
		logWarn("Write response to http client failed", err)
	}
}

type HttpClient struct {
	RemoteHostPort string
	Path           string
	Link           *Link
}

func (ctx *HttpClient) Start() error {
	url := "http://" + ctx.RemoteHostPort + ctx.Path
	contentType := "application/octet-stream"
	client := http.Client{
		Timeout: time.Second * 5,
	}

	for {
		//Read from internal link
		rawBytes, err := ctx.Link.Read()
		if err != nil {
			logWarn("(Http Client) Read from internal link failed", err)
			continue
		}
		buf := bytes.NewBuffer(rawBytes)
		logRequest(&AccessLog{
			From:    "(internal)",
			To:      "POST " + url,
			Payload: buf.String(),
		})
		//Forward request through http protocol
		response, err := client.Post(url, contentType, buf)
		if err != nil {
			logWarn("Send http request failed: ", err)
			ctx.Link.SetError(err)
			continue
		}
		buf.Reset()
		_, err = io.Copy(buf, response.Body)
		if err != nil {
			logWarn("Receive http response failed: ", err)
			ctx.Link.SetError(err)
			continue
		}
		logResponse(&AccessLog{
			From:    url,
			To:      "(internal)",
			Payload: buf.String(),
		})
		//Write response to internal link
		err = ctx.Link.Write(buf.Bytes())
		if err != nil {
			logWarn("(Http Client) Write response to internal link failed: ", err)
			ctx.Link.SetError(err)
			continue
		}
	}
}
