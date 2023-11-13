/*
 * Copyright 2020 InfAI (CC SES)
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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/statistics"
	"log"
	"net/http"
)

func publish(writer http.ResponseWriter, request *http.Request, config configuration.Config, connector *platform_connector_lib.Connector, topicParser *topic.Topic) {
	msg := PublishWebhookMsg{}
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		log.Println("ERROR: InitWebhooks::publish::jsondecoding", err)
		sendError(writer, err.Error(), config.Debug)
		return
	}
	if msg.Username != config.AuthClientId {
		payload, err := base64.StdEncoding.DecodeString(msg.Payload)
		if err != nil {
			log.Println("ERROR: InitWebhooks::publish::base64decoding", err)
			sendError(writer, err.Error(), config.Debug)
			return
		}
		statistics.SourceReceive(float64(len(payload)), msg.Username)
		token, err := connector.Security().GetCachedUserToken(msg.Username)
		if err != nil {
			log.Println("ERROR: InitWebhooks::publish::GetUserToken", err)
			sendError(writer, err.Error(), config.Debug)
			return
		}

		device, service, err := topicParser.Parse(token, msg.Topic)
		if err == topic.ErrNoDeviceIdCandidateFound {
			//topics that cant possibly be connected to a device may be handled at will
			if config.Debug {
				log.Println("WARNING: InitWebhooks::publish::ParseTopic", err, msg.Topic)
			}
			fmt.Fprintf(writer, `{"result": "ok"}`)
			return
		}
		if err == topic.ErrNoServiceMatchFound {
			//we want to only check device access
			if config.Debug {
				log.Println("WARNING: InitWebhooks::publish::ParseTopic", err, msg.Topic)
			}
			fmt.Fprintf(writer, `{"result": "ok"}`)
			return
		}
		if err != nil {
			log.Println("ERROR: InitWebhooks::publish::Parse", err)
			sendError(writer, err.Error(), config.Debug)
			return
		}
		err = connector.HandleDeviceIdentEventWithAuthToken(token, device.Id, device.LocalId, service.Id, service.LocalId, map[string]string{
			"payload": string(payload),
		}, platform_connector_lib.Qos(msg.Qos))
		if err != nil {
			log.Println("WARNING: InitWebhooks::publish::HandleDeviceIdentEventWithAuthToken", err, "\n", device.Id, device.LocalId, service.Id, service.LocalId, msg.Topic)
			fmt.Fprintf(writer, `{"result": "ok"}`)
			return
		}
	}
	fmt.Fprintf(writer, `{"result": "ok"}`)
}
