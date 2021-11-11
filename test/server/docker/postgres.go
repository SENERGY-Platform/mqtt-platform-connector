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
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"log"
	"sync"
)

func Postgres(ctx context.Context, wg *sync.WaitGroup, dbname string) (conStr string, ip string, port string, err error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return "", ip, port, err
	}
	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "11.2",
		Env:        []string{"POSTGRES_DB=" + dbname, "POSTGRES_PASSWORD=pw", "POSTGRES_USER=usr"},
	}, func(config *docker.HostConfig) {
		config.Tmpfs = map[string]string{"/var/lib/postgresql/data": "rw"}
	})
	if err != nil {
		return "", ip, port, err
	}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
		wg.Done()
	}()
	ip = container.Container.NetworkSettings.IPAddress
	port = "5432"
	conStr = fmt.Sprintf("postgres://usr:pw@%s:%s/%s?sslmode=disable", ip, port, dbname)
	err = pool.Retry(func() error {
		var err error
		log.Println("try connecting to pg")
		db, err := sql.Open("postgres", conStr)
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
	})
	return
}

