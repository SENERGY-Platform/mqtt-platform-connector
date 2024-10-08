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

package topic

import (
	"errors"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/shortid"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"regexp"
	"sort"
	"strings"
)

var ErrNoDeviceMatchFound = errors.New("no device match found")
var ErrNoDeviceIdCandidateFound = errors.New("no device id candidate found")
var ErrNoServiceMatchFound = errors.New("no service match found")
var ErrMultipleMatchingDevicesFound = errors.New("multiple matching devices found")

// if no service is fount, ErrNoServiceMatchFound will be returned as error. but the device will be set if one was found
func (this *Topic) Parse(token security.JwtToken, topic string) (device model.Device, service model.Service, err error) {
	candidates, err := this.ParseForCandidates(token, topic)
	if err != nil {
		return device, service, err
	}
	if len(candidates) == 0 {
		return device, service, ErrNoDeviceMatchFound
	}
	if len(candidates) > 1 {
		return device, service, ErrMultipleMatchingDevicesFound
	}
	device = candidates[0].device
	if len(candidates[0].services) == 0 {
		return device, service, ErrNoServiceMatchFound
	}
	service = candidates[0].services[0]
	return device, service, nil
}

// not exported, no one should care
type candidate struct {
	device   model.Device
	services []model.Service
}

func (this *Topic) ParseForCandidates(token security.JwtToken, topic string) (candidates []candidate, err error) {
	devices, err := this.findDeviceCandidates(token, topic)
	if err != nil {
		return candidates, err
	}
	for _, device := range devices {
		services, err := this.findMatchingServices(token, device.DeviceTypeId, topic)
		if err != nil {
			return candidates, err
		}
		//longest matches first
		sort.Slice(services, func(i, j int) bool {
			return len(services[i].LocalId) > len(services[j].LocalId)
		})
		candidates = append(candidates, candidate{
			device:   device,
			services: services,
		})
	}

	//first sort by service count and then by service.LocalId to prefer devices with the longest service match and the most matching services if 2 have the same length of localID
	//this allows devices as result without matching services but prefers devices with matches

	//sort by service count
	sort.SliceStable(candidates, func(i, j int) bool {
		lenI := len(candidates[i].services)
		lenJ := len(candidates[j].services)
		return lenI > lenJ
	})
	//sort by service with longest local id
	//expects service sorting in findMatchingServices()
	sort.SliceStable(candidates, func(i, j int) bool {
		lenI := 0
		lenJ := 0
		if len(candidates[i].services) > 0 {
			lenI = len(candidates[i].services[0].LocalId)
		}
		if len(candidates[j].services) > 0 {
			lenJ = len(candidates[j].services[0].LocalId)
		}
		return lenI > lenJ
	})
	return
}

func (this *Topic) findMatchingServices(token security.JwtToken, deviceTypeId string, topic string) (services []model.Service, err error) {
	deviceType, err := this.iotCache.WithToken(token).GetDeviceType(deviceTypeId)
	if err != nil {
		return services, err
	}
	for _, service := range deviceType.Services {
		if this.serviceMatchesTopic(topic, service) {
			services = append(services, service)
		}
	}
	return services, nil
}

func (this *Topic) serviceMatchesTopic(topic string, service model.Service) bool {
	return strings.Contains(topic, service.LocalId)
}

func (this *Topic) findDeviceCandidates(token security.JwtToken, topic string) (candidates []model.Device, err error) {
	candidateIds, err := this.findDeviceIdCandidates(topic)
	if err != nil {
		return candidates, err
	}
	if len(candidateIds) == 0 {
		return candidates, ErrNoDeviceIdCandidateFound
	}
	for _, id := range candidateIds {
		device, err := this.iotCache.WithToken(token).GetDevice(id)
		if err == nil {
			candidates = append(candidates, device)
		} else {
			if !errors.Is(err, security.ErrorNotFound) && !errors.Is(err, security.ErrorAccessDenied) {
				return candidates, err
			}
		}
	}
	return candidates, nil
}

func (this *Topic) findDeviceIdCandidates(topic string) (candidates []string, err error) {
	candidates = findDeviceIdCandidates(topic)
	for _, shortCandidate := range findShortDeviceIdCandidates(topic) {
		candidate, err := shortid.EnsureLongDeviceId(shortCandidate)
		if err == nil {
			candidates = append(candidates, candidate)
		}
	}
	return candidates, nil
}

func findDeviceIdCandidates(topic string) (candidates []string) {
	for _, part := range strings.Split(topic, "/") {
		if regexp.MustCompile(`^urn:infai:ses:device:[\w-]*$`).MatchString(part) {
			candidates = append(candidates, part)
		}
	}
	return candidates
}

func findShortDeviceIdCandidates(topic string) (candidates []string) {
	for _, part := range strings.Split(topic, "/") {
		if regexp.MustCompile(`^[\w\-_]{22}$`).MatchString(part) {
			candidates = append(candidates, part)
		}
	}
	return candidates
}
