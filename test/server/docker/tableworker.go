package docker

import (
	"context"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"log"
	"strconv"
	"sync"
)

func Tableworker(ctx context.Context, wg *sync.WaitGroup, postgresHost string, postgresPort int, postgresUser string, postgresPw string, postgresDb string, kafkaBootstrap string, deviceManagerUrl string) (err error) {
	log.Println("start tableworker")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:           "ghcr.io/senergy-platform/timescale-tableworker:dev",
			AlwaysPullImage: true,
			Env: map[string]string{
				"POSTGRES_PW":                 postgresPw,
				"POSTGRES_HOST":               postgresHost,
				"POSTGRES_DB":                 postgresDb,
				"POSTGRES_USER":               postgresUser,
				"POSTGRES_PORT":               strconv.Itoa(postgresPort),
				"KAFKA_BOOTSTRAP":             kafkaBootstrap,
				"DEVICE_MANAGER_URL":          deviceManagerUrl,
				"USE_DISTRIBUTED_HYPERTABLES": "false",
				"DEBUG":                       "true",
			},
		},
		Started: true,
	})
	if err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container tableworker", c.Terminate(context.Background()))
	}()

	/*
		err = Dockerlog(ctx, c, "TABLEWORKER")
		if err != nil {
			return "", "", err
		}
	*/

	return err
}
