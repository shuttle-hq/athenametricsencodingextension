// Copyright Shuttle
// SPDX-License-Identifier: Apache-2.0

package athenametricsencodingextension // import "github.com/shuttle-hq/athenametricsencodingextension"

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestExtension_Start(t *testing.T) {
	tests := []struct {
		name         string
		getExtension func() (extension.Extension, error)
		expectedErr  string
	}{
		{
			name: "text",
			getExtension: func() (extension.Extension, error) {
				factory := NewFactory()
				return factory.Create(context.Background(), extensiontest.NewNopSettings(), factory.CreateDefaultConfig())
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ext, err := test.getExtension()
			if test.expectedErr != "" && err != nil {
				require.ErrorContains(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
			}
			err = ext.Start(context.Background(), componenttest.NewNopHost())
			if test.expectedErr != "" && err != nil {
				require.ErrorContains(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMarshalMetrics(t *testing.T) {
	tests := []struct {
		name          string
		createMetrics func() pmetric.Metrics
		expectedJSON  string
		expectedErr   string
	}{
		{
			name:          "sampleMetrics",
			createMetrics: sampleMetrics,
			expectedJSON:  `{"aws.ecs.cluster.name":"production-user-cluster-fargate-1","aws.ecs.task.id":"4b8146f69b564bebadf89b47b904325b","ecs.task.cpu.utilized":9.833394491064633,"ecs.task.memory.utilized":22,"timestamp":1734959017836}`,
		},
		{
			name:          "sampleMetricsNoDataPoints",
			createMetrics: sampleMetricsNoDataPoints,
			expectedJSON:  `{"aws.ecs.cluster.name":"production-user-cluster-fargate-1","aws.ecs.task.id":"4b8146f69b564bebadf89b47b904325b","timestamp":1736440793840}`,
		},
		{
			name:          "sampleMetricsMultipleDataPoints",
			createMetrics: sampleMetricsMultipleDataPoints,
			expectedJSON:  `{"aws.ecs.cluster.name":"production-user-cluster-fargate-1","aws.ecs.task.id":"4b8146f69b564bebadf89b47b904325b","ecs.task.memory.utilized":24,"timestamp":1734959017936}`,
		},
		{
			name:          "sampleMetricsNoResourceAttributes",
			createMetrics: sampleMetricsNoResourceAttributes,
			expectedJSON:  `{"ecs.task.cpu.utilized":9.833394491064633,"timestamp":1734959017836}`,
		},
		{
			name:          "sampleMetricsWithTimestampResourceAttributes",
			createMetrics: sampleMetricsWithTimestampResourceAttributes,
			expectedJSON:  `{"ecs.task.cpu.utilized":9.833394491064633,"timestamp":1734959017736}`,
		},
		{
			name:          "sampleMetricsNoScopeMetrics",
			createMetrics: sampleMetricsNoScopeMetrics,
			expectedJSON:  `{"aws.ecs.cluster.name":"production-user-cluster-fargate-1","aws.ecs.task.id":"4b8146f69b564bebadf89b47b904325b","timestamp":1736440793840}`,
		},
		{
			name:          "sampleMultipleResourceMetrics",
			createMetrics: sampleMultipleResourceMetrics,
			expectedJSON: `{"aws.ecs.cluster.name":"production-user-cluster-fargate-1","aws.ecs.task.id":"4b8146f69b564bebadf89b47b904325b","ecs.task.cpu.utilized":9.833394491064633,"ecs.task.memory.utilized":22,"timestamp":1734959017836}
{"aws.ecs.cluster.name":"production-user-cluster-fargate-1","aws.ecs.task.id":"fa83a50c8a2b29b15492f1667e194b74","ecs.task.cpu.utilized":207.4615648942136,"ecs.task.memory.utilized":87,"timestamp":1734959017836}`,
		},
	}

	j := &jsonMetricsExtension{
		config: &Config{},
	}

	// Override the current time with fixed time
	now = func() time.Time { return time.UnixMilli(1736440793840) }

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lp, err := j.MarshalMetrics(test.createMetrics())
			if test.expectedErr != "" && err != nil {
				assert.ErrorContains(t, err, test.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, lp)
				// Compare raw strings instead of using assert.JSONeq, because the output is JSON lines
				assert.Equal(t, test.expectedJSON, string(lp))
			}
		})
	}
}

// Test the happy path
func sampleMetrics() pmetric.Metrics {
	timestamp := time.Unix(1734959017, 836172421)
	pm := pmetric.NewMetrics()
	rm := pm.ResourceMetrics().AppendEmpty()
	a := rm.Resource().Attributes()
	a.PutStr("aws.ecs.cluster.name", "production-user-cluster-fargate-1")
	a.PutStr("aws.ecs.task.id", "4b8146f69b564bebadf89b47b904325b")
	sm := rm.ScopeMetrics().AppendEmpty()
	{
		m := sm.Metrics().AppendEmpty()
		m.SetName("ecs.task.memory.utilized")
		m.SetUnit("Megabytes")
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetIntValue(22)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
	}
	{
		m := sm.Metrics().AppendEmpty()
		m.SetName("ecs.task.cpu.utilized")
		m.SetUnit("None")
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(9.833394491064633)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
	}
	return pm
}

// Test case when a metric has no data points
func sampleMetricsNoDataPoints() pmetric.Metrics {
	pm := pmetric.NewMetrics()
	rm := pm.ResourceMetrics().AppendEmpty()
	a := rm.Resource().Attributes()
	a.PutStr("aws.ecs.cluster.name", "production-user-cluster-fargate-1")
	a.PutStr("aws.ecs.task.id", "4b8146f69b564bebadf89b47b904325b")
	sm := rm.ScopeMetrics().AppendEmpty()
	{
		m := sm.Metrics().AppendEmpty()
		m.SetName("ecs.task.memory.utilized")
		m.SetUnit("Megabytes")
	}
	return pm
}

// Test case when a metric has multiple data points with different timestamps
func sampleMetricsMultipleDataPoints() pmetric.Metrics {
	timestamp := time.Unix(1734959017, 836172421)
	pm := pmetric.NewMetrics()
	rm := pm.ResourceMetrics().AppendEmpty()
	a := rm.Resource().Attributes()
	a.PutStr("aws.ecs.cluster.name", "production-user-cluster-fargate-1")
	a.PutStr("aws.ecs.task.id", "4b8146f69b564bebadf89b47b904325b")
	sm := rm.ScopeMetrics().AppendEmpty()
	{
		m := sm.Metrics().AppendEmpty()
		m.SetName("ecs.task.memory.utilized")
		m.SetUnit("Megabytes")
		gauge := m.SetEmptyGauge()
		{
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetIntValue(22)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
		}
		{
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetIntValue(24)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp.Add(time.Millisecond * 100)))
		}
		{
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetIntValue(23)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp.Add(time.Millisecond * 50)))
		}
	}
	return pm
}

// Test case when a resource does not have any attributes
func sampleMetricsNoResourceAttributes() pmetric.Metrics {
	timestamp := time.Unix(1734959017, 836172421)
	pm := pmetric.NewMetrics()
	rm := pm.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	{
		m := sm.Metrics().AppendEmpty()
		m.SetName("ecs.task.cpu.utilized")
		m.SetUnit("None")
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(9.833394491064633)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
	}
	return pm
}

// Test case when a resource has a timestamp attributes
func sampleMetricsWithTimestampResourceAttributes() pmetric.Metrics {
	timestamp := time.Unix(1734959017, 836172421)
	pm := pmetric.NewMetrics()
	rm := pm.ResourceMetrics().AppendEmpty()
	a := rm.Resource().Attributes()
	a.PutInt("timestamp", timestamp.Add(-time.Millisecond*100).UnixMilli())
	sm := rm.ScopeMetrics().AppendEmpty()
	{
		m := sm.Metrics().AppendEmpty()
		m.SetName("ecs.task.cpu.utilized")
		m.SetUnit("None")
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(9.833394491064633)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
	}
	return pm
}

// Test case when there are no scope metrics
func sampleMetricsNoScopeMetrics() pmetric.Metrics {
	pm := pmetric.NewMetrics()
	rm := pm.ResourceMetrics().AppendEmpty()
	a := rm.Resource().Attributes()
	a.PutStr("aws.ecs.cluster.name", "production-user-cluster-fargate-1")
	a.PutStr("aws.ecs.task.id", "4b8146f69b564bebadf89b47b904325b")
	return pm
}

// Test a batch with multiple resource metrics
func sampleMultipleResourceMetrics() pmetric.Metrics {
	timestamp := time.Unix(1734959017, 836172421)
	pm := pmetric.NewMetrics()
	{
		rm := pm.ResourceMetrics().AppendEmpty()
		a := rm.Resource().Attributes()
		a.PutStr("aws.ecs.cluster.name", "production-user-cluster-fargate-1")
		a.PutStr("aws.ecs.task.id", "4b8146f69b564bebadf89b47b904325b")
		sm := rm.ScopeMetrics().AppendEmpty()
		{
			m := sm.Metrics().AppendEmpty()
			m.SetName("ecs.task.memory.utilized")
			m.SetUnit("Megabytes")
			dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
			dp.SetIntValue(22)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
		}
		{
			m := sm.Metrics().AppendEmpty()
			m.SetName("ecs.task.cpu.utilized")
			m.SetUnit("None")
			dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
			dp.SetDoubleValue(9.833394491064633)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
		}
	}
	{
		rm := pm.ResourceMetrics().AppendEmpty()
		a := rm.Resource().Attributes()
		a.PutStr("aws.ecs.cluster.name", "production-user-cluster-fargate-1")
		a.PutStr("aws.ecs.task.id", "fa83a50c8a2b29b15492f1667e194b74")
		sm := rm.ScopeMetrics().AppendEmpty()
		{
			m := sm.Metrics().AppendEmpty()
			m.SetName("ecs.task.memory.utilized")
			m.SetUnit("Megabytes")
			dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
			dp.SetIntValue(87)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
		}
		{
			m := sm.Metrics().AppendEmpty()
			m.SetName("ecs.task.cpu.utilized")
			m.SetUnit("None")
			dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
			dp.SetDoubleValue(207.4615648942136)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
		}
	}
	return pm
}
