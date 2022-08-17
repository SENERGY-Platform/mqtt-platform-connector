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

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/helper"
	iotmock "github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/iot"
	"github.com/SENERGY-Platform/platform-connector-lib/iot"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"net/http"
	"net/url"
	"testing"
	"time"
)

const unknownShortDeviceIdExample = "a9B7ddfMShqI26yT9hqnsu"
const unknownLongDeviceIdExample = "urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7bu"

func TestParse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deviceManagerUrl, deviceRepoUrl, err := iotmock.MockWithoutKafka(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	iotRepo := iot.New(deviceManagerUrl, deviceRepoUrl, "")
	iotCache := iot.NewCache(iotRepo, 60, 60, 60, 2, 200*time.Millisecond)

	topics := topic.New(iotCache, "")

	t.Run("create device type", testCreateDeviceType(deviceManagerUrl, model.DeviceType{
		Id:   "dt1",
		Name: "dt1",
		Services: []model.Service{
			{
				LocalId: "void/poweron",
			},
			{
				LocalId: "poweron",
			},
			{
				LocalId: "tele/SENSOR",
			},
		},
	}))

	t.Run("create device", testCreateType(deviceManagerUrl, model.Device{
		Id:           longDeviceIdExample,
		DeviceTypeId: "dt1",
	}))

	t.Run("create device 2", testCreateType(deviceManagerUrl, model.Device{
		Id:           longDeviceIdExample2,
		DeviceTypeId: "dt1",
	}))

	t.Run(testTopicParse(topics, shortDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, shortDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, "void/poweron/"+shortDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, "poweron/"+shortDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, "cmd/"+shortDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, "cmd/"+shortDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, longDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, longDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, "void/poweron/"+longDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, "poweron/"+longDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, "cmd/"+longDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, "cmd/"+longDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, shortDeviceIdExample+"/void/poweron/"+unknownShortDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, shortDeviceIdExample+"/poweron/"+unknownShortDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, unknownLongDeviceIdExample+"/void/poweron/"+shortDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, unknownShortDeviceIdExample+"poweron/"+shortDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, "cmd/"+shortDeviceIdExample+"/"+unknownShortDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, "cmd/"+shortDeviceIdExample+"/"+unknownShortDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, longDeviceIdExample+"/void/poweron/"+unknownLongDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, longDeviceIdExample+"/poweron/"+unknownLongDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, unknownLongDeviceIdExample+"/void/poweron/"+longDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, unknownLongDeviceIdExample+"poweron/"+longDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topics, "cmd/"+longDeviceIdExample+"/"+unknownLongDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topics, "cmd/"+longDeviceIdExample+"/"+unknownLongDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParserExpectError(topics, "cmd/foo/bar", topic.ErrNoDeviceIdCandidateFound))
	t.Run(testTopicParserExpectError(topics, "cmd/"+unknownLongDeviceIdExample+"/bar", topic.ErrNoDeviceMatchFound))
	t.Run(testTopicParserExpectError(topics, unknownLongDeviceIdExample+"/cmd/bar", topic.ErrNoDeviceMatchFound))
	t.Run(testTopicParserExpectError(topics, "cmd/bar/"+unknownLongDeviceIdExample, topic.ErrNoDeviceMatchFound))

	t.Run(testTopicParserExpectError(topics, "cmd/bar/"+longDeviceIdExample2+"/"+longDeviceIdExample, topic.ErrMultipleMatchingDevicesFound))
	t.Run(testTopicParserExpectError(topics, longDeviceIdExample+"/cmd/bar/"+longDeviceIdExample2, topic.ErrMultipleMatchingDevicesFound))

	t.Run(testTopicParserExpectError(topics, longDeviceIdExample+"/cmd/bar", topic.ErrNoServiceMatchFound))
	t.Run(testTopicParserExpectError(topics, "cmd/bar/"+longDeviceIdExample, topic.ErrNoServiceMatchFound))

}

func testCreateType(deviceManagerUrl string, device model.Device) func(t *testing.T) {
	return func(t *testing.T) {
		b := new(bytes.Buffer)
		err := json.NewEncoder(b).Encode(device)
		if err != nil {
			return
		}

		client := http.Client{
			Timeout: 5 * time.Second,
		}
		req, err := http.NewRequest(
			"PUT",
			deviceManagerUrl+"/devices/"+url.PathEscape(device.Id),
			b,
		)
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", string(helper.AdminJwt))

		resp, err := client.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Error(resp.Status)
			return
		}
	}
}

func testCreateDeviceType(deviceManagerUrl string, deviceType model.DeviceType) func(t *testing.T) {
	return func(t *testing.T) {
		b := new(bytes.Buffer)
		err := json.NewEncoder(b).Encode(deviceType)
		if err != nil {
			return
		}

		client := http.Client{
			Timeout: 5 * time.Second,
		}
		req, err := http.NewRequest(
			"PUT",
			deviceManagerUrl+"/device-types/"+url.PathEscape(deviceType.Id),
			b,
		)
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", string(helper.AdminJwt))

		resp, err := client.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Error(resp.Status)
			return
		}
	}
}

func testTopicParse(parser *topic.Topic, topic string, expectedDeviceId string, expectedLocalServiceId string) (string, func(t *testing.T)) {
	return topic, func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("recover", r)
			}
		}()
		device, service, err := parser.Parse(helper.AdminJwt, topic)
		if err != nil {
			t.Error(err)
			return
		}
		if device.Id != expectedDeviceId {
			t.Error(device.Id, expectedDeviceId)
			return
		}
		if service.LocalId != expectedLocalServiceId {
			t.Error(service.LocalId, expectedLocalServiceId)
			return
		}
	}
}

func testTopicParserExpectError(parser *topic.Topic, topic string, expectedError error) (string, func(t *testing.T)) {
	return topic, func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("recover", r)
			}
		}()
		deviceId, localServiceId, err := parser.Parse(helper.AdminJwt, topic)
		if err != expectedError {
			t.Error(err, expectedError, deviceId, localServiceId)
			return
		}
	}
}
