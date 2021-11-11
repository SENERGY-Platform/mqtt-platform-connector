package docker

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"log"
	"sync"
)

func Timescale(pool *dockertest.Pool, ctx context.Context, wg *sync.WaitGroup) (host string, port int, user string, pw string, db string, err error) {
	log.Println("start postgres")
	pw = "postgrespw"
	user = "postgres"
	db = "postgres"
	container, err := pool.Run("timescale/timescaledb", "2.3.0-pg13", []string{
		"POSTGRES_PASSWORD=" + pw,
	})
	if err != nil {
		return "", 0, "", "", "", err
	}
	go Dockerlog(pool, ctx, container, "TIMESCALE")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
		wg.Done()
	}()
	host = container.Container.NetworkSettings.IPAddress
	port = 5432
	err = pool.Retry(func() error {
		log.Println("DEBUG: try to connect to db")
		psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host,
			port, user, pw, "postgres")

		// open database
		db, err := sql.Open("postgres", psqlconn)
		if err != nil {
			return err
		}
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)
		err = db.Ping()
		if err != nil {
			log.Println("DB not ready yet. Connecting with " + psqlconn)
			return err
		}
		return nil
	})
	return
}
