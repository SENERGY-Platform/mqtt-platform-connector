/*
 * Copyright 2020 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package docker

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"sync"
	"time"
)

func Postgres(ctx context.Context, wg *sync.WaitGroup, dbname string) (conStr string, err error) {
	conStr, _, _, err = PostgresWithNetwork(ctx, wg, dbname)
	return
}

func PostgresWithNetwork(ctx context.Context, wg *sync.WaitGroup, dbname string) (conStr string, containerIp string, hostPort string, err error) {
	log.Println("start mqtt")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:11.2",
			ExposedPorts: []string{"5432/tcp"},
			Tmpfs:        map[string]string{"/var/lib/postgresql/data": "rw"},
			Env: map[string]string{
				"POSTGRES_DB":       dbname,
				"POSTGRES_PASSWORD": "pw",
				"POSTGRES_USER":     "usr",
			},
			WaitingFor: wait.ForAll(
				wait.ForLog("server started"),
				wait.ForListeningPort("5432/tcp"),
				wait.ForNop(waitretry(time.Minute, func(ctx context.Context, target wait.StrategyTarget) error {
					p, err := target.MappedPort(ctx, "5432/tcp")
					if err != nil {
						log.Println(err)
						return err
					}
					tryconstr := fmt.Sprintf("postgres://usr:pw@%s:%s/%s?sslmode=disable", "localhost", p.Port(), dbname)
					return tryPostgresConn(tryconstr)
				})),
			),
		},
		Started: true,
	})
	if err != nil {
		return "", containerIp, hostPort, err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container mongo", c.Terminate(context.Background()))
	}()

	containerIp, err = c.ContainerIP(ctx)
	if err != nil {
		return "", containerIp, hostPort, err
	}
	temp, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return "", containerIp, hostPort, err
	}
	hostPort = temp.Port()
	conStr = fmt.Sprintf("postgres://usr:pw@%s:%s/%s?sslmode=disable", containerIp, "5432", dbname)
	err = retry(time.Minute, func() error {
		return tryPostgresConn(conStr)
	})
	if err != nil {
		return conStr, containerIp, hostPort, err
	}
	return conStr, containerIp, hostPort, nil
}

func tryPostgresConn(connstr string) error {
	log.Println("try postgres conn to", connstr)
	db, err := sql.Open("postgres", connstr)
	if err != nil {
		log.Println(err)
		return err
	}
	err = db.Ping()
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
