package server

import (
	"context"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/docker"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/auth"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/iot"
	"github.com/testcontainers/testcontainers-go"
	"log"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
)

func NewWithConnectionLog(ctx context.Context, wg *sync.WaitGroup, defaults configuration.Config) (config configuration.Config, brokerUrlForClients string, err error) {
	config, brokerUrlForClients, err = New(ctx, wg, defaults)
	if err != nil {
		return
	}
	config.SubscriptionDbConStr, err = docker.Postgres(ctx, wg, "subscriptions")

	return config, brokerUrlForClients, err
}

func New(ctx context.Context, wg *sync.WaitGroup, defaults configuration.Config) (config configuration.Config, brokerUrlForClients string, err error) {
	config = defaults
	config.KafkaPartitionNum = 1
	config.KafkaReplicationFactor = 1

	err = auth.Mock(&config, ctx)
	if err != nil {
		return config, "", err
	}

	whPort, err := getFreePort()
	if err != nil {
		log.Println("unable to find free port", err)
		return config, "", err
	}
	config.WebhookPort = strconv.Itoa(whPort)

	_, zk, err := docker.Zookeeper(ctx, wg)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, "", err
	}
	zkUrl := zk + ":2181"

	config.KafkaUrl, err = docker.Kafka(ctx, wg, zkUrl)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, "", err
	}

	config.DeviceManagerUrl, config.DeviceRepoUrl, err = iot.Mock(ctx, config)
	if err != nil {
		return config, "", err
	}

	_, memcacheIp, err := docker.Memcached(ctx, wg)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, "", err
	}

	config.IotCacheUrl = []string{memcacheIp + ":11211"}
	config.TokenCacheUrl = []string{memcacheIp + ":11211"}

	provider, err := testcontainers.NewDockerProvider(testcontainers.DefaultNetwork("bridge"))
	if err != nil {
		return config, "", err
	}
	hostIp, err := provider.GetGatewayIP(ctx)
	if err != nil {
		return config, "", err
	}

	config.MqttBroker, brokerUrlForClients, err = docker.Vernemqtt(ctx, wg, hostIp+":"+config.WebhookPort, config.MqttAuthMethod == "certificate", config.MqttVersion)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, "", err
	}

	config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresPw, config.PostgresDb, err = docker.Timescale(ctx, wg)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, "", err
	}

	//transform local-address to address in docker container
	deviceManagerUrlStruct := strings.Split(config.DeviceManagerUrl, ":")
	deviceManagerUrl := "http://" + hostIp + ":" + deviceManagerUrlStruct[len(deviceManagerUrlStruct)-1]
	log.Println("DEBUG: semantic url transformation:", config.DeviceManagerUrl, "-->", deviceManagerUrl)

	err = docker.Tableworker(ctx, wg, config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresPw, config.PostgresDb, config.KafkaUrl, deviceManagerUrl)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, "", err
	}

	return config, brokerUrlForClients, nil
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
