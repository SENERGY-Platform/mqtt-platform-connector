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

package lib

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SENERGY-Platform/platform-connector-lib"
)

func sendError(writer http.ResponseWriter, msg string, additionalInfo ...int) {
	log.Println("DEBUG: send error:", msg)
	//sendError(writer, fmt.Sprintf(`{"result": { "error": "%s" }}`, msg), statusCode)
	_, err := fmt.Fprintf(writer, `{"result": { "error": "%s" }}`, msg)
	//_, err := fmt.Fprintf(writer, `{"result": "next"}`)
	//_, err := fmt.Fprintf(writer, `{"result": "ok"}`)
	if err != nil {
		log.Println("ERROR: unable to send error msg:", err, msg, additionalInfo)
	}
}

type PublishWebhookMsg struct {
	Username string `json:"username"`
	ClientId string `json:"client_id"`
	Topic    string `json:"topic"`
	Payload  string `json:"payload"`
}

type WebhookmsgTopic struct {
	Topic string `json:"topic"`
}

type SubscribeWebhookMsg struct {
	Username string            `json:"username"`
	Topics   []WebhookmsgTopic `json:"topics"`
}

type LoginWebhookMsg struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type EventHandler func(username string, topic string, payload string)

func AuthWebhooks(ctx context.Context, config Config, connector *platform_connector_lib.Connector) {
	router := http.NewServeMux()
	router.HandleFunc("/publish", func(writer http.ResponseWriter, request *http.Request) {
		msg := PublishWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::publish::jsondecoding", err)
			sendError(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if msg.Username != config.AuthClientId {
			payload, err := base64.StdEncoding.DecodeString(msg.Payload)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::base64decoding", err)
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			token, err := connector.Security().GetCachedUserToken(msg.Username)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::GetUserToken", err)
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			_, deviceId, localDeviceId, serviceId, localServiceId, err := ParseTopic(config.SensorTopicPattern, msg.Topic)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::ParseTopic", err)
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			if deviceId != "" {
				err = connector.HandleDeviceEventWithAuthToken(token, deviceId, serviceId, map[string]string{
					"payload": string(payload),
				})
			} else if localDeviceId != "" {
				err = connector.HandleDeviceRefEventWithAuthToken(token, localDeviceId, localServiceId, map[string]string{
					"payload": string(payload),
				})
			} else {
				err = errors.New("unable to identify device from topic")
			}
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::HandleEventWithAuthToken", err, deviceId, serviceId, localDeviceId, localServiceId)
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
		}
		fmt.Fprintf(writer, `{"result": "ok"}`)
	})

	router.HandleFunc("/subscribe", func(writer http.ResponseWriter, request *http.Request) {
		//{"username":"sepl","mountpoint":"","client_id":"sepl_mqtt_connector_1","topics":[{"topic":"$share/sepl_mqtt_connector/#","qos":2}]}
		msg := SubscribeWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::subscribe::jsondecoding", err)
			sendError(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if msg.Username != config.AuthClientId {
			token, err := connector.Security().GetCachedUserToken(msg.Username)
			if err != nil {
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			for _, topic := range msg.Topics {
				_, deviceId, localDeviceId, _, _, err := ParseTopic(config.ActuatorTopicPattern, topic.Topic)
				if err != nil {
					log.Println("ERROR: AuthWebhooks::subscribe::ParseTopic", err)
					sendError(writer, err.Error(), http.StatusUnauthorized)
					return
				}
				if deviceId != "" {
					_, err = connector.IotCache.WithToken(token).GetDevice(deviceId)
				} else if localDeviceId != "" {
					_, err = connector.IotCache.WithToken(token).GetDeviceByLocalId(localDeviceId)
				} else {
					err = errors.New("unable to identify device from topic")
				}
				if err != nil {
					log.Println("ERROR: AuthWebhooks::subscribe::CheckEndpointAuth", err)
					sendError(writer, err.Error(), http.StatusUnauthorized)
					return
				}
			}
		}
		fmt.Fprintf(writer, `{"result": "ok"}`)
	})

	router.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		//{"peer_addr":"172.20.0.30","peer_port":41310,"mountpoint":"","client_id":"sepl_mqtt_connector_1","username":"sepl","password":"sepl","clean_session":true}
		msg := LoginWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::login::jsondecoding", err)
			sendError(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if msg.Username != config.AuthClientId {
			token, err := connector.Security().GetUserToken(msg.Username, msg.Password)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::login::GetOpenidPasswordToken", err, msg)
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			if token == "" {
				sendError(writer, "access denied", http.StatusUnauthorized)
				return
			}
		}
		fmt.Fprintf(writer, `{"result": "ok"}`)
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
