package iot

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func NewLogger(handler http.Handler, logLevel string) *LoggerMiddleWare {
	return &LoggerMiddleWare{handler: handler, logLevel: logLevel}
}

type LoggerMiddleWare struct {
	handler  http.Handler
	logLevel string //DEBUG | CALL | NONE
}

func (this *LoggerMiddleWare) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	this.log(r)
	if this.handler != nil {
		this.handler.ServeHTTP(w, r)
	} else {
		http.Error(w, "Forbidden", 403)
	}
}

func (this *LoggerMiddleWare) log(request *http.Request) {
	if this.logLevel != "NONE" {
		method := request.Method
		path := request.URL

		if this.logLevel == "CALL" {
			log.Printf("[%v] %v \n", method, path)
		}

		if this.logLevel == "DEBUG" {
			//read on request.Body would empty it -> create new ReadCloser for request.Body while reading
			var buf bytes.Buffer
			temp := io.TeeReader(request.Body, &buf)
			b, err := ioutil.ReadAll(temp)
			if err != nil {
				log.Println("ERROR: read error in debuglog:", err)
			}
			request.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))

			client := request.RemoteAddr
			log.Printf("%v [%v] %v %v", client, method, path, string(b))

		}

	}
}
