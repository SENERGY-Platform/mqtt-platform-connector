/*
 * Copyright 2023 InfAI (CC SES)
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
	"errors"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
)

func (this *Client) startMqtt4() error {
	options := paho.NewClientOptions().
		SetClientID(this.HubId).
		SetAutoReconnect(this.autoReconnect).
		SetCleanSession(this.cleanSession).
		AddBroker(this.mqttUrl).
		SetConnectionLostHandler(func(client paho.Client, err error) {
			log.Println("mqtt connection lost:", err)
		}).
		SetOnConnectHandler(func(client paho.Client) {
			log.Println("mqtt (re)connected")
			err := this.loadOldSubscriptions()
			if err != nil {
				debug.PrintStack()
				log.Fatal("FATAL: ", err)
			}
		})

	if this.authenticationMethod == "certificate" {
		dir, err := os.Getwd()
		ClientCertificatePath := filepath.Join(dir, "mqtt_certs", "mock_client", "client.crt")
		PrivateKeyPath := filepath.Join(dir, "mqtt_certs", "mock_client", "private.key")
		RootCACertificatePath := filepath.Join(dir, "mqtt_certs", "ca", "ca.crt")

		tlsConfig, err := CreateTLSConfig(ClientCertificatePath, PrivateKeyPath, RootCACertificatePath)
		if err != nil {
			log.Println("Error on MQTT TLS config", err)
			return err
		}
		options = options.SetTLSConfig(tlsConfig)
	} else {
		options = options.SetUsername(this.username).SetPassword(this.password)
	}

	this.mqtt = paho.NewClient(options)
	if token := this.mqtt.Connect(); token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Connect(): ", token.Error())
		return token.Error()
	}
	return nil
}

func (this *Client) SubscribeMqtt4(topic string, qos byte, callback func(topic string, pl []byte)) error {
	if !this.mqtt.IsConnected() {
		log.Println("WARNING: mqtt client not connected")
		return errors.New("mqtt client not connected")
	}
	f := func(client paho.Client, message paho.Message) {
		callback(message.Topic(), message.Payload())
	}
	token := this.mqtt.Subscribe(topic, qos, f)
	if token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Subscribe(): ", token.Error())
		return token.Error()
	}
	this.registerSubscription(topic, qos, f)
	return nil
}

func (this *Client) UnsubscribeMqtt4(topic ...string) (err error) {
	if !this.mqtt.IsConnected() {
		log.Println("WARNING: mqtt client not connected")
		return errors.New("mqtt client not connected")
	}
	token := this.mqtt.Unsubscribe(topic...)
	if token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Unsubscribe(): ", token.Error())
		return token.Error()
	}
	return nil
}

func (this *Client) PublishMqtt4(topic string, payload []byte, qos byte) (err error) {
	if !this.mqtt.IsConnected() {
		log.Println("WARNING: mqtt client not connected")
		return errors.New("mqtt client not connected")
	}
	token := this.mqtt.Publish(topic, qos, false, string(payload))
	if token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Publish(): ", token.Error())
		return token.Error()
	}
	return err
}

func (this *Client) loadOldSubscriptionsMqtt4() error {
	if !this.mqtt.IsConnected() {
		log.Println("WARNING: mqtt client not connected")
		return errors.New("mqtt client not connected")
	}
	subs := this.getSubscriptions()
	for _, sub := range subs {
		log.Println("resubscribe to", sub.Topic)
		token := this.mqtt.Subscribe(sub.Topic, sub.Qos, sub.Handler)
		if token.Wait() && token.Error() != nil {
			log.Println("Error on Subscribe: ", sub.Topic, token.Error())
			return token.Error()
		}
	}
	return nil
}
