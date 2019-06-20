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

var mqttClient paho.Client

func MqttClose() {
	// Terminate the Client.
	mqttClient.Disconnect(0)
}

func MqttStart() error {
	options := paho.NewClientOptions().
		SetPassword(Config.AuthClientSecret).
		SetUsername(Config.AuthClientId).
		SetClientID(Config.MqttClientId).
		SetAutoReconnect(true).
		SetCleanSession(true).
		AddBroker(Config.MqttBroker)

	mqttClient = paho.NewClient(options)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Connect(): ", token.Error())
		return token.Error()
	}

	return nil
}

var mqttPublishMux = sync.Mutex{}

func MqttPublish(topic, msg string) (err error) {
	mqttPublishMux.Lock()
	defer mqttPublishMux.Unlock()
	log.Println("DEBUG mqtt out: ", topic, msg)
	if !mqttClient.IsConnected() {
		log.Println("WARNING: mqtt client not connected")
		return errors.New("mqtt client not connected")
	}
	token := mqttClient.Publish(topic, Config.Qos, false, msg)
	if token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Publish(): ", token.Error())
		return token.Error()
	}
	return err
}
