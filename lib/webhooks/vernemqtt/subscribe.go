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
	"errors"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/connectionlog"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"log"
	"net/http"
)

// subscribe godoc
// @Summary      subscribe webhook
// @Description  checks auth for the subscription; SubscriptionResponse.Topics.Qos==128 signals rejected subscription; all responses are with code=200, differences in swagger doc are because of technical incompatibilities of the documentation format
// @Accept       json
// @Produce      json
// @Param        message body SubscribeWebhookMsg true "subscription message"
// @Success      200 {object}  OkResponse
// @Success      201 {object}  SubscribeWebhookResult
// @Failure      400 {object}  ErrorResponse
// @Router       /subscribe [POST]
func subscribe(writer http.ResponseWriter, request *http.Request, config configuration.Config, connector *platform_connector_lib.Connector, topicParser *topic.Topic, connectionLog connectionlog.ConnectionLog) {
	//{"username":"sepl","mountpoint":"","client_id":"sepl_mqtt_connector_1","topics":[{"topic":"$share/sepl_mqtt_connector/#","qos":2}]}
	msg := SubscribeWebhookMsg{}
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		log.Println("ERROR: InitWebhooks::subscribe::jsondecoding", err)
		sendError(writer, err.Error(), config.Debug)
		return
	}
	resultTopics := []WebhookmsgTopic{}
	if msg.Username != config.AuthClientId {
		token, err := connector.Security().GetCachedUserToken(msg.Username, model.RemoteInfo{})
		if err != nil {
			sendError(writer, err.Error(), config.Debug)
			return
		}
		for _, mqtttopic := range msg.Topics {
			device, _, err := topicParser.Parse(token, mqtttopic.Topic)
			if errors.Is(err, topic.ErrNoServiceMatchFound) {
				//we want to only check device access
				err = nil
			}
			if err == nil {
				connectionLog.Subscribe(msg.ClientId, mqtttopic.Topic, device.Id)
			}
			if errors.Is(err, topic.ErrMultipleMatchingDevicesFound) || errors.Is(err, topic.ErrNoDeviceMatchFound) {
				//no err but disallow subscription
				mqtttopic.Qos = 128
				err = nil
			}
			if err != nil && !errors.Is(err, topic.ErrNoDeviceIdCandidateFound) {
				log.Println("WARNING: InitWebhooks::subscribe::ParseTopic", err, mqtttopic.Topic)
				sendError(writer, err.Error(), config.Debug)
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
		log.Println("ERROR: InitWebhooks::subscribe::SubscribeWebhookResult", err)
	}
}
