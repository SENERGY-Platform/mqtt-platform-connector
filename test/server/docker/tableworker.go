package docker

import (
	"context"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"log"
	"strconv"
	"sync"
)

func Tableworker(pool *dockertest.Pool, ctx context.Context, wg *sync.WaitGroup, postgresHost string, postgresPort int, postgresUser string, postgresPw string, postgresDb string, kafkaBootstrap string, deviceManagerUrl string) (err error) {
	log.Println("start Tableworker")
	container, err := pool.Run("ghcr.io/senergy-platform/timescale-tableworker", "dev", []string{
		"POSTGRES_PW=" + postgresPw,
		"POSTGRES_HOST=" + postgresHost,
		"POSTGRES_DB=" + postgresDb,
		"POSTGRES_USER=" + postgresUser,
		"POSTGRES_PORT=" + strconv.Itoa(postgresPort),
		"KAFKA_BOOTSTRAP=" + kafkaBootstrap,
		"DEVICE_MANAGER_URL=" + deviceManagerUrl,
		"USE_DISTRIBUTED_HYPERTABLES=false",
		"DEBUG=true",
	},
	)
	if err != nil {
		return err
	}

	go Dockerlog(pool, ctx, container, "TABLEWORKER")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
		wg.Done()
	}()
	return
}
