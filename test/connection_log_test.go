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
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestConnectionLogDevice1Minimal(t *testing.T) {
	defaultConfig, err := lib.LoadConfigLocation("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := server.NewWithConnectionLog(ctx, wg, defaultConfig)
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
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceType(t, config, protocol, serviceLocalId, serviceId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create devices", func(t *testing.T) {
		device = createTestDevice(t, config, deviceType, deviceLocalId+"_1", "")
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	expected := []DeviceLog{}

	t.Run("run", func(t *testing.T) {
		client1 := paho.NewClient(paho.NewClientOptions().
			SetPassword("foo").
			SetUsername("bar").
			SetClientID("client1").
			SetAutoReconnect(false).
			SetCleanSession(false).
			AddBroker(config.MqttBroker))
		if token := client1.Connect(); token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}

		token := client1.Subscribe(device.Id+"/"+serviceLocalId, 2, func(client paho.Client, message paho.Message) {})
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}
		expected = append(expected, DeviceLog{Id: device.Id, Connected: true})

		token = client1.Subscribe(device.Id+"/"+serviceLocalId+"_2", 2, func(client paho.Client, message paho.Message) {})
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}
		expected = append(expected, DeviceLog{Id: device.Id, Connected: true})

		time.Sleep(2 * time.Second)

		//disconnect device[1]
		token = client1.Unsubscribe(device.Id+"/"+serviceLocalId, device.Id+"/"+serviceLocalId+"_2")
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}
		expected = append(expected, DeviceLog{Id: device.Id, Connected: false})

	})

	t.Run("check", func(t *testing.T) {
		logMessages := []DeviceLog{}
		log.Println("consume", config.DeviceLogTopic)
		err = kafka.NewConsumer(ctx,
			kafka.ConsumerConfig{
				KafkaUrl: config.KafkaUrl,
				Topic:    config.DeviceLogTopic,
				GroupId:  "check_consumer",
			}, func(topic string, msg []byte, time time.Time) error {
				logmsg := DeviceLog{}
				err = json.Unmarshal(msg, &logmsg)
				if err != nil {
					t.Error(err)
					return nil
				}
				logMessages = append(logMessages, logmsg)
				return nil
			}, func(err error) {
				log.Println("consumer error:", err)
			})
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(20 * time.Second)

		if !reflect.DeepEqual(logMessages, expected) {
			expectedJson, _ := json.Marshal(makeMessagesReadable(expected, []model.Device{device}))
			actualJson, _ := json.Marshal(makeMessagesReadable(logMessages, []model.Device{device}))
			t.Error(string(expectedJson), "\n", string(actualJson))
		}
	})
}

func TestConnectionLog(t *testing.T) {
	defaultConfig, err := lib.LoadConfigLocation("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := server.NewWithConnectionLog(ctx, wg, defaultConfig)
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
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceType(t, config, protocol, serviceLocalId, serviceId)
		time.Sleep(10 * time.Second) //wait for cqrs
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
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	expected := []DeviceLog{}

	t.Run("run", func(t *testing.T) {
		client1 := paho.NewClient(paho.NewClientOptions().
			SetPassword("foo").
			SetUsername("bar").
			SetClientID("client1").
			SetAutoReconnect(false).
			SetCleanSession(false).
			AddBroker(config.MqttBroker))
		if token := client1.Connect(); token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}

		for i := 0; i < 8; i++ {
			device := devices[i]
			token := client1.Subscribe(device.Id+"/"+serviceLocalId, 2, func(client paho.Client, message paho.Message) {})
			if token.Wait() && token.Error() != nil {
				t.Error(token.Error())
				return
			}
			expected = append(expected, DeviceLog{Id: devices[i].Id, Connected: true})

			token = client1.Subscribe(device.Id+"/"+serviceLocalId+"_2", 2, func(client paho.Client, message paho.Message) {})
			if token.Wait() && token.Error() != nil {
				t.Error(token.Error())
				return
			}
			expected = append(expected, DeviceLog{Id: devices[i].Id, Connected: true})
		}

		client2 := paho.NewClient(paho.NewClientOptions().
			SetPassword("foo").
			SetUsername("bar").
			SetClientID("client2").
			SetAutoReconnect(false).
			SetCleanSession(true).
			AddBroker(config.MqttBroker))
		if token := client2.Connect(); token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}

		for i := 4; i < 12; i++ {
			device := devices[i]
			token := client2.Subscribe(device.Id+"/"+serviceLocalId, 2, func(client paho.Client, message paho.Message) {})
			if token.Wait() && token.Error() != nil {
				t.Error(token.Error())
				return
			}
			expected = append(expected, DeviceLog{Id: devices[i].Id, Connected: true})

			token = client2.Subscribe(device.Id+"/"+serviceLocalId+"_2", 2, func(client paho.Client, message paho.Message) {})
			if token.Wait() && token.Error() != nil {
				t.Error(token.Error())
				return
			}
			expected = append(expected, DeviceLog{Id: devices[i].Id, Connected: true})
		}

		time.Sleep(2 * time.Second)

		//no disconnect because second service is still used
		token := client1.Unsubscribe(devices[0].Id + "/" + serviceLocalId)
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}

		//disconnect device[1]
		token = client1.Unsubscribe(devices[1].Id+"/"+serviceLocalId, devices[1].Id+"/"+serviceLocalId+"_2")
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}
		expected = append(expected, DeviceLog{Id: devices[1].Id, Connected: false})

		//no disconnect because client2 uses device[4]
		token = client1.Unsubscribe(devices[4].Id+"/"+serviceLocalId, devices[4].Id+"/"+serviceLocalId+"_2")
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}

		//no disconnect because second service is still used
		token = client2.Unsubscribe(devices[11].Id + "/" + serviceLocalId)
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}

		//disconnect device[10]
		token = client2.Unsubscribe(devices[10].Id+"/"+serviceLocalId, devices[10].Id+"/"+serviceLocalId+"_2")
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}
		expected = append(expected, DeviceLog{Id: devices[10].Id, Connected: false})

		//no disconnect because client1 uses device[7]
		token = client2.Unsubscribe(devices[7].Id+"/"+serviceLocalId, devices[7].Id+"/"+serviceLocalId+"_2")
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}

		//disconnect client 2 --> disconnect devices 4, 8, 9, 11
		client2.Disconnect(0)
		expected = append(expected, DeviceLog{Id: devices[4].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[8].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[9].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[11].Id, Connected: false})
		time.Sleep(1 * time.Second)

		//disconnect client 1 --> disconnect devices 0, 2, 3, 5, 6, 7
		client1.Disconnect(0)
		expected = append(expected, DeviceLog{Id: devices[0].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[2].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[3].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[5].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[6].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[7].Id, Connected: false})
		time.Sleep(1 * time.Second)

		//reconnect client2 --> no device connect because clean session = true
		client2 = paho.NewClient(paho.NewClientOptions().
			SetPassword("foo").
			SetUsername("bar").
			SetClientID("client2").
			SetAutoReconnect(false).
			SetCleanSession(true).
			AddBroker(config.MqttBroker))
		if token := client2.Connect(); token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}

		//reconnect client 1 --> connect devices 0, 2, 3, 5, 6, 7 because clean session = false
		client1 = paho.NewClient(paho.NewClientOptions().
			SetPassword("foo").
			SetUsername("bar").
			SetClientID("client1").
			SetAutoReconnect(false).
			SetCleanSession(false).
			AddBroker(config.MqttBroker))
		if token := client1.Connect(); token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}
		expected = append(expected, DeviceLog{Id: devices[0].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[2].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[3].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[5].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[6].Id, Connected: true})
		expected = append(expected, DeviceLog{Id: devices[7].Id, Connected: true})

		//disconnect all --> disconnect devices 0, 2, 3, 5, 6, 7
		client1.Disconnect(0)
		client2.Disconnect(0)
		expected = append(expected, DeviceLog{Id: devices[0].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[2].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[3].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[5].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[6].Id, Connected: false})
		expected = append(expected, DeviceLog{Id: devices[7].Id, Connected: false})
	})

	t.Run("check", func(t *testing.T) {
		logMessages := []DeviceLog{}
		log.Println("consume", config.DeviceLogTopic)
		err = kafka.NewConsumer(ctx,
			kafka.ConsumerConfig{
				KafkaUrl: config.KafkaUrl,
				Topic:    config.DeviceLogTopic,
				GroupId:  "check_consumer",
			}, func(topic string, msg []byte, time time.Time) error {
				logmsg := DeviceLog{}
				err = json.Unmarshal(msg, &logmsg)
				if err != nil {
					t.Error(err)
					return nil
				}
				logMessages = append(logMessages, logmsg)
				return nil
			}, func(err error) {
				log.Println("consumer error:", err)
			})
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(20 * time.Second)

		sortedReadableExpected := makeMessagesReadable(expected, devices)
		sortedReadableActual := makeMessagesReadable(logMessages, devices)
		if !reflect.DeepEqual(sortedReadableExpected, sortedReadableActual) {
			expectedJson, _ := json.Marshal(sortedReadableExpected)
			actualJson, _ := json.Marshal(sortedReadableActual)
			t.Error(string(expectedJson), "\n", string(actualJson))
		}
	})
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
