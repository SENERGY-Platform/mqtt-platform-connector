package server

import (
	"context"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/docker"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/auth"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/iot"
	"github.com/ory/dockertest/v3"
	"log"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
)

func NewWithConnectionLog(basectx context.Context, wg *sync.WaitGroup, defaults lib.Config) (config lib.Config, err error) {
	ctx, cancel := context.WithCancel(basectx)
	config, err = New(ctx, wg, defaults)
	if err != nil {
		return
	}
	config.SubscriptionDbConStr,  _, _, err = docker.Postgres(ctx, wg, "subscriptions")
	if err != nil {
		cancel()
		return config, err
	}
	return config, nil
}

func New(basectx context.Context, wg *sync.WaitGroup, defaults lib.Config) (config lib.Config, err error) {
	config = defaults
	config.KafkaPartitionNum = 1
	config.KafkaReplicationFactor = 1

	ctx, cancel := context.WithCancel(basectx)

	err = auth.Mock(&config, ctx)
	if err != nil {
		cancel()
		return config, err
	}

	whPort, err := getFreePort()
	if err != nil {
		log.Println("unable to find free port", err)
		return config, err
	}
	config.WebhookPort = strconv.Itoa(whPort)

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Println("Could not connect to docker: ", err)
		return config, err
	}

	_, zk, err := docker.Zookeeper(pool, ctx)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	zkUrl := zk + ":2181"

	config.KafkaUrl, err = docker.Kafka(pool, ctx, wg, zkUrl)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	config.DeviceManagerUrl, config.DeviceRepoUrl, err = iot.Mock(ctx, config)
	if err != nil {
		cancel()
		return config, err
	}

	_, memcacheIp, err := docker.Memcached(pool, ctx, wg)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	config.IotCacheUrl = []string{memcacheIp + ":11211"}
	config.TokenCacheUrl = []string{memcacheIp + ":11211"}

	network, err := pool.Client.NetworkInfo("bridge")
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	hostIp := network.IPAM.Config[0].Gateway
	config.MqttBroker, err = docker.Vernemqtt(pool, ctx, wg, hostIp+":"+config.WebhookPort)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresPw, config.PostgresDb, err = docker.Timescale(pool, ctx, wg)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	hostIp = "127.0.0.1"
	networks, _ := pool.Client.ListNetworks()
	for _, network := range networks {
		if network.Name == "bridge" {
			hostIp = network.IPAM.Config[0].Gateway
			break
		}
	}

	//transform local-address to address in docker container
	deviceManagerUrlStruct := strings.Split(config.DeviceManagerUrl, ":")
	deviceManagerUrl := "http://" + hostIp + ":" + deviceManagerUrlStruct[len(deviceManagerUrlStruct)-1]
	log.Println("DEBUG: semantic url transformation:", config.DeviceManagerUrl, "-->", deviceManagerUrl)

	err = docker.Tableworker(pool, ctx, wg, config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresPw, config.PostgresDb, config.KafkaUrl, deviceManagerUrl)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	return config, nil
}

func getFreePortStr() (string, error) {
	intPort, err := getFreePort()
	if err != nil {
		return "", err
	}
	return strconv.Itoa(intPort), nil
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
