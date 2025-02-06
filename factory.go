// Copyright Shuttle
// SPDX-License-Identifier: Apache-2.0

package athenametricsencodingextension // import "github.com/shuttle-hq/athenametricsencodingextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/shuttle-hq/athenametricsencodingextension/internal/metadata"
)

func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		createExtension,
		metadata.ExtensionStability,
	)
}

func createExtension(_ context.Context, _ extension.Settings, config component.Config) (extension.Extension, error) {
	return &jsonMetricsExtension{
		config: config.(*Config),
	}, nil
}

func createDefaultConfig() component.Config {
	return &Config{}
}
