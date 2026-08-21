package docker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SENERGY-Platform/service-commons/pkg/testing/docker"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Timescale(ctx context.Context, wg *sync.WaitGroup) (host string, port int, user string, pw string, db string, err error) {
	log.Println("start timescale")
	pw = "postgrespw"
	user = "postgres"
	db = "postgres"
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "timescale/timescaledb:2.3.0-pg13",
			Tmpfs:        map[string]string{},
			ExposedPorts: []string{"5432/tcp"},
			//the tag is immutable, so a cached image is the same image; pulling it
			//once per test run instead of once per test keeps the registry out of
			//the failure modes
			AlwaysPullImage: false,
			//postgres opens the port during its init phase and restarts before it
			//is actually usable, so waiting for the port alone hands out a
			//connection string that is not ready yet
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5432/tcp"),
				wait.ForNop(docker.Waitretry(time.Minute, func(ctx context.Context, target wait.StrategyTarget) error {
					p, err := target.MappedPort(ctx, "5432/tcp")
					if err != nil {
						log.Println(err)
						return err
					}
					return tryPostgresConn(fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", user, pw, p.Port(), db))
				})),
			),
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

	//the connector reaches the container by ip, not through the mapped port, so
	//that is the connection that has to work before the test starts
	err = docker.Retry(time.Minute, func() error {
		return tryPostgresConn(fmt.Sprintf("postgres://%s:%s@%s:%v/%s?sslmode=disable", user, pw, host, port, db))
	})

	return host, port, user, pw, db, err
}
