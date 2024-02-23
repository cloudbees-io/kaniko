package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_UnknownArguments(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	t.Run("hanging boolean value", func(t *testing.T) {
		os.Args = []string{"kaniko-action", "--skip-default-registry-fallback", "false"}
		err := cmd.Execute()
		require.Error(t, err, "boolean flag without =")
		require.Contains(t, err.Error(), "unknown arguments: [false]", "boolean flag error message")
	})

	t.Run("unknown flag", func(t *testing.T) {
		os.Args = []string{"asdf", "--not-an-arg"}
		err := cmd.Execute()
		require.Error(t, err, "not an argument")
		require.Contains(t, err.Error(), "unknown flag: --not-an-arg", "unknown flag")
	})

}
