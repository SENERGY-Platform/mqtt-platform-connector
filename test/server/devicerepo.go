package server

import (
	"context"
	"github.com/ory/dockertest"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"net/http"
	"time"
)

func DeviceRepo(pool *dockertest.Pool, mongoIp string, zk string, permsearchUrl string) (closer func(), hostPort string, ipAddress string, err error) {
	log.Println("start device repo")
	repo, err := pool.Run("fgseitsrancher.wifa.intern.uni-leipzig.de:5000/device-repository", "test", []string{
		"MONGO_URL=" + "mongodb://" + mongoIp + ":27017",
		"ZOOKEEPER_URL=" + zk,
		"PERMISSIONS_URL=" + permsearchUrl,
		"MONGO_REPL_SET=false",
	})
	if err != nil {
		return func() {}, "", "", err
	}
	hostPort = repo.GetPort("8080/tcp")
	err = pool.Retry(func() error {
		log.Println("try repo connection...")
		_, err := http.Get("http://" + repo.Container.NetworkSettings.IPAddress + ":8080/")
		if err != nil {
			log.Println(err)
		}
		return err
	})
	return func() { repo.Close() }, hostPort, repo.Container.NetworkSettings.IPAddress, err
}

func MongoTestServer(pool *dockertest.Pool) (closer func(), hostPort string, ipAddress string, err error) {
	log.Println("start mongodb")
	repo, err := pool.Run("mongo", "4.1.11", []string{})
	if err != nil {
		return func() {}, "", "", err
	}
	hostPort = repo.GetPort("27017/tcp")
	err = pool.Retry(func() error {
		log.Println("try mongodb connection...")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:"+hostPort))
		err = client.Ping(ctx, readpref.Primary())
		return err
	})
	return func() { repo.Close() }, hostPort, repo.Container.NetworkSettings.IPAddress, err
}
