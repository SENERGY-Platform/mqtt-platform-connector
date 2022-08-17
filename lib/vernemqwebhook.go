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
	"fmt"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/connectionlog"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	"github.com/SENERGY-Platform/platform-connector-lib"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

func sendError(writer http.ResponseWriter, msg string, additionalInfo ...int) {
	log.Println("DEBUG: send error:", msg)
	err := json.NewEncoder(writer).Encode(map[string]map[string]string{"result": {"error": msg}})
	if err != nil {
		log.Println("ERROR: unable to send error msg:", err, msg, additionalInfo)
	}
}

type PublishWebhookMsg struct {
	Username string `json:"username"`
	ClientId string `json:"client_id"`
	Topic    string `json:"topic"`
	Payload  string `json:"payload"`
	Qos      int    `json:"qos"`
}

type WebhookmsgTopic struct {
	Topic string `json:"topic"`
	Qos   int    `json:"qos"`
}

type SubscribeWebhookMsg struct {
	ClientId string            `json:"client_id"`
	Username string            `json:"username"`
	Topics   []WebhookmsgTopic `json:"topics"`
}

type SubscribeWebhookResult struct {
	Result string            `json:"result"`
	Topics []WebhookmsgTopic `json:"topics"`
}

type LoginWebhookMsg struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	ClientId     string `json:"client_id"`
	CleanSession bool   `json:"clean_session"`
}

type OnlineWebhookMsg struct {
	ClientId string `json:"client_id"`
}

type DisconnectWebhookMsg struct {
	ClientId string `json:"client_id"`
}

type UnsubscribeWebhookMsg struct {
	ClientId string   `json:"client_id"`
	Username string   `json:"username"`
	Topics   []string `json:"topics"`
}

type EventHandler func(username string, topic string, payload string)

func AuthWebhooks(ctx context.Context, config Config, connector *platform_connector_lib.Connector, connectionLog connectionlog.ConnectionLog) {
	topicParser := topic.New(connector.IotCache, config.ActuatorTopicPattern)
	router := http.NewServeMux()

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
			connectionLog.SetCleanSession(msg.ClientId, msg.CleanSession)
		}
		fmt.Fprintf(writer, `{"result": "ok"}`)
	})

	router.HandleFunc("/online", func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				debug.PrintStack()
				sendError(writer, fmt.Sprint(p))
				return
			} else {
				fmt.Fprint(writer, `{}`)
			}
		}()
		msg := OnlineWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			sendError(writer, err.Error())
			return
		}
		if config.Debug {
			log.Println("DEBUG: /online", msg)
		}
		connectionLog.Connect(msg.ClientId)
	})

	router.HandleFunc("/disconnect", func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				debug.PrintStack()
				sendError(writer, fmt.Sprint(p))
				return
			} else {
				fmt.Fprintf(writer, "{}")
			}
		}()
		msg := DisconnectWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: InitWebhooks::disconnect::jsondecoding", err)
			return
		}
		if config.Debug {
			log.Println("DEBUG: /disconnect", msg)
		}
		connectionLog.Disconnect(msg.ClientId)
	})

	router.HandleFunc("/publish", func(writer http.ResponseWriter, request *http.Request) {
		msg := PublishWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::publish::jsondecoding", err)
			sendError(writer, err.Error())
			return
		}
		if msg.Username != config.AuthClientId {
			payload, err := base64.StdEncoding.DecodeString(msg.Payload)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::base64decoding", err)
				sendError(writer, err.Error())
				return
			}
			token, err := connector.Security().GetCachedUserToken(msg.Username)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::GetUserToken", err)
				sendError(writer, err.Error())
				return
			}

			device, service, err := topicParser.Parse(token, msg.Topic)
			if err == topic.ErrNoDeviceIdCandidateFound {
				//topics that cant possibly be connected to a device may be handled at will
				if config.Debug {
					log.Println("WARNING: AuthWebhooks::publish::ParseTopic", err, msg.Topic)
				}
				fmt.Fprintf(writer, `{"result": "ok"}`)
				return
			}
			if err == topic.ErrNoServiceMatchFound {
				//we want to only check device access
				if config.Debug {
					log.Println("WARNING: AuthWebhooks::publish::ParseTopic", err, msg.Topic)
				}
				fmt.Fprintf(writer, `{"result": "ok"}`)
				return
			}
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::Parse", err)
				sendError(writer, err.Error())
				return
			}
			err = connector.HandleDeviceIdentEventWithAuthToken(token, device.Id, device.LocalId, service.Id, service.LocalId, map[string]string{
				"payload": string(payload),
			}, platform_connector_lib.Qos(msg.Qos))
			if err != nil {
				if config.Debug {
					log.Println("WARNING: AuthWebhooks::publish::HandleDeviceIdentEventWithAuthToken", err, device.Id, device.LocalId, service.Id, service.LocalId, msg.Topic)
				}
				fmt.Fprintf(writer, `{"result": "ok"}`)
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
		resultTopics := []WebhookmsgTopic{}
		if msg.Username != config.AuthClientId {
			token, err := connector.Security().GetCachedUserToken(msg.Username)
			if err != nil {
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			for _, mqtttopic := range msg.Topics {
				device, _, err := topicParser.Parse(token, mqtttopic.Topic)
				if err == topic.ErrNoServiceMatchFound {
					//we want to only check device access
					err = nil
				}
				if err == nil {
					connectionLog.Subscribe(msg.ClientId, mqtttopic.Topic, device.Id)
				}
				if err == topic.ErrMultipleMatchingDevicesFound || err == topic.ErrNoDeviceMatchFound {
					//no err but disallow subscription
					mqtttopic.Qos = 128
					err = nil
				}
				if err != nil && err != topic.ErrNoDeviceIdCandidateFound {
					log.Println("WARNING: AuthWebhooks::subscribe::ParseTopic", err, mqtttopic.Topic)
					sendError(writer, err.Error())
					return
				}
				resultTopics = append(resultTopics, mqtttopic)
			}
		}
		err = json.NewEncoder(writer).Encode(SubscribeWebhookResult{
			Result: "ok",
			Topics: resultTopics,
		})
		if err != nil {
			log.Println("ERROR: AuthWebhooks::subscribe::SubscribeWebhookResult", err)
		}
	})

	router.HandleFunc("/unsubscribe", func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				debug.PrintStack()
				sendError(writer, fmt.Sprint(p))
				return
			}
		}()
		msg := UnsubscribeWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			sendError(writer, err.Error())
			return
		}
		if config.Debug {
			log.Println("DEBUG: /unsubscribe", msg)
		}
		//defer json.NewEncoder(writer).Encode(map[string]interface{}{"result": "ok", "topics": msg.Topics})
		defer json.NewEncoder(writer).Encode(map[string]interface{}{"result": "ok", "topics": msg.Topics})
		if msg.Username != config.AuthClientId {
			token, err := connector.Security().GetCachedUserToken(msg.Username)
			if err != nil {
				log.Println("ERROR: InitWebhooks::unsubscribe::GenerateUserToken", err)
				return
			}
			for _, topic := range msg.Topics {
				device, _, err := topicParser.Parse(token, topic)
				if err != nil {
					log.Println("ERROR: InitWebhooks::unsubscribe::parseTopic", err)
					return
				}
				connectionLog.Unsubscribe(msg.ClientId, topic, device.Id)
			}
		}
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
