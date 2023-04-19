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
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/connectionlog"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"log"
	"net/http"
	"runtime/debug"
)

func unsubscribe(writer http.ResponseWriter, request *http.Request, config configuration.Config, connector *platform_connector_lib.Connector, topicParser *topic.Topic, connectionLog connectionlog.ConnectionLog) {
	defer func() {
		if p := recover(); p != nil {
			debug.PrintStack()
			sendError(writer, fmt.Sprint(p), config.Debug)
			return
		}
	}()
	msg := UnsubscribeWebhookMsg{}
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		sendError(writer, err.Error(), config.Debug)
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
}
