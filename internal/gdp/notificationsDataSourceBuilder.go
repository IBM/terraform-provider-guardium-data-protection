// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package gdp

import "encoding/json"

// NotificationsPayload represents the JSON payload for configuring notifications
type ConfigureNotificationsPayload struct {
	DatasourceName   string   `json:"datasource_name"`
	NotificationType string   `json:"notification_type"`
	Recipients       []string `json:"recipients"`
	Severity         string   `json:"severity"`
	Enabled          bool     `json:"enabled"`
}

// ConfigureNotificationsPayloadBuilder implements the builder pattern for ConfigureDatasourcePayload
type ConfigureNotificationsPayloadBuilder struct {
	payload *ConfigureNotificationsPayload
}

// NewConfigureNotificationsPayloadBuilder creates a new builder for ConfigureDatasourcePayload
func NewConfigureNotificationsPayloadBuilder() *ConfigureNotificationsPayloadBuilder {
	return &ConfigureNotificationsPayloadBuilder{
		payload: &ConfigureNotificationsPayload{},
	}
}

// DatasourceName sets the name of the datasource
func (b *ConfigureNotificationsPayloadBuilder) DatasourceName(name string) *ConfigureNotificationsPayloadBuilder {
	b.payload.DatasourceName = name
	return b
}

// NotificationType sets the name of the datasource
func (b *ConfigureNotificationsPayloadBuilder) NotificationType(name string) *ConfigureNotificationsPayloadBuilder {
	b.payload.NotificationType = name
	return b
}

// Recipients sets the name of the datasource
func (b *ConfigureNotificationsPayloadBuilder) Recipients(recipients []string) *ConfigureNotificationsPayloadBuilder {
	b.payload.Recipients = recipients
	return b
}

// Severity sets the name of the datasource
func (b *ConfigureNotificationsPayloadBuilder) Severity(severity string) *ConfigureNotificationsPayloadBuilder {
	b.payload.Severity = severity
	return b
}

// Enabled sets the name of the datasource
func (b *ConfigureNotificationsPayloadBuilder) Enabled(enabled bool) *ConfigureNotificationsPayloadBuilder {
	b.payload.Enabled = enabled
	return b
}

// Build returns the constructed RegisterDatasourcePayload
func (b *ConfigureNotificationsPayloadBuilder) Build() ([]byte, error) {
	return json.Marshal(b.payload)
}
