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

package vernemqtt

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
)

type connectionLogMock struct{}

func (connectionLogMock) Connect(string)                     {}
func (connectionLogMock) Disconnect(string)                  {}
func (connectionLogMock) Subscribe(string, string, string)   {}
func (connectionLogMock) Unsubscribe(string, string, string) {}
func (connectionLogMock) SetCleanSession(string, bool)       {}

func TestLoginDoesNotLogPassword(t *testing.T) {
	const password = "test-secret-pw-4711"

	for name, status := range map[string]int{
		"auth answers 401": http.StatusUnauthorized,
		"auth answers 500": http.StatusInternalServerError,
	} {
		t.Run(name, func(t *testing.T) {
			auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			}))
			defer auth.Close()

			config := configuration.Config{
				MqttAuthMethod:    "password",
				AuthClientId:      "auth-client",
				SecRemoteProtocol: "mqtt",
			}
			connector, err := platform_connector_lib.New(platform_connector_lib.Config{
				AuthEndpoint: auth.URL,
				AuthClientId: config.AuthClientId,
			})
			if err != nil {
				t.Fatal(err)
			}

			logs := &bytes.Buffer{}
			logger := slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

			body := `{"peer_addr":"10.0.0.7","peer_port":41310,"mountpoint":"","client_id":"client-1","username":"alice","password":"` + password + `","clean_session":true}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
			rec := httptest.NewRecorder()
			login(rec, req, config, connector, connectionLogMock{}, logger)

			out := logs.String()
			if !strings.Contains(out, "unable to get user token") {
				t.Fatalf("failing login path not reached, log: %s", out)
			}
			if strings.Contains(out, password) {
				t.Errorf("password found in log output: %s", strings.ReplaceAll(out, password, "<PASSWORD>"))
			}
			for _, field := range []string{"alice", "client-1", "10.0.0.7"} {
				if !strings.Contains(out, field) {
					t.Errorf("expected %q in log output: %s", field, out)
				}
			}
		})
	}
}
