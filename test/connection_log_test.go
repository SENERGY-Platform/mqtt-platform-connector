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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/client"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/google/uuid"
)

// deviceLogSettle is the grace period the device-log checks require the
// expectation to keep holding. The checks also assert that no further
// connect/disconnect was logged, which a snapshot taken the moment the
// expectation is first met would not cover.
const deviceLogSettle = 2 * time.Second

func TestConnectionLogDevice1Minimal(t *testing.T) {
	t.Skip("collection of tests")
	t.Run("TestConnectionLogDevice1MinimalMqtt4", TestConnectionLogDevice1MinimalMqtt4)
	t.Run("TestConnectionLogDevice1MinimalCertMqtt4", TestConnectionLogDevice1MinimalCertMqtt4)
}

func TestConnectionLogDevice1MinimalMqtt4(t *testing.T) {
	testConnectionLogDevice1Minimal(t, "password", client.MQTT4)
}

func TestConnectionLogDevice1MinimalMqtt5(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	testConnectionLogDevice1Minimal(t, "password", client.MQTT5)
}

func TestConnectionLogDevice1MinimalCertMqtt4(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	testConnectionLogDevice1Minimal(t, "certificate", client.MQTT4)
}

func TestConnectionLogDevice1MinimalCertMqtt5(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	testConnectionLogDevice1Minimal(t, "certificate", client.MQTT5)
}

func testConnectionLogDevice1Minimal(t *testing.T, authMethod string, mqttVersion client.MqttVersion) {
	if mqttVersion == client.MQTT5 {
		t.Skip("clean-start=false is currently not supported by mqtt 5 paho client")
	}
	defaultConfig, err := configuration.Load("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.InitTopics = true

	defaultConfig.MqttAuthMethod = authMethod
	if mqttVersion == client.MQTT5 {
		defaultConfig.MqttVersion = "5"
	}
	defer t.Log("test done")
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, brokerForClients, err := server.NewWithConnectionLog(ctx, wg, defaultConfig)
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
	serviceLocalId := "testservice1"
	serviceId := "urn:infai:ses:service:efed3e07-e738-445f-8a4f-847b87688506"
	deviceType := model.DeviceType{}
	protocol := model.Protocol{}

	device := model.Device{}

	t.Run("create protocol", func(t *testing.T) {
		protocol = createTestProtocol(t, config)
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceType(t, config, protocol, serviceLocalId, serviceId)
	})

	t.Run("create devices", func(t *testing.T) {
		device = createTestDevice(t, config, deviceType, deviceLocalId+"_1", "")
	})

	expected := []DeviceLog{}

	t.Run("run", func(t *testing.T) {
		client1, err := client.New(brokerForClients, "bar", "foo", "client1", authMethod, mqttVersion, false, false)
		if err != nil {
			t.Error(err)
			return
		}
		defer client1.Stop()

		err = client1.Subscribe(device.Id+"/"+serviceLocalId, 2, func(topic string, pl []byte) {})
		if err != nil {
			t.Error(err)
			return
		}
		expected = append(expected, DeviceLog{Id: device.Id, Connected: true})

		err = client1.Subscribe(device.Id+"/"+serviceLocalId+"_2", 2, func(topic string, pl []byte) {})
		if err != nil {
			t.Error(err)
			return
		}
		expected = append(expected, DeviceLog{Id: device.Id, Connected: true})

		time.Sleep(2 * time.Second)

		//disconnect device[1]
		err = client1.Unsubscribe(device.Id+"/"+serviceLocalId, device.Id+"/"+serviceLocalId+"_2")
		if err != nil {
			t.Error(err)
			return
		}
		expected = append(expected, DeviceLog{Id: device.Id, Connected: false})

	})

	t.Run("check", func(t *testing.T) {
		state := &deviceLogState{}
		log.Println("consume", config.DeviceLogTopic)
		err = kafka.NewConsumer(ctx,
			kafka.ConsumerConfig{
				KafkaUrl: config.KafkaUrl,
				Topic:    config.DeviceLogTopic,
				GroupId:  "check_consumer_" + uuid.NewString(),
				MaxWait:  100 * time.Millisecond,
				//nothing has produced to the device-log topic yet, and a consumer
				//group that starts on a topic the broker does not know does not
				//reliably recover once the topic appears
				InitTopic:      true,
				TopicConfigMap: config.KafkaTopicConfigs,
			}, state.consume, state.handleError)
		if err != nil {
			t.Error(err)
			return
		}
		waitFor(t, 60*time.Second, deviceLogSettle, state.matchesInOrder(expected, []model.Device{device}))
	})
}

func TestConnectionLogMqtt4(t *testing.T) {
	testConnectionLog(t, "password", client.MQTT4)
}

func TestConnectionLogMqtt5(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	testConnectionLog(t, "password", client.MQTT5)
}

func TestConnectionLogCertMqtt4(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	testConnectionLog(t, "certificate", client.MQTT4)
}

func TestConnectionLogCertMqtt5(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	testConnectionLog(t, "certificate", client.MQTT5)
}

func testConnectionLog(t *testing.T, authMethod string, mqttVersion client.MqttVersion) {
	if mqttVersion == client.MQTT5 {
		t.Skip("clean-start=false is currently not supported by mqtt 5 paho client")
	}
	defaultConfig, err := configuration.Load("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.InitTopics = true
	defaultConfig.MqttAuthMethod = authMethod
	if mqttVersion == client.MQTT5 {
		defaultConfig.MqttVersion = "5"
	}

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, brokerForClients, err := server.NewWithConnectionLog(ctx, wg, defaultConfig)
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
	serviceLocalId := "testservice1"
	serviceId := "urn:infai:ses:service:efed3e07-e738-445f-8a4f-847b87688506"
	deviceType := model.DeviceType{}
	protocol := model.Protocol{}

	devices := []model.Device{}

	t.Run("create protocol", func(t *testing.T) {
		protocol = createTestProtocol(t, config)
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceType(t, config, protocol, serviceLocalId, serviceId)
	})

	t.Run("create devices", func(t *testing.T) {
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_0", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_1", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_2", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_3", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_4", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_5", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_6", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_7", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_8", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_9", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_10", ""))
		devices = append(devices, createTestDevice(t, config, deviceType, deviceLocalId+"_11", ""))
	})

	state := &deviceLogState{}

	err = kafka.NewConsumer(ctx,
		kafka.ConsumerConfig{
			KafkaUrl: config.KafkaUrl,
			Topic:    config.DeviceLogTopic,
			GroupId:  "check_consumer" + uuid.NewString(),
			MaxWait:  100 * time.Millisecond,
			//nothing has produced to the device-log topic yet, and a consumer
			//group that starts on a topic the broker does not know does not
			//reliably recover once the topic appears
			InitTopic:      true,
			TopicConfigMap: config.KafkaTopicConfigs,
		}, state.consume, state.handleError)
	if err != nil {
		t.Error(err)
		return
	}

	expected := []DeviceLog{}

	t.Run("run", func(t *testing.T) {
		client1, err := client.New(brokerForClients, "bar", "foo", "client1", authMethod, mqttVersion, false, false)
		if err != nil {
			t.Error(err)
			return
		}
		defer client1.Stop()

		for i := 0; i < 8; i++ {
			device := devices[i]
			err := client1.Subscribe(device.Id+"/"+serviceLocalId, 2, func(topic string, pl []byte) {})
			if err != nil {
				t.Error(err)
				return
			}
			expected = append(expected, DeviceLog{Id: devices[i].Id, Connected: true})

			err = client1.Subscribe(device.Id+"/"+serviceLocalId+"_2", 2, func(topic string, pl []byte) {})
			if err != nil {
				t.Error(err)
				return
			}
			expected = append(expected, DeviceLog{Id: devices[i].Id, Connected: true})
		}

		t.Run("check 1", func(t *testing.T) {
			waitFor(t, 60*time.Second, deviceLogSettle, state.matches(expected, devices))
		})

		client2, err := client.New(brokerForClients, "bar", "foo", "client2", authMethod, mqttVersion, true, false)
		if err != nil {
			t.Error(err)
			return
		}
		defer client2.Stop()

		for i := 4; i < 12; i++ {
			device := devices[i]
			err = client2.Subscribe(device.Id+"/"+serviceLocalId, 2, func(topic string, pl []byte) {})
			if err != nil {
				t.Error(err)
				return
			}
			expected = append(expected, DeviceLog{Id: devices[i].Id, Connected: true})

			err = client2.Subscribe(device.Id+"/"+serviceLocalId+"_2", 2, func(topic string, pl []byte) {})
			if err != nil {
				t.Error(err)
				return
			}
			expected = append(expected, DeviceLog{Id: devices[i].Id, Connected: true})
		}

		t.Run("check 2", func(t *testing.T) {
			waitFor(t, 60*time.Second, deviceLogSettle, state.matches(expected, devices))
		})

		//no disconnect because second service is still used
		err = client1.Unsubscribe(devices[0].Id + "/" + serviceLocalId)
		if err != nil {
			t.Error(err)
			return
		}

		//disconnect device[1]
		err = client1.Unsubscribe(devices[1].Id+"/"+serviceLocalId, devices[1].Id+"/"+serviceLocalId+"_2")
		if err != nil {
			t.Error(err)
			return
		}
		expected = append(expected, DeviceLog{Id: devices[1].Id, Connected: false})

		//no disconnect because client2 uses device[4]
		err = client1.Unsubscribe(devices[4].Id+"/"+serviceLocalId, devices[4].Id+"/"+serviceLocalId+"_2")
		if err != nil {
			t.Error(err)
			return
		}

		//no disconnect because second service is still used
		err = client2.Unsubscribe(devices[11].Id + "/" + serviceLocalId)
		if err != nil {
			t.Error(err)
			return
		}

		t.Run("check 3", func(t *testing.T) {
			waitFor(t, 60*time.Second, deviceLogSettle, state.matches(expected, devices))
		})

		//disconnect device[10]
		err = client2.Unsubscribe(devices[10].Id+"/"+serviceLocalId, devices[10].Id+"/"+serviceLocalId+"_2")
		if err != nil {
			t.Error(err)
			return
		}
		expected = append(expected, DeviceLog{Id: devices[10].Id, Connected: false})

		//no disconnect because client1 uses device[7]
		err = client2.Unsubscribe(devices[7].Id+"/"+serviceLocalId, devices[7].Id+"/"+serviceLocalId+"_2")
		if err != nil {
			t.Error(err)
			return
		}

		t.Run("check 4", func(t *testing.T) {
			waitFor(t, 60*time.Second, deviceLogSettle, state.matches(expected, devices))
		})

		//disconnect client 2 --> disconnect devices 4, 8, 9, 11
		client2.Stop()
		expected = append(expected, DeviceLog{Id: devices[4].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[8].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[9].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[11].Id, Connected: false})

		t.Run("check 5", func(t *testing.T) {
			waitFor(t, 60*time.Second, deviceLogSettle, state.matches(expected, devices))
		})

		//disconnect client 1 --> disconnect devices 0, 2, 3, 5, 6, 7
		client1.Stop()
		expected = append(expected, DeviceLog{Id: devices[0].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[2].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[3].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[5].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[6].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[7].Id, Connected: false})

		t.Run("check 6", func(t *testing.T) {
			waitFor(t, 60*time.Second, deviceLogSettle, state.matches(expected, devices))
		})

		//reconnect client2 --> no device connect because clean session = true
		client2, err = client.New(brokerForClients, "bar", "foo", "client2", authMethod, mqttVersion, true, false)
		if err != nil {
			t.Error(err)
			return
		}

		//reconnect client 1 --> connect devices 0, 2, 3, 5, 6, 7 because clean session = false
		client1, err = client.New(brokerForClients, "bar", "foo", "client1", authMethod, mqttVersion, false, false)
		if err != nil {
			t.Error(err)
			return
		}
		expected = append(expected, DeviceLog{Id: devices[0].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[2].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[3].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[5].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[6].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[7].Id, Connected: true})

		t.Run("check 7", func(t *testing.T) {
			waitFor(t, 60*time.Second, deviceLogSettle, state.matches(expected, devices))
		})

		//disconnect all --> disconnect devices 0, 2, 3, 5, 6, 7
		client1.Stop()
		client2.Stop()
		expected = append(expected, DeviceLog{Id: devices[0].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[2].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[3].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[5].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[6].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[7].Id, Connected: false})

		t.Run("final check", func(t *testing.T) {
			waitFor(t, 60*time.Second, deviceLogSettle, state.matches(expected, devices))
		})
	})

}

// deviceLogState collects what the device-log consumer received. The consumer
// runs in its own goroutine and outlives the test, so it records messages and
// errors instead of asserting: t.Error from a goroutine that keeps running after
// the test has finished panics the whole test binary.
type deviceLogState struct {
	mux      sync.Mutex
	messages []DeviceLog
	errs     []error
}

func (this *deviceLogState) consume(_ string, msg []byte, _ time.Time) error {
	logmsg := DeviceLog{}
	err := json.Unmarshal(msg, &logmsg)
	this.mux.Lock()
	defer this.mux.Unlock()
	if err != nil {
		this.errs = append(this.errs, err)
		return err
	}
	this.messages = append(this.messages, logmsg)
	return nil
}

func (this *deviceLogState) handleError(err error) {
	log.Println("consumer error:", err)
	this.mux.Lock()
	defer this.mux.Unlock()
	this.errs = append(this.errs, err)
}

func (this *deviceLogState) snapshot() ([]DeviceLog, error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	if len(this.errs) > 0 {
		return nil, errors.Join(this.errs...)
	}
	return slices.Clone(this.messages), nil
}

// matchesInOrder returns a check for waitFor that compares the received
// device-logs to expected in the order they were produced.
func (this *deviceLogState) matchesInOrder(expected []DeviceLog, devices []model.Device) func() error {
	return func() error {
		actual, err := this.snapshot()
		if err != nil {
			return err
		}
		if reflect.DeepEqual(actual, expected) {
			return nil
		}
		return deviceLogDiff(makeMessagesReadable(expected, devices), makeMessagesReadable(actual, devices))
	}
}

// matches returns a check for waitFor that compares the received device-logs to
// expected ignoring their order, because they come from several mqtt clients
// whose webhooks are handled concurrently.
func (this *deviceLogState) matches(expected []DeviceLog, devices []model.Device) func() error {
	return func() error {
		actual, err := this.snapshot()
		if err != nil {
			return err
		}
		expectedReadable := makeMessagesReadable(expected, devices)
		actualReadable := makeMessagesReadable(actual, devices)
		if reflect.DeepEqual(expectedReadable, actualReadable) {
			return nil
		}
		return deviceLogDiff(expectedReadable, actualReadable)
	}
}

// deviceLogDiff renders the difference so a run that never matches prints what
// was missing instead of only the fact that it timed out.
func deviceLogDiff(expected []DeviceLog, actual []DeviceLog) error {
	expectedJson, _ := json.Marshal(expected)
	actualJson, _ := json.Marshal(actual)
	return fmt.Errorf("expected %s\ngot      %s", expectedJson, actualJson)
}

func makeMessagesReadable(messages []DeviceLog, devices []model.Device) (result []DeviceLog) {
	idToIndex := map[string]string{}
	for index, device := range devices {
		idToIndex[device.Id] = strconv.Itoa(index)
	}
	for _, msg := range messages {
		msg.Id = idToIndex[msg.Id]
		result = append(result, msg)
	}
	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]
		if a.Id == b.Id {
			if a.Connected == b.Connected {
				return false
			}
			return !a.Connected
		}
		return a.Id < b.Id
	})
	return result
}

type DeviceLog struct {
	Id        string `json:"id"`
	Connected bool   `json:"connected"`
}
