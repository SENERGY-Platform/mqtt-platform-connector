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

package server

import (
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/ory/dockertest"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

func New(startConfig platform_connector_lib.Config) (connector *platform_connector_lib.Connector, libConfig platform_connector_lib.Config, shutdown func(), err error) {
	libConfig = startConfig

	whPort, err := getFreePort()
	if err != nil {
		log.Println("unable to find free port", err)
		return connector, libConfig, func() {}, err
	}
	lib.Config.WebhookPort = strconv.Itoa(whPort)

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Println("Could not connect to docker: %s", err)
		return connector, libConfig, func() {}, err
	}

	network, err := pool.Client.NetworkInfo("bridge")
	if err != nil {
		return connector, libConfig, func() {}, err
	}
	hostIp := network.IPAM.Config[0].Gateway

	closerList := []func(){}
	close := func(list []func()) {
		for i := len(list)/2 - 1; i >= 0; i-- {
			opp := len(list) - 1 - i
			list[i], list[opp] = list[opp], list[i]
		}
		for _, c := range list {
			if c != nil {
				c()
			}
		}
	}

	mux := sync.Mutex{}
	var globalError error
	wait := sync.WaitGroup{}

	//zookeeper
	zkWait := sync.WaitGroup{}
	zkWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer zkWait.Done()
		closer, _, zkIp, err := Zookeeper(pool)
		mux.Lock()
		defer mux.Unlock()
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
		libConfig.ZookeeperUrl = zkIp + ":2181"
	}()

	//kafka
	kafkaWait := sync.WaitGroup{}
	kafkaWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer kafkaWait.Done()
		zkWait.Wait()
		if globalError != nil {
			return
		}
		closer, err := Kafka(pool, libConfig.ZookeeperUrl)
		mux.Lock()
		defer mux.Unlock()
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
	}()

	var elasticIp string
	var mongoIp string

	//kafka
	elasticWait := sync.WaitGroup{}
	elasticWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer elasticWait.Done()
		if globalError != nil {
			return
		}
		closer, _, ip, err := Elasticsearch(pool)
		elasticIp = ip
		mux.Lock()
		defer mux.Unlock()
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
	}()

	mongoWait := sync.WaitGroup{}
	mongoWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer mongoWait.Done()
		if globalError != nil {
			return
		}
		closer, _, ip, err := MongoTestServer(pool)
		mongoIp = ip
		mux.Lock()
		defer mux.Unlock()
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
	}()

	var permissionUrl string
	permWait := sync.WaitGroup{}
	permWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer permWait.Done()
		kafkaWait.Wait()
		elasticWait.Wait()
		if globalError != nil {
			return
		}
		closer, _, permIp, err := PermSearch(pool, libConfig.ZookeeperUrl, elasticIp)
		mux.Lock()
		defer mux.Unlock()
		permissionUrl = "http://" + permIp + ":8080"
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
	}()

	//device-repo
	deviceRepoWait := sync.WaitGroup{}
	deviceRepoWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer deviceRepoWait.Done()
		mongoWait.Wait()
		zkWait.Wait()
		kafkaWait.Wait()
		permWait.Wait()
		if globalError != nil {
			return
		}
		closer, _, ip, err := DeviceRepo(pool, mongoIp, libConfig.ZookeeperUrl, permissionUrl)
		mux.Lock()
		defer mux.Unlock()
		libConfig.DeviceRepoUrl = "http://" + ip + ":8080"
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
	}()

	//device-manager
	deviceManagerWait := sync.WaitGroup{}
	deviceManagerWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer deviceManagerWait.Done()
		deviceRepoWait.Wait()
		mongoWait.Wait()
		if globalError != nil {
			return
		}
		closer, _, ip, err := DeviceManager(pool, libConfig.ZookeeperUrl, libConfig.DeviceRepoUrl, "-", permissionUrl)
		mux.Lock()
		defer mux.Unlock()
		libConfig.DeviceManagerUrl = "http://" + ip + ":8080"
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
	}()

	//memcached
	cacheWait := sync.WaitGroup{}
	cacheWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer cacheWait.Done()
		if globalError != nil {
			return
		}
		closer, _, ip, err := Memcached(pool)
		mux.Lock()
		defer mux.Unlock()
		libConfig.IotCacheUrl = []string{ip + ":11211"}
		libConfig.TokenCacheUrl = []string{ip + ":11211"}
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
	}()

	keycloakWait := sync.WaitGroup{}
	keycloakWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer keycloakWait.Done()
		if globalError != nil {
			return
		}
		closer, _, ip, err := Keycloak(pool)
		mux.Lock()
		defer mux.Unlock()
		libConfig.AuthEndpoint = "http://" + ip + ":8080"
		libConfig.AuthClientSecret = "d61daec4-40d6-4d3e-98c9-f3b515696fc6"
		libConfig.AuthClientId = "connector"
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
	}()

	//vernemq
	mqttWait := sync.WaitGroup{}
	mqttWait.Add(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		defer mqttWait.Done()
		if globalError != nil {
			return
		}
		closer, _, ip, err := Vernemqtt(pool, hostIp+":"+lib.Config.WebhookPort)
		mux.Lock()
		defer mux.Unlock()
		lib.Config.MqttBroker = "tcp://" + ip + ":1883"
		closerList = append(closerList, closer)
		if err != nil {
			globalError = err
			return
		}
	}()


	wait.Wait()
	if globalError != nil {
		close(closerList)
		return connector, libConfig, shutdown, globalError
	}
	//senergy-connector

	connector = platform_connector_lib.New(libConfig)

	connector.SetKafkaLogger(log.New(os.Stdout, "[CONNECTOR-KAFKA] ", 0))


	go lib.AuthWebhooks(connector)

	err = lib.MqttStart()
	if err != nil {
		close(closerList)
		return connector, libConfig, shutdown, err
	}


	connector.SetDeviceCommandHandler(lib.CommandHandler)

	err = connector.Start()
	if err != nil {
		close(closerList)
		return connector, libConfig, shutdown, err
	}
	closerList = append(closerList, func() {
		connector.Stop()
	})

	return connector, libConfig, func() { close(closerList) }, nil
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
