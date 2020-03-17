package docker

import (
	"context"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/ory/dockertest"
	"log"
)

func Memcached(pool *dockertest.Pool, ctx context.Context) (hostPort string, ipAddress string, err error) {
	log.Println("start memcached")
	mem, err := pool.Run("memcached", "1.5.12-alpine", []string{})
	if err != nil {
		return "", "", err
	}
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + mem.Container.Name)
		mem.Close()
	}()
	hostPort = mem.GetPort("11211/tcp")
	err = pool.Retry(func() error {
		log.Println("try memcache connection...")
		_, err := memcache.New(mem.Container.NetworkSettings.IPAddress + ":11211").Get("foo")
		if err == memcache.ErrCacheMiss {
			return nil
		}
		if err != nil {
			log.Println(err)
		}
		return err
	})
	return hostPort, mem.Container.NetworkSettings.IPAddress, err
}
