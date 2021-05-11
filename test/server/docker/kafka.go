package docker

import (
	"context"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/segmentio/kafka-go"
	"github.com/wvanbergen/kazoo-go"
	"log"
	"strconv"
	"strings"
	"time"
)

func Kafka(pool *dockertest.Pool, ctx context.Context, zookeeperUrl string) (kafkaUrl string, err error) {
	kafkaport, err := getFreePort()
	if err != nil {
		return kafkaUrl, err
	}
	networks, _ := pool.Client.ListNetworks()
	hostIp := ""
	for _, network := range networks {
		if network.Name == "bridge" {
			hostIp = network.IPAM.Config[0].Gateway
		}
	}
	kafkaUrl = hostIp + ":" + strconv.Itoa(kafkaport)
	log.Println("host ip: ", hostIp)
	log.Println("kafkaUrl url: ", kafkaUrl)
	env := []string{
		"ALLOW_PLAINTEXT_LISTENER=yes",
		"KAFKA_LISTENERS=OUTSIDE://:9092",
		"KAFKA_ADVERTISED_LISTENERS=OUTSIDE://" + kafkaUrl,
		"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=OUTSIDE:PLAINTEXT",
		"KAFKA_INTER_BROKER_LISTENER_NAME=OUTSIDE",
		"KAFKA_ZOOKEEPER_CONNECT=" + zookeeperUrl,
	}
	log.Println("start kafka with env ", env)
	container, err := pool.RunWithOptions(&dockertest.RunOptions{Repository: "bitnami/kafka", Tag: "latest", Env: env, PortBindings: map[docker.Port][]docker.PortBinding{
		"9092/tcp": {{HostIP: "", HostPort: strconv.Itoa(kafkaport)}},
	}})
	if err != nil {
		return kafkaUrl, err
	}
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
	}()
	err = pool.Retry(func() error {
		log.Println("try kafka connection...")
		conn, err := kafka.Dial("tcp", kafkaUrl)
		if err != nil {
			log.Println(err)
			return err
		}
		defer conn.Close()
		return nil
	})
	time.Sleep(5 * time.Second)
	return kafkaUrl, err
}

func Zookeeper(pool *dockertest.Pool, ctx context.Context) (hostPort string, ipAddress string, err error) {
	zkport, err := getFreePort()
	if err != nil {
		log.Fatalf("Could not find new port: %s", err)
	}
	env := []string{}
	log.Println("start zookeeper on ", zkport)
	container, err := pool.RunWithOptions(&dockertest.RunOptions{Repository: "wurstmeister/zookeeper", Tag: "latest", Env: env, PortBindings: map[docker.Port][]docker.PortBinding{
		"2181/tcp": {{HostIP: "", HostPort: strconv.Itoa(zkport)}},
	}})
	if err != nil {
		return "", "", err
	}
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
	}()
	hostPort = strconv.Itoa(zkport)
	err = pool.Retry(func() error {
		log.Println("try zk connection...")
		zookeeper := kazoo.NewConfig()
		zk, chroot := kazoo.ParseConnectionString(container.Container.NetworkSettings.IPAddress)
		zookeeper.Chroot = chroot
		kz, err := kazoo.NewKazoo(zk, zookeeper)
		if err != nil {
			log.Println("kazoo", err)
			return err
		}
		_, err = kz.Brokers()
		if err != nil && strings.TrimSpace(err.Error()) != strings.TrimSpace("zk: node does not exist") {
			log.Println("brokers", err)
			return err
		}
		return nil
	})
	return hostPort, container.Container.NetworkSettings.IPAddress, err
}
