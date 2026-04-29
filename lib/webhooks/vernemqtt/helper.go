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
	"log/slog"
	"net/http"

	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
)

func sendError(writer http.ResponseWriter, msg string, logging bool) {
	if logging {
		slog.Debug("send error", "msg", msg)
	}
	err := json.NewEncoder(writer).Encode(ErrorResponse{Result: ErrorResponseResult{Error: msg}})
	if err != nil {
		slog.Error("unable to send error msg", "error", err, "msg", msg)
	}
}

func sendIgnoreRedirect(writer http.ResponseWriter, topic string, base64Msg string) {
	slog.Debug("send ignore redirect", "topic", topic, "msg", base64Msg)
	err := json.NewEncoder(writer).Encode(RedirectResponse{
		Result: "ok",
		Modifiers: RedirectModifiers{
			Topic:   "ignored/" + topic,
			Payload: base64Msg,
			Retain:  false,
			Qos:     0,
		},
	})
	if err != nil {
		slog.Error("unable to send ignore redirect", "error", err, "msg", base64Msg)
	}
}

func sendIgnoreRedirectAndNotification(writer http.ResponseWriter, connector *platform_connector_lib.Connector, user, clientId, topic, base64Msg string) {
	sendIgnoreRedirect(writer, topic, base64Msg)
	userId, err := connector.Security().GetUserId(user)
	if err != nil {
		slog.Error("unable to get user id", "error", err, "user", user)
		return
	}
	connector.HandleClientError(userId, clientId, "ignore message to "+topic+": "+base64Msg)
}

func sendSubscriptionResult(writer http.ResponseWriter, ok []WebhookmsgTopic, rejected []WebhookmsgTopic) {
	topics := []WebhookmsgTopic{}
	for _, topic := range ok {
		topics = append(topics, topic)
	}
	for _, topic := range rejected {
		topics = append(topics, WebhookmsgTopic{
			Topic: topic.Topic,
			Qos:   128,
		})
	}
	err := json.NewEncoder(writer).Encode(SubscribeWebhookResult{
		Result: "ok",
		Topics: topics,
	})
	if err != nil {
		slog.Error("unable to send sendSubscriptionResult msg", "error", err)
	}
}
