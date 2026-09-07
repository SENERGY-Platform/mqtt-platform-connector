/*
 * Copyright 2026 InfAI (CC SES)
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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/client"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/auth"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/google/uuid"
)

// A publish whose device resolves but whose service does not used to leave the
// response empty, which the broker reads as invalid JSON and answers by dropping
// the connection. It has to be answered with an ignore redirect instead.
func TestSNRGY4673(t *testing.T) {
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

	deviceLocalId := "snrgy4673"
	deviceId := "urn:infai:ses:device:8d1f2a3b-4c5d-6e7f-8091-a2b3c4d5e6f7"
	serviceLocalId := "snrgy4673"
	serviceId := "urn:infai:ses:service:9e2f3a4b-5c6d-7e8f-9012-b3c4d5e6f708"
	deviceType := model.DeviceType{}
	protocol := model.Protocol{}
	device := model.Device{}

	t.Run("create protocol", func(t *testing.T) {
		protocol = createTestProtocol(t, config)
	})

	// carries no senergy/mqtt-generate-services attribute, so the connector
	// will not generate a service for an unknown topic part
	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceTypeWithTextPayload(t, config, protocol, serviceLocalId, serviceId)
	})

	t.Run("create device", func(t *testing.T) {
		device = createTestDeviceWithUserToken(t, auth.UserToken, config, deviceType, deviceLocalId, deviceId)
	})

	// <deviceId>/<part> resolves the device and takes the rest as the service
	// local id (see TestParse), so an unused part reaches ErrNoServiceMatchFound
	// rather than the unmatched-device branch next to it.
	t.Run("unmatched service is answered with an ignore redirect", func(t *testing.T) {
		adminClient, err := client.New(clientBroker, config.AuthClientId, config.AuthClientSecret, uuid.NewString(), "password", client.MQTT4, true, true)
		if err != nil {
			t.Error(err)
			return
		}
		defer adminClient.Stop()

		//the subscription callback runs in the paho client goroutine, so it only
		//records that a message arrived and the test goroutine asserts it
		ignored := &atomic.Bool{}
		err = adminClient.Subscribe("ignored/#", 2, func(topic string, payload []byte) {
			ignored.Store(true)
		})
		if err != nil {
			t.Error(err)
			return
		}

		deviceClient, err := client.New(clientBroker, "user", "user", uuid.NewString(), "password", client.MQTT4, true, true)
		if err != nil {
			t.Error(err)
			return
		}
		defer deviceClient.Stop()

		err = deviceClient.Publish(device.Id+"/error", "unroutable", 2)
		if err != nil {
			t.Error(err)
			return
		}

		waitFor(t, 30*time.Second, 0, func() error {
			if !ignored.Load() {
				return errors.New("publish to an unmatched service should have been redirected to ignored/")
			}
			return nil
		})
	})
}
