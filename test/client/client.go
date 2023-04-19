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

package client

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	paho4 "github.com/eclipse/paho.mqtt.golang"
	"io/ioutil"
	"log"
	"sync"
)

func New(mqttUrl string, userName string, password string, hubId string, authenticationMethod string, mqttVersion MqttVersion, cleanSession bool, autoReconnect bool) (client *Client, err error) {
	client = &Client{
		mqttUrl:              mqttUrl,
		HubId:                hubId,
		username:             userName,
		password:             password,
		subscriptions:        map[string]Subscription{},
		authenticationMethod: authenticationMethod,
		mqttVersion:          mqttVersion,
		cleanSession:         cleanSession,
		autoReconnect:        autoReconnect,
	}
	err = client.startMqtt()
	return
}

type Client struct {
	mqttVersion MqttVersion
	mqttUrl     string
	HubId       string

	username string
	password string

	mqtt        paho4.Client
	mqtt5       *autopaho.ConnectionManager
	mqtt5router paho.Router

	subscriptionsMux sync.Mutex
	subscriptions    map[string]Subscription

	authenticationMethod string
	cleanSession         bool
	autoReconnect        bool
}

func CreateTLSConfig(ClientCertificatePath string, PrivateKeyPath string, RootCACertificatePath string) (tlsConfig *tls.Config, err error) {
	// Load Client/Server Certificate
	var cer tls.Certificate
	cer, err = tls.LoadX509KeyPair(ClientCertificatePath, PrivateKeyPath)
	if err != nil {
		log.Println("Error on TLS: cant read client certificate", err)
		return nil, err
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(RootCACertificatePath)
	if err != nil {
		log.Println("Error on TLS: cant read CA certificate", err)
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cer},
		RootCAs:      caCertPool,

		// This circumvents the need for matching request hostname and server certficate CN field
		InsecureSkipVerify: true,
	}
	return
}
