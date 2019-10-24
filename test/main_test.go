package test

import (
	"context"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	defaultConfig, err := lib.LoadConfigLocation("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.Start(ctx, config)
	if err != nil {
		t.Error(err)
		return
	}

}
