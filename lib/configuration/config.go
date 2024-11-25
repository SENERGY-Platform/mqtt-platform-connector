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

package configuration

import (
	"encoding/json"
	"flag"
	"github.com/segmentio/kafka-go"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	KafkaUrl           string `json:"kafka_url"`
	KafkaResponseTopic string `json:"kafka_response_topic"`
	KafkaGroupName     string `json:"kafka_group_name"`
	FatalKafkaError    bool   `json:"fatal_kafka_error"`
	Protocol           string `json:"protocol"`

	DeviceManagerUrl string `json:"device_manager_url"`
	DeviceRepoUrl    string `json:"device_repo_url"`

	AuthClientId             string  `json:"auth_client_id" config:"secret"`
	AuthClientSecret         string  `json:"auth_client_secret" config:"secret"`
	AuthExpirationTimeBuffer float64 `json:"auth_expiration_time_buffer"`
	AuthEndpoint             string  `json:"auth_endpoint"`

	JwtPrivateKey string `json:"jwt_private_key" config:"secret"`
	JwtExpiration int64  `json:"jwt_expiration"`
	JwtIssuer     string `json:"jwt_issuer" config:"secret"`

	DeviceExpiration     int32    `json:"device_expiration"`
	DeviceTypeExpiration int32    `json:"device_type_expiration"`
	TokenCacheExpiration int32    `json:"token_cache_expiration"`
	IotCacheUrl          []string `json:"iot_cache_url"`
	TokenCacheUrl        []string `json:"token_cache_url" config:"secret"`
	SyncKafka            bool     `json:"sync_kafka"`
	SyncKafkaIdempotent  bool     `json:"sync_kafka_idempotent"`
	Debug                bool     `json:"debug"`

	MqttBroker     string `json:"mqtt_broker"`
	MqttClientId   string `json:"mqtt_client_id"`
	Qos            byte   `json:"qos"`
	MqttLogLevel   string `json:"mqtt_log_level"`
	MqttVersion    string `json:"mqtt_version"`
	MqttAuthMethod string `json:"mqtt_auth_method"` // Whether the MQTT broker uses a username/password or client certificate authetication

	WebhookPort             string `json:"webhook_port"`
	HttpCommandConsumerPort string `json:"http_command_consumer_port"`

	ActuatorTopicPattern string `json:"actuator_topic_pattern"`

	SerializationFallback string `json:"serialization_fallback"`

	Validate                  bool `json:"validate"`
	ValidateAllowUnknownField bool `json:"validate_allow_unknown_field"`
	ValidateAllowMissingField bool `json:"validate_allow_missing_field"`

	KafkaProducerSlowTimeoutSec int64 `json:"kafka_producer_slow_timeout_sec"`

	KafkaPartitionNum      int `json:"kafka_partition_num"`
	KafkaReplicationFactor int `json:"kafka_replication_factor"`

	PublishToPostgres bool   `json:"publish_to_postgres"`
	PostgresHost      string `json:"postgres_host"`
	PostgresPort      int    `json:"postgres_port"`
	PostgresUser      string `json:"postgres_user" config:"secret"`
	PostgresPw        string `json:"postgres_pw" config:"secret"`
	PostgresDb        string `json:"postgres_db"`

	AsyncPgThreadMax    int64  `json:"async_pg_thread_max"`
	AsyncFlushMessages  int64  `json:"async_flush_messages"`
	AsyncFlushFrequency string `json:"async_flush_frequency"`
	AsyncCompression    string `json:"async_compression"`
	SyncCompression     string `json:"sync_compression"`

	KafkaConsumerMaxWait  string `json:"kafka_consumer_max_wait"`
	KafkaConsumerMinBytes int64  `json:"kafka_consumer_min_bytes"`
	KafkaConsumerMaxBytes int64  `json:"kafka_consumer_max_bytes"`

	IotCacheMaxIdleConns int64  `json:"iot_cache_max_idle_conns"`
	IotCacheTimeout      string `json:"iot_cache_timeout"`

	CommandWorkerCount int64 `json:"command_worker_count"`

	SubscriptionDbConStr string `json:"subscription_db_con_str"`
	DeviceLogTopic       string `json:"device_log_topic"`

	DeviceTypeTopic string `json:"device_type_topic"`

	NotificationUrl string `json:"notification_url"`
	PermQueryUrl    string `json:"perm_query_url"`

	NotificationsIgnoreDuplicatesWithinS int      `json:"notifications_ignore_duplicates_within_s"`
	NotificationUserOverwrite            string   `json:"notification_user_overwrite"`
	DeveloperNotificationUrl             string   `json:"developer_notification_url"`
	MutedUserNotificationTitles          []string `json:"muted_user_notification_titles"`

	SecRemoteProtocol string `json:"sec_remote_protocol"`

	KafkaTopicConfigs map[string][]kafka.ConfigEntry `json:"kafka_topic_configs"`
}

func LoadConfig() (result Config, err error) {
	return LoadConfigFlag("config")
}

func LoadConfigFlag(configLocationFlag string) (result Config, err error) {
	configLocation := flag.String(configLocationFlag, "config.json", "configuration file")
	flag.Parse()
	return Load(*configLocation)
}

func Load(location string) (result Config, err error) {
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
		fieldConfig := configType.Field(index).Tag.Get("config")
		envName := fieldNameToEnvName(fieldName)
		envValue := os.Getenv(envName)
		if envValue != "" {
			if !strings.Contains(fieldConfig, "secret") {
				log.Println("use environment variable: ", envName, " = ", envValue)
			}
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
