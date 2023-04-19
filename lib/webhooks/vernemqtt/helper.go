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
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"log"
	"net/http"
)

func sendError(writer http.ResponseWriter, msg string, logging bool) {
	if logging {
		log.Println("DEBUG: send error:", msg)
	}
	err := json.NewEncoder(writer).Encode(map[string]map[string]string{"result": {"error": msg}})
	if err != nil {
		log.Println("ERROR: unable to send error msg:", err, msg)
	}
}

func sendIgnoreRedirect(writer http.ResponseWriter, topic string, msg string) {
	log.Println("WARNING: send ignore redirect:", topic, msg)
	err := json.NewEncoder(writer).Encode(map[string]interface{}{
		"result": "ok",
		"modifiers": map[string]interface{}{
			"topic":   "ignored/" + topic,
			"payload": base64.StdEncoding.EncodeToString([]byte(msg)),
			"retain":  false,
			"qos":     0,
		}})
	if err != nil {
		log.Println("ERROR: unable to send ignore redirect:", err, msg)
	}
}

func sendIgnoreRedirectAndNotification(writer http.ResponseWriter, connector *platform_connector_lib.Connector, user, clientId, topic, msg string) {
	sendIgnoreRedirect(writer, topic, msg)
	userId, err := connector.Security().GetUserId(user)
	if err != nil {
		log.Println("ERROR: unable to get user id", err)
		return
	}
	connector.HandleClientError(userId, clientId, "ignore message to "+topic+": "+msg)
}

func sendSubscriptionResult(writer http.ResponseWriter, ok []WebhookmsgTopic, rejected []WebhookmsgTopic) {
	topics := []interface{}{}
	for _, topic := range ok {
		topics = append(topics, topic)
	}
	for _, topic := range rejected {
		topics = append(topics, map[string]interface{}{
			"topic": topic.Topic,
			"qos":   128,
		})
	}
	msg := map[string]interface{}{
		"result": "ok",
		"topics": topics,
	}
	err := json.NewEncoder(writer).Encode(msg)
	if err != nil {
		log.Println("ERROR: unable to send sendSubscriptionResult msg:", err)
	}
}
