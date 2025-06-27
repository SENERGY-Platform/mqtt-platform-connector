/*
 * Copyright 2018 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package vernemqtt

import (
	"context"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/connectionlog"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	"github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/swaggo/swag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

//go:generate go tool swag init -o ../../../docs --parseDependency -d .. -g vernemqtt/vernemqwebhook.go

// InitWebhooks doc
// @title         Mqtt-Connector-Webhooks
// @description   webhooks for vernemqtt; all responses are with code=200, differences in swagger doc are because of technical incompatibilities of the documentation format
// @version       0.1
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath  /
func InitWebhooks(ctx context.Context, config configuration.Config, connector *platform_connector_lib.Connector, connectionLog connectionlog.ConnectionLog) {
	topicParser := topic.New(connector.IotCache, config.ActuatorTopicPattern)
	router := http.NewServeMux()

	logger := GetLogger()

	router.HandleFunc("GET /doc", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		doc, err := swag.ReadDoc()
		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		//remove empty host to enable developer-swagger-api service to replace it; can not use cleaner delete on json object, because developer-swagger-api is sensible to formatting; better alternative is refactoring of developer-swagger-api/apis/db/db.py
		doc = strings.Replace(doc, `"host": "",`, "", 1)
		_, _ = writer.Write([]byte(doc))
	})

	router.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		login(writer, request, config, connector, connectionLog, logger)
	})

	router.HandleFunc("/online", func(writer http.ResponseWriter, request *http.Request) {
		online(writer, request, config, connectionLog)
	})

	router.HandleFunc("/disconnect", func(writer http.ResponseWriter, request *http.Request) {
		disconnect(writer, request, config, connectionLog, logger)
	})

	router.HandleFunc("/publish", func(writer http.ResponseWriter, request *http.Request) {
		publish(writer, request, config, connector, topicParser)
	})

	router.HandleFunc("/subscribe", func(writer http.ResponseWriter, request *http.Request) {
		subscribe(writer, request, config, connector, topicParser, connectionLog)
	})

	router.HandleFunc("/unsubscribe", func(writer http.ResponseWriter, request *http.Request) {
		unsubscribe(writer, request, config, connector, topicParser, connectionLog)
	})

	var handler http.Handler = router
	if config.Debug {
		handler = Logger(router)
	}
	server := &http.Server{Addr: ":" + config.WebhookPort, Handler: handler, WriteTimeout: 10 * time.Second, ReadTimeout: 2 * time.Second, ReadHeaderTimeout: 2 * time.Second}
	go func() {
		log.Println("Listening on ", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("ERROR: api server error", err)
			log.Fatal(err)
		}
	}()
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: webhook shutdown", server.Shutdown(context.Background()))
	}()
}

func Logger(handler http.Handler) *LoggerMiddleWare {
	return &LoggerMiddleWare{handler: handler}
}

type LoggerMiddleWare struct {
	handler http.Handler
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
	method := request.Method
	path := request.URL
	log.Printf("[%v] %v \n", method, path)
}

func GetLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	if info, ok := debug.ReadBuildInfo(); ok {
		logger = logger.With("go-module", info.Path)
	}
	return logger.With("snrgy-log-type", "connector-webhook")
}
