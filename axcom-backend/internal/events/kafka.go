// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package events

import "errors"

type KafkaEventBus struct {
	cfg KafkaConfig //nolint:unused
}

//nolint:unused
func newKafkaEventBus(_ KafkaConfig) (EventBus, error) {
	return nil, errors.New("kafka event bus is not yet implemented")
}

func (b *KafkaEventBus) Subscribe(_ string, _ EventHandler) {
	panic("KafkaEventBus.Subscribe is not implemented")
}

func (b *KafkaEventBus) Publish(_ Event) {
	panic("KafkaEventBus.Publish is not implemented")
}

func (b *KafkaEventBus) Close() error {
	return nil
}
