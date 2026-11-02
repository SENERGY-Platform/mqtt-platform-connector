/*
 * Copyright 2026 InfAI (CC SES)
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
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

func TestHandleEnvironmentVarsHidesSecrets(t *testing.T) {
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	t.Setenv("SUBSCRIPTION_DB_CON_STR", "postgres://user:hunter2@db/x")
	t.Setenv("WEBHOOK_PORT", "8080")

	config := &Config{}
	handleEnvironmentVars(config)

	if config.SubscriptionDbConStr != "postgres://user:hunter2@db/x" {
		t.Fatalf("secret not applied: %q", config.SubscriptionDbConStr)
	}
	if strings.Contains(buf.String(), "hunter2") {
		t.Fatalf("secret logged: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "WEBHOOK_PORT") {
		t.Fatalf("non-secret not logged: %s", buf.String())
	}
}

func TestHandleEnvironmentVarsSetsKafkaTopicSettings(t *testing.T) {
	t.Setenv("KAFKA_REPLICATION_FACTOR", "2")
	t.Setenv("KAFKA_TOPIC_CONFIGS", `{"urn_infai_ses_service_":[{"ConfigName":"retention.ms","ConfigValue":"604800000"}]}`)

	config := &Config{KafkaReplicationFactor: 1}
	handleEnvironmentVars(config)

	if config.KafkaReplicationFactor != 2 {
		t.Fatalf("replication factor not applied: %d", config.KafkaReplicationFactor)
	}
	entries := config.KafkaTopicConfigs["urn_infai_ses_service_"]
	if len(entries) != 1 || entries[0].ConfigName != "retention.ms" || entries[0].ConfigValue != "604800000" {
		t.Fatalf("topic configs not applied: %#v", config.KafkaTopicConfigs)
	}
}
