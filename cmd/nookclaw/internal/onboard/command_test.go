package onboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOnboardCommand(t *testing.T) {
	cmd := NewOnboardCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "onboard", cmd.Use)
	assert.Equal(t, "Initialize NookClaw configuration and workspace", cmd.Short)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("o"))

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.Args)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	for _, flagName := range []string{
		"non-interactive",
		"advanced",
		"force",
		"launcher-public",
		"provider",
		"api-key",
		"channel",
		"channel-secret",
		"channel-app-token",
		"channel-user-id",
		"channel-homeserver",
	} {
		assert.NotNil(t, cmd.Flags().Lookup(flagName), "expected flag %q to exist", flagName)
	}

	assert.False(t, cmd.HasSubCommands())
}
