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

type PublishWebhookMsg struct {
	Username string `json:"username"`
	ClientId string `json:"client_id"`
	Topic    string `json:"topic"`
	Payload  string `json:"payload"`
	Qos      int    `json:"qos"`
}

type WebhookmsgTopic struct {
	Topic string `json:"topic"`
	Qos   int64  `json:"qos"`
}

type SubscribeWebhookMsg struct {
	Username string            `json:"username"`
	ClientId string            `json:"client_id"`
	Topics   []WebhookmsgTopic `json:"topics"`
}

type UnsubscribeWebhookMsg struct {
	Username string   `json:"username"`
	ClientId string   `json:"client_id"`
	Topics   []string `json:"topics"`
}

type LoginWebhookMsg struct {
	PeerAddr     string `json:"peer_addr"`
	PeerPort     int    `json:"peer_port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	ClientId     string `json:"client_id"`
	CleanSession bool   `json:"clean_session"` //v4
	CleanStart   bool   `json:"clean_start"`   //v5
}

type OnlineWebhookMsg struct {
	ClientId string `json:"client_id"`
}

type DisconnectWebhookMsg struct {
	ClientId string `json:"client_id"`
}

type SubscribeWebhookResult struct {
	Result string            `json:"result"`
	Topics []WebhookmsgTopic `json:"topics"`
}

type OkResponse struct {
	Result string `json:"result" example:"ok" default:"ok"`
}

type ErrorResponse struct {
	Result ErrorResponseResult `json:"result"`
}

type ErrorResponseResult struct {
	Error string `json:"error"`
}

type EmptyResponse struct{}

type RedirectResponse struct {
	Result    string            `json:"result" example:"ok" default:"ok"`
	Modifiers RedirectModifiers `json:"modifiers"`
}

type RedirectModifiers struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload" example:"base-64-encoded-payload"`
	Retain  bool   `json:"retain"`
	Qos     int    `json:"qos"`
}

type UnsubResponse struct {
	Result string   `json:"result" example:"ok" default:"ok"`
	Topics []string `json:"topics"`
}
