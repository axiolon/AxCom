// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package events

import "time"

// defaultRetryDelays are the backoff durations used when RetryDelays is not configured.
var defaultRetryDelays = []time.Duration{5 * time.Second, 30 * time.Second, 120 * time.Second}

// BusConfig represents the consolidated configuration structure needed for the event bus.
type BusConfig struct {
	Provider string         `json:"provider"   yaml:"provider"`
	Retry    RetryConfig    `json:"retry"      yaml:"retry"`
	Local    LocalBusConfig `json:"local"      yaml:"local"`
	Kafka    KafkaConfig    `json:"kafka"      yaml:"kafka"`
	RabbitMQ RabbitMQConfig `json:"rabbitmq"   yaml:"rabbitmq"`
}

// RetryConfig holds parameters for the message retry mechanism.
type RetryConfig struct {
	MaxRetries     int           `json:"max_retries"     yaml:"max_retries"`
	InitialBackoff time.Duration `json:"initial_backoff" yaml:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"     yaml:"max_backoff"`
}

// LocalBusConfig holds parameters for the in-process event bus.
type LocalBusConfig struct {
	DLQBufferSize int `json:"dlq_buffer_size" yaml:"dlq_buffer_size"`
}

// KafkaConfig holds configurations for a Kafka-backed event bus.
type KafkaConfig struct {
	Brokers       []string `json:"brokers"          yaml:"brokers"`
	GroupID       string   `json:"group_id"         yaml:"group_id"`
	ClientID      string   `json:"client_id"        yaml:"client_id"`
	DLQTopic      string   `json:"dlq_topic"        yaml:"dlq_topic"`
	AutoOffsetOld bool     `json:"auto_offset_old"  yaml:"auto_offset_old"` // true maps to auto.offset.reset="earliest"
}

// RabbitMQConfig holds configurations for a RabbitMQ-backed event bus.
type RabbitMQConfig struct {
	URL          string `json:"url"           yaml:"url"`
	ExchangeName string `json:"exchange_name" yaml:"exchange_name"`
	ExchangeType string `json:"exchange_type" yaml:"exchange_type"`
	QueueName    string `json:"queue_name"    yaml:"queue_name"`
	DLQExchange  string `json:"dlq_exchange"  yaml:"dlq_exchange"`
	DLQQueue     string `json:"dlq_queue"     yaml:"dlq_queue"`
	// RetryDelays defines the backoff duration for each retry attempt.
	// The length determines the maximum number of retries.
	// Defaults to [5s, 30s, 120s] if not set.
	RetryDelays   []time.Duration `json:"retry_delays"   yaml:"retry_delays"`
	PrefetchCount int             `json:"prefetch_count" yaml:"prefetch_count"`
}
