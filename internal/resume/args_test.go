package resume

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseExtraArgsParsesQuotedArguments(t *testing.T) {
	t.Parallel()

	args, err := ParseExtraArgs(`--model sonnet --system "hello world"`)
	require.NoError(t, err)
	require.Equal(t, []string{"--model", "sonnet", "--system", "hello world"}, args)
}

func TestParseExtraArgsRejectsUnterminatedQuotes(t *testing.T) {
	t.Parallel()

	_, err := ParseExtraArgs(`--system "hello world`)
	require.Error(t, err)
}

func TestValidateExtraArgsRejectsResume(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"resume"},
		{"--resume"},
		{"--resume=other"},
	} {
		err := ValidateExtraArgs(args)
		require.Error(t, err)
	}
}
