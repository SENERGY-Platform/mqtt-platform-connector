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
	"errors"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/client"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/auth"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/google/uuid"
	"sync"
	"sync/atomic"
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
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceTypeWithTextPayload(t, config, protocol, serviceLocalId, serviceId)
	})

	t.Run("create device", func(t *testing.T) {
		device = createTestDeviceWithUserToken(t, auth.UserToken, config, deviceType, deviceLocalId, deviceId)
	})

	t.Run("check device/# subscription", func(t *testing.T) {
		mqtt, err := client.New(clientBroker, "user", "user", uuid.NewString(), "password", client.MQTT4, true, true)
		if err != nil {
			t.Error(err)
			return
		}
		defer mqtt.Stop()
		//the subscription callback runs in the paho client goroutine, so it only
		//records that a message arrived and the test goroutine asserts it
		received := &atomic.Bool{}
		err = mqtt.Subscribe(device.Id+"/#", 2, func(topic string, payload []byte) {
			received.Store(true)
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
		waitFor(t, 30*time.Second, 0, func() error {
			if !received.Load() {
				return errors.New("message should have been received")
			}
			return nil
		})
	})

	t.Run("check # subscription", func(t *testing.T) {
		mqtt, err := client.New(clientBroker, "user", "user", uuid.NewString(), "password", client.MQTT4, true, true)
		if err != nil {
			t.Error(err)
			return
		}
		defer mqtt.Stop()
		delivered := &atomic.Bool{}
		err = mqtt.Subscribe("#", 2, func(topic string, payload []byte) {
			delivered.Store(true)
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

		//nothing arriving is the expectation here, so this stays a fixed wait
		time.Sleep(2 * time.Second)
		if delivered.Load() {
			t.Error("access should have been denied")
		}
	})

}
