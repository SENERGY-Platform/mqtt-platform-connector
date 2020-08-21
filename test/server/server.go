package server

import (
	"context"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/docker"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/auth"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server/mock/iot"
	"github.com/ory/dockertest"
	"log"
	"net"
	"runtime/debug"
	"strconv"
)

func New(basectx context.Context, defaults lib.Config) (config lib.Config, err error) {
	config = defaults

	ctx, cancel := context.WithCancel(basectx)

	err = auth.Mock(&config, ctx)
	if err != nil {
		cancel()
		return config, err
	}

	config.DeviceManagerUrl, config.DeviceRepoUrl, err = iot.Mock(ctx)
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
	config.ZookeeperUrl = zk + ":2181"

	err = docker.Kafka(pool, ctx, config.ZookeeperUrl)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	_, memcacheIp, err := docker.Memcached(pool, ctx)
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
	config.MqttBroker, err = docker.Vernemqtt(pool, ctx, hostIp+":"+config.WebhookPort)
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
