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
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"log"
	"log/slog"
	"net/http"
)

func login(writer http.ResponseWriter, request *http.Request, config configuration.Config, connector *platform_connector_lib.Connector, connectionLog connectionlog.ConnectionLog, logger *slog.Logger) {
	//{"peer_addr":"172.20.0.30","peer_port":41310,"mountpoint":"","client_id":"sepl_mqtt_connector_1","username":"sepl","password":"sepl","clean_session":true}
	msg := LoginWebhookMsg{}
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		log.Println("ERROR: InitWebhooks::login::jsondecoding", err)
		sendError(writer, err.Error(), config.Debug)
		return
	}

	authenticationMethod := config.MqttAuthMethod

	if authenticationMethod == "certificate" {
		logger.Info("login", "action", "login", "peerAddr", msg.PeerAddr, "loginType", "cert", "clientId", msg.ClientId, "cleanStart", msg.CleanStart, "cleanSession", msg.CleanSession)
	} else {
		logger.Info("login", "action", "login", "peerAddr", msg.PeerAddr, "loginType", "pw", "username", msg.Username, "clientId", msg.ClientId, "cleanStart", msg.CleanStart, "cleanSession", msg.CleanSession)
	}

	if msg.Username != config.AuthClientId {
		var token security.JwtToken
		var err error

		if authenticationMethod == "password" {
			token, err = connector.Security().GetUserToken(msg.Username, msg.Password)
		} else if authenticationMethod == "certificate" {
			// The user is already authenticated by the TLS client certificate validation in the broker
			token, err = connector.Security().ExchangeUserToken(msg.Username)
		}
		if err != nil {
			log.Println("ERROR: InitWebhooks::login::GetOpenidPasswordToken", err, msg)
			sendError(writer, err.Error(), config.Debug)
			return
		}
		if token == "" {
			sendError(writer, "access denied", config.Debug)
			return
		}
		connectionLog.SetCleanSession(msg.ClientId, msg.CleanSession || msg.CleanStart)
	}
	fmt.Fprintf(writer, `{"result": "ok"}`)
}
