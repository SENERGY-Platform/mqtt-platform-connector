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
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SmartEnergyPlatform/iot-device-repository/lib/model"
	"strings"
	"testing"
	"time"
)

func TestPublish(t *testing.T) {
	token := mqtt.Publish("foo/bar", 0, false, `{"foo1": "bar"}`)
	if token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}
	time.Sleep(20 * time.Second)
	endpoints, err := connector.Iot().GetInEndpoints(usertoken, "foo/bar")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 {
		ds, err := connector.Iot().GetDevices(usertoken, 100, 0)
		t.Log(ds, err)
		dt, _ := connector.Iot().GetDeviceType(ds[0].DeviceType, usertoken)
		b, _ := json.Marshal(dt)
		t.Log(string(b))
		ep, _ := connector.Iot().GetOutEndpoint(usertoken, ds[0].Id, dt.Services[0].Id)
		b, _ = json.Marshal(ep)
		t.Log(string(b))
		t.Fatal(endpoints)
	}
	endpoint := endpoints[0]
	device, err := connector.Iot().GetDevice(endpoint.Device, usertoken)
	if err != nil {
		t.Fatal(err)
	}
	if device.Url != "foo/bar" {
		t.Fatal(device)
	}
	dt, err := connector.Iot().GetDeviceType(device.DeviceType, usertoken)
	if err != nil {
		t.Fatal(err)
	}
	if len(dt.Services) != 1 {
		t.Fatal(dt.Services)
	}
	service := dt.Services[0]
	if service.EndpointFormat != "{{device_uri}}" {
		t.Fatal(service)
	}
	if len(service.Output) != 1 {
		t.Fatal(service.Output)
	}
	output := service.Output[0]
	if output.Type.BaseType != model.StructBaseType {
		t.Fatal(output.Type.BaseType)
	}
	if len(output.Type.Fields) != 1 {
		t.Fatal(output.Type.Fields)
	}
	field := output.Type.Fields[0]
	if field.Name != "foo1" {
		t.Fatal(field)
	}
	if field.Type.BaseType != model.XsdString {
		t.Fatal(field.Type.BaseType)
	}

	type EventTestType struct {
		DeviceId  string                            `json:"device_id"`
		ServiceId string                            `json:"service_id"`
		Value     map[string]map[string]interface{} `json:"value"`
	}
	envelope := EventTestType{}
	//kafka consumer to ensure no timouts on webhook because topics had to be created
	eventConsumer, err := kafka.NewConsumer(config.ZookeeperUrl, "test_client", strings.Replace(endpoints[0].Service, "#", "_", 1), func(topic string, msg []byte) error {
		json.Unmarshal(msg, &envelope)
		return nil
	}, func(err error, consumer *kafka.Consumer) {
		t.Error(err)
	})
	defer eventConsumer.Stop()

	token = mqtt.Publish("foo/bar", 0, false, `{"foo1": "bar"}`)
	if token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}
	time.Sleep(20 * time.Second)
	if envelope.DeviceId != endpoints[0].Device {
		t.Fatal(envelope)
	}
	if envelope.Value["payload"]["foo1"] != "bar" {
		t.Fatal(envelope.Value)
	}
}
