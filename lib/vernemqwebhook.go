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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

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

func AuthWebhooks(connector *platform_connector_lib.Connector) {
	http.HandleFunc("/publish", func(writer http.ResponseWriter, request *http.Request) {
		msg := PublishWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::publish::jsondecoding", err)
			sendError(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if msg.Username != Config.AuthClientId {
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
			_, deviceId, localDeviceId, serviceId, localServiceId, err := ParseTopic(Config.SensorTopicPattern, msg.Topic)
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::ParseTopic", err)
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			if deviceId != "" {
				err = connector.HandleDeviceEventWithAuthToken(token, deviceId, serviceId, map[string]string{
					"payload": string(payload),
				})
			}else if localDeviceId != "" {
				err = connector.HandleDeviceRefEventWithAuthToken(token, localDeviceId, localServiceId, map[string]string{
					"payload": string(payload),
				})
			}else {
				err = errors.New("unable to identify device from topic")
			}
			if err != nil {
				log.Println("ERROR: AuthWebhooks::publish::HandleEventWithAuthToken", err, deviceId, serviceId,localDeviceId, localServiceId)
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
		}
		fmt.Fprintf(writer, `{"result": "ok"}`)
	})

	http.HandleFunc("/subscribe", func(writer http.ResponseWriter, request *http.Request) {
		//{"username":"sepl","mountpoint":"","client_id":"sepl_mqtt_connector_1","topics":[{"topic":"$share/sepl_mqtt_connector/#","qos":2}]}
		msg := SubscribeWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::subscribe::jsondecoding", err)
			sendError(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if msg.Username != Config.AuthClientId {
			token, err := connector.Security().GetCachedUserToken(msg.Username)
			if err != nil {
				sendError(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			for _, topic := range msg.Topics {
				_, deviceId, localDeviceId, _, _, err := ParseTopic(Config.ActuatorTopicPattern, topic.Topic)
				if err != nil {
					log.Println("ERROR: AuthWebhooks::subscribe::ParseTopic", err)
					sendError(writer, err.Error(), http.StatusUnauthorized)
					return
				}
				if deviceId != "" {
					_, err = connector.IotCache.WithToken(token).GetDevice(deviceId)
				}else if localDeviceId != "" {
					_, err = connector.IotCache.WithToken(token).GetDeviceByLocalId(localDeviceId)
				}else {
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

	http.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		//{"peer_addr":"172.20.0.30","peer_port":41310,"mountpoint":"","client_id":"sepl_mqtt_connector_1","username":"sepl","password":"sepl","clean_session":true}
		msg := LoginWebhookMsg{}
		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			log.Println("ERROR: AuthWebhooks::login::jsondecoding", err)
			sendError(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if msg.Username != Config.AuthClientId {
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

	log.Fatal(http.ListenAndServe(":"+Config.WebhookPort, http.DefaultServeMux))
}
