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

const SqlCreateSubscriptionTable = `CREATE TABLE IF NOT EXISTS ClientDeviceSubscription (
	Client		VARCHAR(255) NOT NULL,
	Topic 		VARCHAR(255) NOT NULL,
	Device		VARCHAR(255) NOT NULL,
	Inactive    BOOLEAN DEFAULT FALSE,
	PRIMARY KEY (Client, Topic)
);
CREATE INDEX IF NOT EXISTS client_index ON ClientDeviceSubscription (Client);
CREATE INDEX IF NOT EXISTS device_index ON ClientDeviceSubscription (Device);`

const SqlCreateCleanSessionTable = `CREATE TABLE IF NOT EXISTS CleanSession (
	Client		VARCHAR(255) NOT NULL,
	CleanSession BOOLEAN,
	PRIMARY KEY (Client)
);`

const SqlStoreSubscription = `INSERT INTO ClientDeviceSubscription(Client, Topic, Device) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;`

const SqlDeleteSubscription = `DELETE FROM ClientDeviceSubscription WHERE Client = $1 AND Topic = $2;`

const SqlDeleteClient = `DELETE FROM ClientDeviceSubscription WHERE Client = $1;`

const SqlSelectDeviceByClient = `SELECT DISTINCT Device FROM ClientDeviceSubscription WHERE Client = $1;`

const SqlCheckDeviceSubscriptionExists = `SELECT COUNT(1) FROM ClientDeviceSubscription WHERE Device = $1 AND Inactive = FALSE LIMIT 1;`

const SqlAnyDeviceSubscription = `SELECT DISTINCT Device FROM ClientDeviceSubscription WHERE Device = any($1) AND Inactive = FALSE LIMIT $2`

const SqlCleanSessionUpsert = `INSERT INTO CleanSession(Client, CleanSession) VALUES ($1, $2) ON CONFLICT (Client) DO UPDATE SET CleanSession = $2;`

const SqlQueryCleanSession = `SELECT CleanSession FROM CleanSession WHERE Client = $1 LIMIT 1;`

const SqlSetClientInactive = `UPDATE ClientDeviceSubscription SET Inactive = $2 WHERE Client = $1;`
