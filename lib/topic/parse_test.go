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

package topic

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
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
	conf := lib.Config{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := iotmock.Mock(&conf, ctx)
	if err != nil {
		t.Error(err)
		return
	}
	iotRepo := iot.New(conf.DeviceManagerUrl, conf.DeviceRepoUrl, "")
	iotCache := iot.NewCache(iotRepo, 60, 60, 60)

	topic := New(iotCache, "")

	t.Run("create device type", testCreateDeviceType(conf, model.DeviceType{
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

	t.Run("create device", testCreateType(conf, model.Device{
		Id:           longDeviceIdExample,
		DeviceTypeId: "dt1",
	}))

	t.Run("create device 2", testCreateType(conf, model.Device{
		Id:           longDeviceIdExample2,
		DeviceTypeId: "dt1",
	}))

	t.Run(testTopicParse(topic, shortDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, shortDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, "void/poweron/"+shortDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, "poweron/"+shortDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, "cmd/"+shortDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, "cmd/"+shortDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, longDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, longDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, "void/poweron/"+longDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, "poweron/"+longDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, "cmd/"+longDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, "cmd/"+longDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, shortDeviceIdExample+"/void/poweron/"+unknownShortDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, shortDeviceIdExample+"/poweron/"+unknownShortDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, unknownLongDeviceIdExample+"/void/poweron/"+shortDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, unknownShortDeviceIdExample+"poweron/"+shortDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, "cmd/"+shortDeviceIdExample+"/"+unknownShortDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, "cmd/"+shortDeviceIdExample+"/"+unknownShortDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, longDeviceIdExample+"/void/poweron/"+unknownLongDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, longDeviceIdExample+"/poweron/"+unknownLongDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, unknownLongDeviceIdExample+"/void/poweron/"+longDeviceIdExample, longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, unknownLongDeviceIdExample+"poweron/"+longDeviceIdExample, longDeviceIdExample, "poweron"))

	t.Run(testTopicParse(topic, "cmd/"+longDeviceIdExample+"/"+unknownLongDeviceIdExample+"/void/poweron", longDeviceIdExample, "void/poweron"))
	t.Run(testTopicParse(topic, "cmd/"+longDeviceIdExample+"/"+unknownLongDeviceIdExample+"/poweron", longDeviceIdExample, "poweron"))

	t.Run(testTopicParserExpectError(topic, "cmd/foo/bar", ErrNoDeviceMatchFound))
	t.Run(testTopicParserExpectError(topic, "cmd/"+unknownLongDeviceIdExample+"/bar", ErrNoDeviceMatchFound))
	t.Run(testTopicParserExpectError(topic, unknownLongDeviceIdExample+"/cmd/bar", ErrNoDeviceMatchFound))
	t.Run(testTopicParserExpectError(topic, "cmd/bar/"+unknownLongDeviceIdExample, ErrNoDeviceMatchFound))

	t.Run(testTopicParserExpectError(topic, "cmd/bar/"+longDeviceIdExample2+"/"+longDeviceIdExample, ErrMultipleMatchingDevicesFound))
	t.Run(testTopicParserExpectError(topic, longDeviceIdExample+"/cmd/bar/"+longDeviceIdExample2, ErrMultipleMatchingDevicesFound))

}

func testCreateType(conf lib.Config, device model.Device) func(t *testing.T) {
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
			conf.DeviceManagerUrl+"/devices/"+url.PathEscape(device.Id),
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

func testCreateDeviceType(conf lib.Config, deviceType model.DeviceType) func(t *testing.T) {
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
			conf.DeviceManagerUrl+"/device-types/"+url.PathEscape(deviceType.Id),
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

func testTopicParse(parser *Topic, topic string, expectedDeviceId string, expectedLocalServiceId string) (string, func(t *testing.T)) {
	return topic, func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("recover", r)
			}
		}()
		deviceId, localServiceId, err := parser.Parse(helper.AdminJwt, topic)
		if err != nil {
			t.Error(err)
			return
		}
		if deviceId != expectedDeviceId {
			t.Error(deviceId, expectedDeviceId)
			return
		}
		if localServiceId != expectedLocalServiceId {
			t.Error(localServiceId, expectedLocalServiceId)
			return
		}
	}
}

func testTopicParserExpectError(parser *Topic, topic string, expectedError error) (string, func(t *testing.T)) {
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
