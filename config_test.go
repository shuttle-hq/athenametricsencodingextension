// Copyright Shuttle
// SPDX-License-Identifier: Apache-2.0

package athenametricsencodingextension

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DefaultConfig(t *testing.T) {
	c := createDefaultConfig().(*Config)
	require.NoError(t, c.Validate())
}
