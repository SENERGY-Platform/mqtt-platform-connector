package server

import (
	"context"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"log"
	"net"
	"os"
	"runtime/debug"
	"strconv"
)

func New(parentCtx context.Context, defaults lib.Config) (config lib.Config, err error) {
	config = defaults
	ctx, cancel := context.WithCancel(parentCtx)

	config.WebhookPort, err = getFreePortStr()

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	_, zk, err := Zookeeper(pool, ctx)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	config.ZookeeperUrl = zk + ":2181"

	err = Kafka(pool, ctx, config.ZookeeperUrl)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	_, elasticIp, err := Elasticsearch(pool, ctx)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	_, mongoIp, err := MongoTestServer(pool, ctx)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	_, permIp, err := PermSearch(pool, ctx, config.ZookeeperUrl, elasticIp)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	permissionUrl := "http://" + permIp + ":8080"

	_, deviceRepoIp, err := DeviceRepo(pool, ctx, mongoIp, config.ZookeeperUrl, permissionUrl)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	config.DeviceRepoUrl = "http://" + deviceRepoIp + ":8080"

	_, deviceManagerIp, err := DeviceManager(pool, ctx, config.ZookeeperUrl, config.DeviceRepoUrl, "-", permissionUrl)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	config.DeviceManagerUrl = "http://" + deviceManagerIp + ":8080"

	_, cacheIp, err := Memcached(pool, ctx)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	config.IotCacheUrl = []string{cacheIp + ":11211"}
	config.TokenCacheUrl = []string{cacheIp + ":11211"}

	_, keycloakIp, err := Keycloak(pool, ctx)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	config.AuthEndpoint = "http://" + keycloakIp + ":8080"
	config.AuthClientSecret = "d61daec4-40d6-4d3e-98c9-f3b515696fc6"
	config.AuthClientId = "connector"

	err = ConfigKeycloak(config.AuthEndpoint, "connector")
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}

	network, err := pool.Client.NetworkInfo("bridge")
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	hostIp := network.IPAM.Config[0].Gateway
	_, ip, err := Vernemqtt(pool, ctx, hostIp+":"+config.WebhookPort)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		cancel()
		return config, err
	}
	config.MqttBroker = "tcp://" + ip + ":1883"

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

func Dockerlog(pool *dockertest.Pool, ctx context.Context, repo *dockertest.Resource, name string) {
	out := &LogWriter{logger: log.New(os.Stdout, "["+name+"]", 0)}
	err := pool.Client.Logs(docker.LogsOptions{
		Stdout:       true,
		Stderr:       true,
		Context:      ctx,
		Container:    repo.Container.ID,
		Follow:       true,
		OutputStream: out,
		ErrorStream:  out,
	})
	if err != nil && err != context.Canceled {
		log.Println("DEBUG-ERROR: unable to start docker log", name, err)
	}
}

type LogWriter struct {
	logger *log.Logger
}

func (this *LogWriter) Write(p []byte) (n int, err error) {
	this.logger.Print(string(p))
	return len(p), nil
}
