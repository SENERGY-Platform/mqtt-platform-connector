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

package client

import (
	"context"
	"encoding/json"
	"errors"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"time"
)

type MqttVersion int

const (
	MQTT4 MqttVersion = iota
	MQTT5
)

func (this *Client) startMqtt() error {
	if this.mqttVersion == MQTT4 {
		return this.startMqtt4()
	}
	if this.mqttVersion == MQTT5 {
		return this.startMqtt5()
	}
	return errors.New("unknown mqtt version")
}

func (this *Client) Publish(topic string, msg interface{}, qos byte) (err error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return this.PublishRaw(topic, payload, qos)
}

func (this *Client) PublishRaw(topic string, payload []byte, qos byte) (err error) {
	log.Println("DEBUG: publish", topic, string(payload))
	if this.mqttVersion == MQTT4 {
		return this.PublishMqtt4(topic, payload, qos)
	}
	if this.mqttVersion == MQTT5 {
		return this.PublishMqtt5(topic, payload, qos)
	}
	return errors.New("unknown mqtt version")
}

func (this *Client) Stop() {
	if this.mqttVersion == MQTT4 {
		this.mqtt.Disconnect(200)
	}
	if this.mqttVersion == MQTT5 {
		disconnecttimeout, _ := context.WithTimeout(context.Background(), 1*time.Second)
		this.mqtt5.Disconnect(disconnecttimeout)
	}
}

func (this *Client) Subscribe(topic string, qos byte, callback func(topic string, pl []byte)) error {
	if this.mqttVersion == MQTT4 {
		return this.SubscribeMqtt4(topic, qos, callback)
	}
	if this.mqttVersion == MQTT5 {
		return this.SubscribeMqtt5(topic, qos, callback)
	}
	return errors.New("unknown mqtt version")
}

func (this *Client) Unsubscribe(topic ...string) error {
	if this.mqttVersion == MQTT4 {
		return this.UnsubscribeMqtt4(topic...)
	}
	if this.mqttVersion == MQTT5 {
		return this.UnsubscribeMqtt5(topic...)
	}
	return errors.New("unknown mqtt version")
}

type Subscription struct {
	Topic   string
	Handler paho.MessageHandler
	Qos     byte
}

func (this *Client) registerSubscription(topic string, qos byte, handler paho.MessageHandler) {
	this.subscriptionsMux.Lock()
	defer this.subscriptionsMux.Unlock()
	this.subscriptions[topic] = Subscription{
		Topic:   topic,
		Handler: handler,
		Qos:     qos,
	}
}

func (this *Client) unregisterSubscriptions(topic string) {
	this.subscriptionsMux.Lock()
	defer this.subscriptionsMux.Unlock()
	delete(this.subscriptions, topic)
}

func (this *Client) getSubscriptions() (result []Subscription) {
	this.subscriptionsMux.Lock()
	defer this.subscriptionsMux.Unlock()
	for _, sub := range this.subscriptions {
		result = append(result, sub)
	}
	return
}

func (this *Client) loadOldSubscriptions() error {
	if this.mqttVersion == MQTT4 {
		return this.loadOldSubscriptionsMqtt4()
	}
	if this.mqttVersion == MQTT5 {
		return this.loadOldSubscriptionsMqtt5()
	}
	return errors.New("unknown mqtt version")
}
