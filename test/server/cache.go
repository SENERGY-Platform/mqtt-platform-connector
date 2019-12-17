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
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/ory/dockertest"
	"log"
)

func Memcached(pool *dockertest.Pool, ctx context.Context) (hostPort string, ipAddress string, err error) {
	log.Println("start memcached")
	container, err := pool.Run("memcached", "1.5.12-alpine", []string{})
	if err != nil {
		return "", "", err
	}
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
	}()
	hostPort = container.GetPort("11211/tcp")
	err = pool.Retry(func() error {
		log.Println("try memcache connection...")
		_, err := memcache.New(container.Container.NetworkSettings.IPAddress + ":11211").Get("foo")
		if err == memcache.ErrCacheMiss {
			return nil
		}
		if err != nil {
			log.Println(err)
		}
		return err
	})
	return hostPort, container.Container.NetworkSettings.IPAddress, err
}
