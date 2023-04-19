package docker

import (
	"context"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"sync"
)

func Timescale(ctx context.Context, wg *sync.WaitGroup) (host string, port int, user string, pw string, db string, err error) {
	log.Println("start timescale")
	pw = "postgrespw"
	user = "postgres"
	db = "postgres"
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:           "timescale/timescaledb:2.3.0-pg13",
			Tmpfs:           map[string]string{},
			WaitingFor:      wait.ForListeningPort("5432/tcp"),
			ExposedPorts:    []string{"5432/tcp"},
			AlwaysPullImage: true,
			Env: map[string]string{
				"POSTGRES_PASSWORD": pw,
			},
		},
		Started: true,
	})
	if err != nil {
		return host, port, user, pw, db, err
	}
	host, err = c.ContainerIP(ctx)
	if err != nil {
		return host, port, user, pw, db, err
	}
	port = 5432
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container timescale", c.Terminate(context.Background()))
	}()

	/*
		err = Dockerlog(ctx, c, "TIMESCALE")
		if err != nil {
			return "", "", err
		}
	*/

	return host, port, user, pw, db, err
}
