package ch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gohopper/core/config"
)

// TestPreparationHandler_Enabled ports Java CHPreparationHandlerTest.testEnabled.
func TestPreparationHandler_Enabled(t *testing.T) {
	h := NewPreparationHandler()
	assert.False(t, h.IsEnabled())
	h.SetCHProfiles(config.CHProfile{Profile: "myconfig"})
	assert.True(t, h.IsEnabled())
}
