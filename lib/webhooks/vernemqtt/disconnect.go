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
	"log"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func disconnect(writer http.ResponseWriter, request *http.Request, config configuration.Config, connectionLog connectionlog.ConnectionLog, logger *slog.Logger) {
	defer func() {
		if p := recover(); p != nil {
			if config.Debug {
				debug.PrintStack()
			}
			sendError(writer, fmt.Sprint(p), config.Debug)
			return
		} else {
			fmt.Fprintf(writer, "{}")
		}
	}()
	msg := DisconnectWebhookMsg{}
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		log.Println("ERROR: InitWebhooks::disconnect::jsondecoding", err)
		return
	}
	logger.Info("disconnect", "action", "disconnect", "clientId", msg.ClientId)
	connectionLog.Disconnect(msg.ClientId)
}
