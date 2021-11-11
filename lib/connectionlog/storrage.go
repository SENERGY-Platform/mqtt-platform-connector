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

package connectionlog

import (
	"context"
	"database/sql"
	"github.com/lib/pq"
	"time"
)

var Timeout = 1 * time.Second

func initDbConnection(conStr string) (db *sql.DB, err error) {
	db, err = sql.Open("postgres", conStr)
	if err != nil {
		return
	}
	_, err = db.Exec(SqlCreateSubscriptionTable)
	if err != nil {
		return db, err
	}
	_, err = db.Exec(SqlCreateCleanSessionTable)
	if err != nil {
		return db, err
	}
	return db, err
}

func (this *ConnectionLogImpl) storeSubscription(client string, topic string, id string) (err error) {
	_, err = this.db.Exec(SqlStoreSubscription, client, topic, id)
	return
}

func (this *ConnectionLogImpl) removeSubscription(client string, topic string) (err error) {
	_, err = this.db.Exec(SqlDeleteSubscription, client, topic)
	return
}

func (this *ConnectionLogImpl) removeClient(client string) (err error) {
	_, err = this.db.Exec(SqlDeleteClient, client)
	return
}

func (this *ConnectionLogImpl) loadSubscriptions(client string) (devices []string, err error) {
	ctx, _ := context.WithTimeout(context.Background(), Timeout)
	resp, err := this.db.QueryContext(ctx, SqlSelectDeviceByClient, client)
	if err != nil {
		return devices, err
	}
	for resp.Next() {
		var device string
		err = resp.Scan(&device)
		if err != nil {
			return devices, err
		}
		devices = append(devices, device)
	}
	return devices, err
}

func (this *ConnectionLogImpl) noDeviceSubscriptionStored(device string) (result bool, err error) {
	ctx, _ := context.WithTimeout(context.Background(), Timeout)
	row := this.db.QueryRowContext(ctx, SqlCheckDeviceSubscriptionExists, device)
	err = row.Err()
	if err != nil {
		return result, err
	}
	count := 0
	err = row.Scan(&count)
	if err != nil {
		return result, err
	}
	result = count == 0
	return
}

//returns devices that are in input but not subscribed to
func (this *ConnectionLogImpl) filterByStoredDevices(devices []string) (result []string, err error) {
	subscribed := map[string]bool{}
	ctx, _ := context.WithTimeout(context.Background(), Timeout)
	rows, err := this.db.QueryContext(ctx, SqlAnyDeviceSubscription, pq.Array(devices), len(devices))
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var device string
		err = rows.Scan(&device)
		if err != nil {
			return devices, err
		}
		subscribed[device] = true
	}
	for _, device := range devices {
		if !subscribed[device] {
			result = append(result, device)
		}
	}
	return result, nil
}

func (this *ConnectionLogImpl) setCleanSession(client string, clean bool) (err error) {
	_, err = this.db.Exec(SqlCleanSessionUpsert, client, clean)
	return
}

func (this *ConnectionLogImpl) isCleanSession(client string) (isCleanSession bool, err error) {
	ctx, _ := context.WithTimeout(context.Background(), Timeout)
	row := this.db.QueryRowContext(ctx, SqlQueryCleanSession, client)
	err = row.Err()
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return isCleanSession, err
	}
	err = row.Scan(&isCleanSession)
	if err == sql.ErrNoRows {
		return true, nil
	}
	return isCleanSession, err
}

func (this *ConnectionLogImpl) setClientInactive(client string, inactive bool) (err error) {
	_, err = this.db.Exec(SqlSetClientInactive, client, inactive)
	return
}
