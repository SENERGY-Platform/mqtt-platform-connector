/*
 * Copyright 2018 InfAI (CC SES)
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

package lib

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	ZookeeperUrl       string `json:"zookeeper_url"`
	KafkaResponseTopic string `json:"kafka_response_topic"`
	KafkaGroupName     string `json:"kafka_group_name"`
	FatalKafkaError    bool   `json:"fatal_kafka_error"`
	Protocol           string `json:"protocol"`

	DeviceManagerUrl string `json:"device_manager_url"`
	DeviceRepoUrl    string `json:"device_repo_url"`

	AuthClientId             string  `json:"auth_client_id"`
	AuthClientSecret         string  `json:"auth_client_secret"`
	AuthExpirationTimeBuffer float64 `json:"auth_expiration_time_buffer"`
	AuthEndpoint             string  `json:"auth_endpoint"`

	JwtPrivateKey string `json:"jwt_private_key"`
	JwtExpiration int64  `json:"jwt_expiration"`
	JwtIssuer     string `json:"jwt_issuer"`

	DeviceExpiration     int32    `json:"device_expiration"`
	DeviceTypeExpiration int32    `json:"device_type_expiration"`
	TokenCacheExpiration int32    `json:"token_cache_expiration"`
	IotCacheUrl          []string `json:"iot_cache_url"`
	TokenCacheUrl        []string `json:"token_cache_url"`
	SyncKafka            bool     `json:"sync_kafka"`
	SyncKafkaIdempotent  bool     `json:"sync_kafka_idempotent"`
	Debug                bool     `json:"debug"`

	MqttBroker   string `json:"mqtt_broker"`
	MqttClientId string `json:"mqtt_client_id"`
	Qos          byte   `json:"qos"`
	MqttLogLevel string `json:"mqtt_log_level"`

	WebhookPort string `json:"webhook_port"`

	ActuatorTopicPattern string `json:"actuator_topic_pattern"`
	SensorTopicPattern   string `json:"sensor_topic_pattern"`

	SerializationFallback string `json:"serialization_fallback"`

	Validate                  bool `json:"validate"`
	ValidateAllowUnknownField bool `json:"validate_allow_unknown_field"`
	ValidateAllowMissingField bool `json:"validate_allow_missing_field"`
}

func LoadConfig() (result Config, err error) {
	return LoadConfigFlag("config")
}

func LoadConfigFlag(configLocationFlag string) (result Config, err error) {
	configLocation := flag.String(configLocationFlag, "config.json", "configuration file")
	flag.Parse()
	return LoadConfigLocation(*configLocation)
}

func LoadConfigLocation(location string) (result Config, err error) {
	file, err := os.Open(location)
	if err != nil {
		log.Println("error on config load: ", err)
		return result, err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&result)
	if err != nil {
		log.Println("invalid config json: ", err)
		return result, err
	}
	log.Println("handle environment variables")
	handleEnvironmentVars(&result)
	return
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func fieldNameToEnvName(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToUpper(strings.Join(a, "_"))
}

func handleEnvironmentVars(config interface{}) {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()
	for index := 0; index < configType.NumField(); index++ {
		fieldName := configType.Field(index).Name
		envName := fieldNameToEnvName(fieldName)
		envValue := os.Getenv(envName)
		if envValue != "" {
			log.Println("use environment variable: ", envName, " = ", envValue)
			if configValue.FieldByName(fieldName).Kind() == reflect.Int64 {
				i, _ := strconv.ParseInt(envValue, 10, 64)
				configValue.FieldByName(fieldName).SetInt(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Bool {
				b, _ := strconv.ParseBool(envValue)
				configValue.FieldByName(fieldName).SetBool(b)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Map {
				value := map[string]string{}
				for _, element := range strings.Split(envValue, ",") {
					keyVal := strings.Split(element, ":")
					key := strings.TrimSpace(keyVal[0])
					val := strings.TrimSpace(keyVal[1])
					value[key] = val
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}

		}
	}
}
