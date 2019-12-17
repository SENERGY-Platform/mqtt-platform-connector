/*
 * Copyright 2018 InfAI (CC SES)
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

package lib

import (
	"errors"
	"log"

	"sync"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type MqttClient struct {
	client paho.Client
	config Config
}

func (this *MqttClient) Close() {
	// Terminate the Client.
	this.client.Disconnect(0)
}

func NewMqtt(config Config) (result *MqttClient, err error) {
	options := paho.NewClientOptions().
		SetPassword(config.AuthClientSecret).
		SetUsername(config.AuthClientId).
		SetClientID(config.MqttClientId).
		SetAutoReconnect(true).
		SetCleanSession(true).
		AddBroker(config.MqttBroker)
	result = &MqttClient{config: config}
	result.client = paho.NewClient(options)
	if token := result.client.Connect(); token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Connect(): ", token.Error())
		return result, token.Error()
	}

	return result, nil
}

var mqttPublishMux = sync.Mutex{}

func (this *MqttClient) Publish(topic, msg string) (err error) {
	mqttPublishMux.Lock()
	defer mqttPublishMux.Unlock()
	log.Println("DEBUG mqtt out: ", topic, msg)
	if !this.client.IsConnected() {
		log.Println("WARNING: mqtt client not connected")
		return errors.New("mqtt client not connected")
	}
	token := this.client.Publish(topic, this.config.Qos, false, msg)
	if token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Publish(): ", token.Error())
		return token.Error()
	}
	return err
}
