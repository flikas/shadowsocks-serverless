// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !android

package main

import (
	"fmt"
	"log"
)

func logInit() {
}

func logWarn(v ...interface{}) {
	log.Println("[WARN]", fmt.Sprint(v...))
}

func logInfo(v ...interface{}) {
	log.Println("[INFO]", fmt.Sprint(v...))
}

func logDebug(v ...interface{}) {
	log.Println("[DEBUG]", fmt.Sprint(v...))
}

type AccessLog struct {
	From    string
	To      string
	Payload string
}

func logAccess(access *AccessLog) {
	logInfo("Connection established ", access.From, " -> ", access.To)
}

func logRequest(access *AccessLog) {
	if access.Payload != "" {
		logDebug("Request " + access.From + " -> " + access.To + ", Payload:\n" + access.Payload)
	} else {
		logDebug("Request " + access.From + " -> " + access.To)
	}
}

func logResponse(access *AccessLog) {
	if access.Payload != "" {
		logDebug("Response " + access.To + " <- " + access.From + ", Payload:\n" + access.Payload)
	} else {
		logDebug("Response " + access.To + " <- " + access.From)
	}
}
