/*
 * Copyright 2025 InfAI (CC SES)
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
	"context"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/client"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/auth"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

func TestSNRGY4089(t *testing.T) {
	defaultConfig, err := configuration.Load("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.InitTopics = true
	defaultConfig.PublishToPostgres = true
	defaultConfig.MqttAuthMethod = "password"
	defaultConfig.MqttVersion = "4"

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, clientBroker, err := server.New(ctx, wg, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	err = lib.Start(ctx, config)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(1 * time.Second)

	deviceLocalId := "testservice1"
	deviceId := "urn:infai:ses:device:00dbdd68-7a57-41fc-a959-1f04892b5c5e"
	serviceLocalId := "testservice1"
	serviceId := "urn:infai:ses:service:d2ada448-9e3f-408a-ab5c-b3040ab99470"
	deviceType := model.DeviceType{}
	protocol := model.Protocol{}
	device := model.Device{}

	t.Run("create protocol", func(t *testing.T) {
		protocol = createTestProtocol(t, config)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceTypeWithTextPayload(t, config, protocol, serviceLocalId, serviceId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device", func(t *testing.T) {
		device = createTestDeviceWithUserToken(t, auth.UserToken, config, deviceType, deviceLocalId, deviceId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("check device/# subscription", func(t *testing.T) {
		mqtt, err := client.New(clientBroker, "user", "user", uuid.NewString(), "password", client.MQTT4, true, true)
		if err != nil {
			t.Error(err)
			return
		}
		defer mqtt.Stop()
		received := false
		err = mqtt.Subscribe(device.Id+"/#", 2, func(topic string, payload []byte) {
			received = true
		})
		if err != nil {
			t.Error(err)
			return
		}

		adminClient, err := client.New(clientBroker, config.AuthClientId, config.AuthClientSecret, uuid.NewString(), "password", client.MQTT4, true, true)
		if err != nil {
			t.Error(err)
			return
		}
		defer adminClient.Stop()
		err = adminClient.Publish(device.Id+"/bar", "foobar", 2)
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(2 * time.Second)
		if !received {
			t.Error("message should have been received")
		}
	})

	t.Run("check # subscription", func(t *testing.T) {
		mqtt, err := client.New(clientBroker, "user", "user", uuid.NewString(), "password", client.MQTT4, true, true)
		if err != nil {
			t.Error(err)
			return
		}
		defer mqtt.Stop()
		err = mqtt.Subscribe("#", 2, func(topic string, payload []byte) {
			t.Error("access should have been denied")
		})
		if err != nil {
			t.Error(err)
			return
		}

		adminClient, err := client.New(clientBroker, config.AuthClientId, config.AuthClientSecret, uuid.NewString(), "password", client.MQTT4, true, true)
		if err != nil {
			t.Error(err)
			return
		}
		defer adminClient.Stop()
		err = adminClient.Publish("foo/bar", "foobar", 2)
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(2 * time.Second)
	})

}
