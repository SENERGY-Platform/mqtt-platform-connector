/*
 * Copyright 2019 InfAI (CC SES)
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

package test

import (
	"encoding/json"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"log"
	"testing"
	"time"
)

func testEvent(t *testing.T, connector *platform_connector_lib.Connector) {
	typeId, serviceTopic, err := server.CreateDeviceType(connector.Config, "payload")
	if err != nil {
		t.Fatal(err)
	}

	device := model.Device{}
	err = server.AdminJwt.PostJSON(connector.Config.DeviceManagerUrl+"/devices", model.Device{LocalId: "foo", DeviceTypeId: typeId, Name: "foo"}, &device)
	if err != nil {
		t.Fatal(err)
	}

	type EventTestType struct {
		DeviceId  string                            `json:"device_id"`
		ServiceId string                            `json:"service_id"`
		Value     map[string]map[string]interface{} `json:"value"`
	}
	envelope := EventTestType{}
	//kafka consumer to ensure no timouts on webhook because topics had to be created
	eventConsumer, err := kafka.NewConsumer(connector.Config.ZookeeperUrl, "test_client", serviceTopic, func(topic string, msg []byte, t time.Time) error {
		log.Println("DEBUG: eventconsumer", string(msg))
		return json.Unmarshal(msg, &envelope)
	}, func(err error, consumer *kafka.Consumer) {
		t.Error(err)
	})
	defer eventConsumer.Stop()

	time.Sleep(10 * time.Second)

	token := mqtt.Publish("event/foo/sepl_get", 0, false, `{"level": 42}`)
	if token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	time.Sleep(20 * time.Second)
	if envelope.DeviceId != device.Id {
		t.Fatal(device.Id, envelope)
	}
	if envelope.Value["metrics"]["level"].(float64) != float64(42) {
		t.Fatal(envelope.Value)
	}
}
