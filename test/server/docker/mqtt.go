package docker

import (
	"context"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/ory/dockertest/v3"
	uuid "github.com/satori/go.uuid"
	"log"
	"strings"
	"sync"
)

func Vernemqtt(pool *dockertest.Pool, ctx context.Context, wg *sync.WaitGroup, connecorUrl string) (brokerUrl string, err error) {
	log.Println("start mqtt")
	container, err := pool.Run("erlio/docker-vernemq", "1.9.1-alpine", []string{
		"DOCKER_VERNEMQ_LOG__CONSOLE__LEVEL=debug",
		"DOCKER_VERNEMQ_SHARED_SUBSCRIPTION_POLICY=random",
		"DOCKER_VERNEMQ_PLUGINS__VMQ_PASSWD=off",
		"DOCKER_VERNEMQ_PLUGINS__VMQ_ACL=off",
		"DOCKER_VERNEMQ_PLUGINS__VMQ_WEBHOOKS=on",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLSUBSCRIBE__HOOK=auth_on_subscribe",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLSUBSCRIBE__ENDPOINT=http://" + connecorUrl + "/subscribe",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLPUBLISH__HOOK=auth_on_publish",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLPUBLISH__ENDPOINT=http://" + connecorUrl + "/publish",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLREG__HOOK=auth_on_register",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLREG__ENDPOINT=http://" + connecorUrl + "/login",

		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLGONE__HOOK=on_client_gone",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLGONE__ENDPOINT=http://" + connecorUrl + "/disconnect",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLOFF__HOOK=on_client_offline",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLOFF__ENDPOINT=http://" + connecorUrl + "/disconnect",

		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLUNSUBSCR__HOOK=on_unsubscribe",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLUNSUBSCR__ENDPOINT=http://" + connecorUrl + "/unsubscribe",

		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLWAKE__HOOK=on_client_wakeup",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLWAKE__ENDPOINT=http://" + connecorUrl + "/online",
	})
	if err != nil {
		return "", err
	}
	go Dockerlog(pool, ctx, container, "VERNEMQ")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
		wg.Done()
	}()
	err = pool.Retry(func() error {
		log.Println("DEBUG: try to connection to broker")
		options := paho.NewClientOptions().
			SetAutoReconnect(false).
			SetCleanSession(false).
			SetClientID(uuid.NewV4().String()).
			AddBroker("tcp://" + container.Container.NetworkSettings.IPAddress + ":1883")

		client := paho.NewClient(options)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			if strings.Contains(token.Error().Error(), "Not Authorized") {
				return nil
			}
			log.Println("Error on Mqtt.Connect(): ", token.Error())
			return token.Error()
		}
		defer client.Disconnect(0)
		return nil
	})
	return "tcp://" + container.Container.NetworkSettings.IPAddress + ":1883", err
}
