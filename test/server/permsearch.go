/*
 * Copyright 2019 InfAI (CC SES)
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

package server

import (
	"context"
	"github.com/ory/dockertest"
	"log"
	"net/http"
)

func Elasticsearch(pool *dockertest.Pool, ctx context.Context) (hostPort string, ipAddress string, err error) {
	log.Println("start elasticsearch")
	repo, err := pool.Run("docker.elastic.co/elasticsearch/elasticsearch", "6.4.3", []string{"discovery.type=single-node"})
	if err != nil {
		return "", "", err
	}
	go func() {
		<-ctx.Done()
		repo.Close()
	}()
	hostPort = repo.GetPort("9200/tcp")
	err = pool.Retry(func() error {
		log.Println("try elastic connection...")
		_, err := http.Get("http://" + repo.Container.NetworkSettings.IPAddress + ":9200/_cluster/health")
		if err != nil {
			log.Println(err)
		}
		return err
	})
	return hostPort, repo.Container.NetworkSettings.IPAddress, err
}

func PermSearch(pool *dockertest.Pool, ctx context.Context, zk string, elasticIp string) (hostPort string, ipAddress string, err error) {
	log.Println("start permsearch")
	container, err := pool.Run("fgseitsrancher.wifa.intern.uni-leipzig.de:5000/permission-search", "dev", []string{
		"ZOOKEEPER_URL=" + zk,
		"ELASTIC_URL=" + "http://" + elasticIp + ":9200",
	})
	if err != nil {
		return "", "", err
	}
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
	}()
	hostPort = container.GetPort("8080/tcp")
	err = pool.Retry(func() error {
		log.Println("try permsearch connection...")
		_, err := http.Get("http://" + container.Container.NetworkSettings.IPAddress + ":8080/jwt/check/devices/foo/r/bool")
		if err != nil {
			log.Println(err)
		}
		return err
	})
	return hostPort, container.Container.NetworkSettings.IPAddress, err
}
