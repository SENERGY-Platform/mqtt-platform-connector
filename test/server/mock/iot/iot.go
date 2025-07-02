package iot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"sync"
)

func Mock(ctx context.Context, config configuration.Config) (deviceManagerUrl string, deviceRepoUrl string, err error) {
	kafkaProducer, err := kafka.PrepareProducer(ctx, config.KafkaUrl, false, false, 1, 1, true)
	if err != nil {
		return "", "", err
	}
	router, err := getRouter(&Controller{
		mux:              sync.Mutex{},
		devices:          map[string]model.Device{},
		devicesByLocalId: map[string]string{},
		deviceTypes:      map[string]model.DeviceType{},
		hubs:             map[string]model.Hub{},
		protocols:        map[string]model.Protocol{},
		kafkaProducer:    kafkaProducer,
	})
	if err != nil {
		return "", "", err
	}

	server := &httptest.Server{
		Config: &http.Server{Handler: NewLogger(router, "DEBUG")},
	}
	server.Listener, err = net.Listen("tcp", ":")
	if err != nil {
		return "", "", err
	}
	server.Start()
	go func() {
		<-ctx.Done()
		server.Close()
	}()
	return server.URL, server.URL, nil
}

type MockProducer struct {
}

func (this MockProducer) Produce(topic string, message string) (err error) {
	return nil
}

func (this MockProducer) ProduceWithKey(topic string, message string, key string) (err error) {
	return nil
}

func (this MockProducer) Log(logger *log.Logger) {}

func (this MockProducer) Close() {}

func MockWithoutKafka(ctx context.Context) (deviceManagerUrl string, deviceRepoUrl string, err error) {
	router, err := getRouter(&Controller{
		mux:              sync.Mutex{},
		devices:          map[string]model.Device{},
		devicesByLocalId: map[string]string{},
		deviceTypes:      map[string]model.DeviceType{},
		hubs:             map[string]model.Hub{},
		protocols:        map[string]model.Protocol{},
		kafkaProducer:    MockProducer{},
	})
	if err != nil {
		return "", "", err
	}

	server := &httptest.Server{
		Config: &http.Server{Handler: NewLogger(router, "DEBUG")},
	}
	server.Listener, err = net.Listen("tcp", ":")
	if err != nil {
		return "", "", err
	}
	server.Start()
	go func() {
		<-ctx.Done()
		server.Close()
	}()
	return server.URL, server.URL, nil
}

func getRouter(controller *Controller) (router *httprouter.Router, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			log.Printf("%s: %s", r, debug.Stack())
			err = errors.New(fmt.Sprint("Recovered Error: ", r))
		}
	}()
	router = httprouter.New()
	DevicesEndpoints(controller, router)
	LocalDevicesEndpoints(controller, router)
	DeviceTypesEndpoints(controller, router)
	HubsEndpoints(controller, router)
	ProtocolsEndpoints(controller, router)
	return
}

type Controller struct {
	mux              sync.Mutex
	devices          map[string]model.Device
	devicesByLocalId map[string]string
	deviceTypes      map[string]model.DeviceType
	hubs             map[string]model.Hub
	protocols        map[string]model.Protocol
	kafkaProducer    kafka.ProducerInterface
}

type command string

const (
	putCommand    command = "PUT"
	deleteCommand command = "DELETE"
)

type deviceCommand struct {
	Command command      `json:"command"`
	Id      string       `json:"id"`
	Owner   string       `json:"owner"`
	Device  model.Device `json:"device"`
}

func (this *Controller) ReadDevice(id string) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	var exists bool
	result, exists = this.devices[id]
	if exists {
		return result, nil, 200
	} else {
		return nil, errors.New("404"), 404
	}
}

func (this *Controller) PublishDeviceCreate(device model.Device) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.devices[device.Id] = device
	this.devicesByLocalId[device.LocalId] = device.Id
	msg, err := json.Marshal(&deviceCommand{
		Command: putCommand,
		Id:      device.Id,
		Owner:   "someone",
		Device:  device,
	})
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	err = this.kafkaProducer.ProduceWithKey("devices", string(msg), device.Id)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	return device, nil, 200
}

func (this *Controller) PublishDeviceUpdate(id string, device model.Device) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.devices[id] = device
	this.devicesByLocalId[id] = device.LocalId
	msg, err := json.Marshal(&deviceCommand{
		Command: putCommand,
		Id:      device.Id,
		Owner:   "someone",
		Device:  device,
	})
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	err = this.kafkaProducer.ProduceWithKey("devices", string(msg), device.Id)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	return device, nil, 200
}

func (this *Controller) PublishDeviceDelete(id string) (err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	delete(this.devices, id)
	delete(this.devicesByLocalId, id)
	msg, err := json.Marshal(&deviceCommand{
		Command: deleteCommand,
		Id:      id,
		Owner:   "someone",
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	err = this.kafkaProducer.ProduceWithKey("devices", string(msg), id)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, 200
}

func (this *Controller) ReadHub(id string) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	var exists bool
	result, exists = this.hubs[id]
	if exists {
		return result, nil, 200
	} else {
		return nil, errors.New("404"), 404
	}
}

func (this *Controller) PublishHubCreate(hub model.Hub) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	hub.Id = uuid.NewString()
	this.hubs[hub.Id] = hub
	return hub, nil, 200
}

func (this *Controller) PublishHubUpdate(id string, hub model.Hub) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.hubs[id] = hub
	return hub, nil, 200
}

func (this *Controller) PublishHubDelete(id string) (err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	delete(this.hubs, id)
	return nil, 200
}

func (this *Controller) ReadDeviceType(id string) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	var exists bool
	result, exists = this.deviceTypes[id]
	if exists {
		return result, nil, 200
	} else {
		return nil, errors.New("404"), 404
	}
}

func (this *Controller) PublishDeviceTypeCreate(devicetype model.DeviceType) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	devicetype.Id = uuid.NewString()
	for i, service := range devicetype.Services {
		devicetype.Services[i] = service
	}
	this.deviceTypes[devicetype.Id] = devicetype
	return devicetype, nil, 200
}

func (this *Controller) PublishDeviceTypeUpdate(id string, devicetype model.DeviceType) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.deviceTypes[id] = devicetype
	return devicetype, nil, 200
}

func (this *Controller) PublishDeviceTypeDelete(id string) (err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	delete(this.deviceTypes, id)
	return nil, 200
}

func (this *Controller) DeviceLocalIdToId(id string) (result string, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	var exists bool
	result, exists = this.devicesByLocalId[id]
	if exists {
		return result, nil, 200
	} else {
		return "", errors.New("404"), 404
	}
}

func (this *Controller) ReadProtocol(id string) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	var exists bool
	result, exists = this.protocols[id]
	if exists {
		return result, nil, 200
	} else {
		return nil, errors.New("404"), 404
	}
}

func (this *Controller) PublishProtocolCreate(protocol model.Protocol) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	protocol.Id = uuid.NewString()
	this.protocols[protocol.Id] = protocol
	return protocol, nil, 200
}

func (this *Controller) PublishProtocolUpdate(id string, protocol model.Protocol) (result interface{}, err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.protocols[id] = protocol
	return protocol, nil, 200
}

func (this *Controller) PublishProtocolDelete(id string) (err error, code int) {
	this.mux.Lock()
	defer this.mux.Unlock()
	delete(this.protocols, id)
	return nil, 200
}
