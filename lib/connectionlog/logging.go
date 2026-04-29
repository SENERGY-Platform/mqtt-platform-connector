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

package connectionlog

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	connection_check_lib "github.com/SENERGY-Platform/connection-check-v2/lib"
	"github.com/SENERGY-Platform/platform-connector-lib/connectionlog"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	_ "github.com/lib/pq"
)

type ConnectionLog interface {
	Connect(client string)
	Disconnect(client string)
	Subscribe(client string, topic string, deviceId string)
	Unsubscribe(client string, topic string, deviceId string)
	SetCleanSession(id string, session bool)
}

func New(producer kafka.ProducerInterface, conStr string, deviceLogTopic string, connCheckUrl string, httpTimeoutStr string) (result ConnectionLog, err error) {
	logger := &ConnectionLogImpl{}
	if connCheckUrl != "" {
		var httpTimeout time.Duration
		httpTimeout, err = time.ParseDuration(httpTimeoutStr)
		if err != nil && httpTimeoutStr != "" {
			slog.Warn("invalid ConnectionCheckHttpTimeout; use default 15s")
			httpTimeout = 15 * time.Second
		}
		logger.logger, err = connectionlog.NewWithProducerAndConnCheck(producer, connection_check_lib.New(&http.Client{Timeout: httpTimeout}, connCheckUrl), deviceLogTopic, "")
	} else {
		logger.logger, err = connectionlog.NewWithProducer(producer, deviceLogTopic, "")
	}
	if err != nil {
		return logger, err
	}
	logger.db, err = initDbConnection(conStr)
	if err != nil {
		return logger, err
	}
	return logger, nil
}

type ConnectionLogImpl struct {
	logger connectionlog.Logger
	db     *sql.DB
}

func (this *ConnectionLogImpl) Subscribe(client string, topic string, deviceId string) {
	err := this.storeSubscription(client, topic, deviceId)
	if err != nil {
		slog.Error("could not store subscription", "client", client, "topic", topic, "device_id", deviceId)
		return
	}
	err = this.logger.LogDeviceConnect(deviceId)
	if err != nil {
		slog.Error("could not log device connect", "client", client, "topic", topic, "device_id", deviceId)
		return
	}
	return
}

func (this *ConnectionLogImpl) Unsubscribe(client string, topic string, deviceId string) {
	err := this.removeSubscription(client, topic)
	if err != nil {
		slog.Error("could not remove subscription", "client", client, "topic", topic, "device_id", deviceId)
		return
	}
	noSub, err := this.noDeviceSubscriptionStored(deviceId)
	if err != nil {
		slog.Error("could not check if device has subscriptions", "client", client, "topic", topic, "device_id", deviceId)
		return
	}
	if noSub {
		err = this.logger.LogDeviceDisconnect(deviceId)
		if err != nil {
			slog.Error("could not log device disconnect", "client", client, "topic", topic, "device_id", deviceId)
			return
		}
	}
	return
}

func (this *ConnectionLogImpl) Disconnect(client string) {
	cleanSession, err := this.isCleanSession(client)
	if err != nil {
		slog.Error("could not check if client uses clean session", "client", client)
		return
	}
	devices, err := this.loadSubscriptions(client)
	if err != nil {
		slog.Error("could not load subscriptions", "client", client)
		return
	}
	if cleanSession {
		err = this.removeClient(client)
	} else {
		err = this.setClientInactive(client, true)
	}
	if err != nil {
		slog.Error("could not set client inactive", "client", client)
		return
	}
	filtered, err := this.filterByStoredDevices(devices)
	if err != nil {
		slog.Error("could not filter devices", "client", client)
		return
	}
	for _, d := range filtered {
		err = this.logger.LogDeviceDisconnect(d)
		if err != nil {
			slog.Error("could not log device disconnect", "client", client, "device_id", d)
			continue
		}
	}
}

func (this *ConnectionLogImpl) Connect(client string) {
	cleanSession, err := this.isCleanSession(client)
	if err != nil {
		slog.Error("could not check if client uses clean session", "client", client)
		return
	}
	if cleanSession {
		return
	}
	err = this.setClientInactive(client, false)
	if err != nil {
		slog.Error("could not set client inactive", "client", client)
		return
	}
	devices, err := this.loadSubscriptions(client)
	if err != nil {
		slog.Error("could not load subscriptions", "client", client)
		return
	}
	for _, d := range devices {
		err = this.logger.LogDeviceConnect(d)
		if err != nil {
			slog.Error("could not log device connect", "client", client, "device_id", d)
			continue
		}
	}
}

func (this *ConnectionLogImpl) SetCleanSession(client string, clean bool) {
	err := this.setCleanSession(client, clean)
	if err != nil {
		slog.Error("could not set clean session", "client", client, "clean", clean)
		return
	}
}
