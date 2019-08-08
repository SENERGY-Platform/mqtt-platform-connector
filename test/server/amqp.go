package server

import (
	"github.com/ory/dockertest"
	"github.com/streadway/amqp"
	"log"
)

func Amqp(pool *dockertest.Pool) (closer func(), hostPort string, ipAddress string, err error) {
	log.Println("start rabbitmq")
	rabbitmq, err := pool.Run("rabbitmq", "3-management", []string{})
	if err != nil {
		return func() {}, "", "", err
	}
	hostPort = rabbitmq.GetPort("5672/tcp")
	err = pool.Retry(func() error {
		log.Println("try amqp connection...")
		conn, err := amqp.Dial("amqp://guest:guest@" + rabbitmq.Container.NetworkSettings.IPAddress + ":5672/")
		if err != nil {
			return err
		}
		defer conn.Close()
		c, err := conn.Channel()
		defer c.Close()
		return err
	})
	return func() { rabbitmq.Close() }, hostPort, rabbitmq.Container.NetworkSettings.IPAddress, err
}
