package server

import (
	"github.com/ory/dockertest"
	"log"
	"strings"
	"time"
)

func Vernemqtt(pool *dockertest.Pool, connecorUrl string) (closer func(), hostPort string, ipAddress string, err error) {
	log.Println("start vernemqtt")
	log.Println(strings.Join([]string{
		"DOCKER_VERNEMQ_LOG__CONSOLE__LEVEL=debug",
		"DOCKER_VERNEMQ_SHARED_SUBSCRIPTION_POLICY=random",
		"DOCKER_VERNEMQ_PLUGINS__VMQ_WEBHOOKS=on",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLSUBSCRIBE__HOOK=auth_on_subscribe",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLSUBSCRIBE__ENDPOINT=http://" + connecorUrl + "/subscribe",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLPUBLISH__HOOK=auth_on_publish",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLPUBLISH__ENDPOINT=http://" + connecorUrl + "/publish",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLREG__HOOK=auth_on_register",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLREG__ENDPOINT=http://" + connecorUrl + "/login",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLOFF__HOOK=on_client_offline",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLOFF__ENDPOINT=http://" + connecorUrl + "/disconnect",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLUNSUBSCR__HOOK=on_unsubscribe",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLUNSUBSCR__ENDPOINT=http://" + connecorUrl + "/unsubscribe",
		"DOCKER_VERNEMQ_PLUGINS__VMQ_PASSWD=off",
		"DOCKER_VERNEMQ_PLUGINS__VMQ_ACL=off",
	}, "\n"))
	repo, err := pool.Run("erlio/docker-vernemq", "latest", []string{
		"DOCKER_VERNEMQ_LOG__CONSOLE__LEVEL=debug",
		"DOCKER_VERNEMQ_SHARED_SUBSCRIPTION_POLICY=random",
		"DOCKER_VERNEMQ_PLUGINS__VMQ_WEBHOOKS=on",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLSUBSCRIBE__HOOK=auth_on_subscribe",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLSUBSCRIBE__ENDPOINT=http://" + connecorUrl + "/subscribe",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLPUBLISH__HOOK=auth_on_publish",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLPUBLISH__ENDPOINT=http://" + connecorUrl + "/publish",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLREG__HOOK=auth_on_register",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLREG__ENDPOINT=http://" + connecorUrl + "/login",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLOFF__HOOK=on_client_offline",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLOFF__ENDPOINT=http://" + connecorUrl + "/disconnect",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLUNSUBSCR__HOOK=on_unsubscribe",
		"DOCKER_VERNEMQ_VMQ_WEBHOOKS__SEPLUNSUBSCR__ENDPOINT=http://" + connecorUrl + "/unsubscribe",
		"DOCKER_VERNEMQ_PLUGINS__VMQ_PASSWD=off",
		"DOCKER_VERNEMQ_PLUGINS__VMQ_ACL=off",
	})
	if err != nil {
		return func() {}, "", "", err
	}
	hostPort = repo.GetPort("1883/tcp")
	time.Sleep(2 * time.Second)
	return func() { repo.Close() }, hostPort, repo.Container.NetworkSettings.IPAddress, err
}
