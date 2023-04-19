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
	"log"
	"net/http"
	"time"
)

func InitWebhooks(ctx context.Context, config configuration.Config, connector *platform_connector_lib.Connector, connectionLog connectionlog.ConnectionLog) {
	topicParser := topic.New(connector.IotCache, config.ActuatorTopicPattern)
	router := http.NewServeMux()

	router.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		login(writer, request, config, connector, connectionLog)
	})

	router.HandleFunc("/online", func(writer http.ResponseWriter, request *http.Request) {
		online(writer, request, config, connectionLog)
	})

	router.HandleFunc("/disconnect", func(writer http.ResponseWriter, request *http.Request) {
		disconnect(writer, request, config, connectionLog)
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
