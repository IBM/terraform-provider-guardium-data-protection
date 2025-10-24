// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package gdp

import "encoding/json"

// ConfigureDatasourcePayloadBuilder implements the builder pattern for ConfigureDatasourcePayload
type ConfigureDatasourcePayloadBuilder struct {
	payload *ConfigureDatasourcePayload
}

// VAConfigPayload represents the JSON payload for configuring VA for a datasource
type ConfigureDatasourcePayload struct {
	DatasourceName string     `json:"datasource_name"`
	Schedule       VASchedule `json:"schedule"`
	Enabled        bool       `json:"enabled"`
}

// VASchedule represents the schedule configuration for VA
type VASchedule struct {
	Frequency string `json:"frequency"`
	Day       string `json:"day"`
	Time      string `json:"time"`
}

// NewConfigureDatasourcePayloadBuilder creates a new builder for ConfigureDatasourcePayload
func NewConfigureDatasourcePayloadBuilder() *ConfigureDatasourcePayloadBuilder {
	return &ConfigureDatasourcePayloadBuilder{
		payload: &ConfigureDatasourcePayload{
			Enabled: false,
		},
	}
}

// DatasourceName sets the name of the datasource
func (b *ConfigureDatasourcePayloadBuilder) DatasourceName(name string) *ConfigureDatasourcePayloadBuilder {
	b.payload.DatasourceName = name
	return b
}

// Enabled sets the name of the datasource
func (b *ConfigureDatasourcePayloadBuilder) Enabled(enabled bool) *ConfigureDatasourcePayloadBuilder {
	b.payload.Enabled = enabled
	return b
}

// Frequency sets the name of the datasource
func (b *ConfigureDatasourcePayloadBuilder) Frequency(frequency string) *ConfigureDatasourcePayloadBuilder {
	b.payload.Schedule.Frequency = frequency
	return b
}

// Day sets the name of the datasource
func (b *ConfigureDatasourcePayloadBuilder) Day(day string) *ConfigureDatasourcePayloadBuilder {
	b.payload.Schedule.Day = day
	return b
}

// Time sets the name of the datasource
func (b *ConfigureDatasourcePayloadBuilder) Time(time string) *ConfigureDatasourcePayloadBuilder {
	b.payload.Schedule.Time = time
	return b
}

// Build returns the constructed RegisterDatasourcePayload
func (b *ConfigureDatasourcePayloadBuilder) Build() ([]byte, error) {
	return json.Marshal(b.payload)
}
