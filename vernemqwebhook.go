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

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/SENERGY-Platform/platform-connector-lib"
)

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

func AuthWebhooks(connector *platform_connector_lib.Connector) {
	http.HandleFunc("/publish", func(writer http.ResponseWriter, request *http.Request) {
		msg := PublishWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::publish::jsondecoding", err)
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if msg.Username != Config.AuthClientId {
			payload, err := base64.StdEncoding.DecodeString(msg.Payload)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::base64decoding", err)
				http.Error(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			token, err := connector.GetUserToken(msg.Username)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::GetUserToken", err)
				http.Error(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			connector.HandleEventWithAuthToken(token, msg.Topic, map[string]string{
				"topic":   msg.Topic,
				"payload": string(payload),
			})
		}
		fmt.Fprintf(writer, `{"result": "ok"}`)
	})

	http.HandleFunc("/subscribe", func(writer http.ResponseWriter, request *http.Request) {
		//{"username":"sepl","mountpoint":"","client_id":"sepl_mqtt_connector_1","topics":[{"topic":"$share/sepl_mqtt_connector/#","qos":2}]}
		msg := SubscribeWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::subscribe::jsondecoding", err)
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if msg.Username != Config.AuthClientId {
			token, err := connector.GetUserToken(msg.Username)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			for _, topic := range msg.Topics {
				err = connector.CheckEndpointAuth(token, topic.Topic)
				if err != nil {
					log.Println("ERROR: AuthWebhooks::subscribe::CheckEndpointAuth", err)
					http.Error(writer, err.Error(), http.StatusUnauthorized)
					return
				}
			}
		}
		fmt.Fprintf(writer, `{"result": "ok"}`)
	})

	http.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		//{"peer_addr":"172.20.0.30","peer_port":41310,"mountpoint":"","client_id":"sepl_mqtt_connector_1","username":"sepl","password":"sepl","clean_session":true}
		msg := LoginWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::login::jsondecoding", err)
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if msg.Username != Config.AuthClientId {
			token, err := connector.GetOpenidPasswordToken(msg.Username, msg.Password)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::login::GetOpenidPasswordToken", err, msg)
				http.Error(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			if token.AccessToken == "" {
				http.Error(writer, "access denied", http.StatusUnauthorized)
				return
			}
		}
		fmt.Fprintf(writer, `{"result": "ok"}`)
	})

	log.Fatal(http.ListenAndServe(":8080", http.DefaultServeMux))
}
