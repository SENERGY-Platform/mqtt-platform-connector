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
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/SENERGY-Platform/platform-connector-lib/statistics"
	"io"
	"log"
	"net/http"
)

// publish godoc
// @Summary      publish webhook
// @Description  checks auth for the published message and forwards it to kafka; all responses are with code=200, differences in swagger doc are because of technical incompatibilities of the documentation format
// @Accept       json
// @Produce      json
// @Param        message body PublishWebhookMsg true "publish message"
// @Success      200 {object}  OkResponse
// @Success      201 {object}  RedirectResponse
// @Failure      400 {object}  ErrorResponse
// @Router       /publish [POST]
func publish(writer http.ResponseWriter, request *http.Request, config configuration.Config, connector *platform_connector_lib.Connector, topicParser *topic.Topic) {
	buf, err := io.ReadAll(request.Body)
	if err != nil {
		sendError(writer, err.Error(), true)
		return
	}
	msgSize := float64(len(buf))
	msg := PublishWebhookMsg{}
	err = json.Unmarshal(buf, &msg)
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
		statistics.SourceReceive(msgSize, msg.Username)
		token, err := connector.Security().GetCachedUserToken(msg.Username, model.RemoteInfo{})
		if err != nil {
			log.Println("ERROR: InitWebhooks::publish::GetUserToken", err)
			sendError(writer, err.Error(), config.Debug)
			return
		}

		device, service, err := topicParser.Parse(token, msg.Topic)
		if errors.Is(err, topic.ErrNoDeviceIdCandidateFound) || errors.Is(err, topic.ErrNoDeviceMatchFound) || errors.Is(err, topic.ErrNoServiceMatchFound) {
			sendIgnoreRedirect(writer, msg.Topic, msg.Payload)
			return
		}
		if err != nil {
			log.Println("ERROR: InitWebhooks::publish::Parse", err)
			sendError(writer, err.Error(), config.Debug)
			return
		}
		var info platform_connector_lib.HandledDeviceInfo
		info, err = connector.HandleDeviceIdentEventWithAuthToken(token, device.Id, device.LocalId, service.Id, service.LocalId, map[string]string{
			"payload": string(payload),
		}, platform_connector_lib.Qos(msg.Qos))
		if info.DeviceId != "" && info.DeviceTypeId != "" {
			statistics.DeviceMsgReceive(msgSize, msg.Username, info.DeviceId, info.DeviceTypeId, info.ServiceIds)
		}
		if err != nil {
			log.Println("WARNING: InitWebhooks::publish::HandleDeviceIdentEventWithAuthToken", err, "\n", device.Id, device.LocalId, service.Id, service.LocalId, msg.Topic)
			fmt.Fprintf(writer, `{"result": "ok"}`)
			return
		}
		statistics.SourceReceiveHandled(msgSize, msg.Username)
		statistics.DeviceMsgHandled(msgSize, msg.Username, info.DeviceId, info.DeviceTypeId, info.ServiceIds)
	}
	fmt.Fprintf(writer, `{"result": "ok"}`)
}
