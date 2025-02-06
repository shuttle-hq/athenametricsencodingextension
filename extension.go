// Copyright Shuttle
// SPDX-License-Identifier: Apache-2.0

package athenametricsencodingextension // import "github.com/shuttle-hq/athenametricsencodingextension"

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var _ encoding.MetricsMarshalerExtension = (*jsonMetricsExtension)(nil)

// Can be overridden in unit tests
var now = time.Now

type jsonMetricsExtension struct {
	config *Config
}

func (e *jsonMetricsExtension) MarshalMetrics(md pmetric.Metrics) ([]byte, error) {
	var b []byte
	err := error(nil)

	for i := range md.ResourceMetrics().Len() {
		rm := md.ResourceMetrics().At(i)
		data, e := json.Marshal(flattenResourceMetrics(rm))
		if e != nil {
			err = e
			continue
		}
		if i > 0 {
			b = append(b, 0x0A) // Add line separator \n
		}
		b = append(b, data...)
	}

	return b, err
}

func (e *jsonMetricsExtension) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (e *jsonMetricsExtension) Shutdown(_ context.Context) error {
	return nil
}

func flattenResourceMetrics(rm pmetric.ResourceMetrics) map[string]any {
	timestamp := time.UnixMilli(0)

	// Add all resource attributes at the same level as metrics
	metrics := rm.Resource().Attributes().AsRaw()

	// Flatten the scope metrics, take the latest (by time) datapoint from each set of data points
	// Only gauges and sums are supported
	for i := range rm.ScopeMetrics().Len() {
		sm := rm.ScopeMetrics().At(i)
		for j := range sm.Metrics().Len() {
			m := sm.Metrics().At(j)
			name := m.Name()
			dps := pmetric.NewNumberDataPointSlice()
			switch m.Type() {
			case pmetric.MetricTypeGauge:
				dps = m.Gauge().DataPoints()
			case pmetric.MetricTypeSum:
				dps = m.Sum().DataPoints()
			default:
				// Do nothing
			}

			latest := time.UnixMilli(0)
			for k := range dps.Len() {
				dp := dps.At(k)

				t := dp.Timestamp().AsTime()
				if t.Before(latest) {
					continue // Skip this data point in favor of a more recent one
				}
				latest = t

				if t.After(timestamp) {
					timestamp = t // Update the timestamp for all metrics with the latest data point timestamp
				}

				switch dp.ValueType() {
				case pmetric.NumberDataPointValueTypeInt:
					metrics[name] = dp.IntValue()
				case pmetric.NumberDataPointValueTypeDouble:
					metrics[name] = dp.DoubleValue()
				default:
					// Do nothing
				}
			}
		}
	}

	// Fallback: set timestamp to the current time, if no timestamp could be extracted from the data points
	if timestamp.UnixMilli() == 0 {
		timestamp = now()
	}

	// Add the timestamp unless it was already present in resource metrics
	if _, hasTimestamp := metrics["timestamp"]; !hasTimestamp {
		metrics["timestamp"] = timestamp.UnixMilli()
	}

	return metrics
}
