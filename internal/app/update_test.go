package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitHubReleaseUpdateNotifierNoticeReturnsMessageWhenNewVersionExists(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "application/vnd.github+json", request.Header.Get("Accept"))
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"tag_name":"v1.2.0"}`))
	}))
	t.Cleanup(server.Close)

	notifier := GitHubReleaseUpdateNotifier{Client: server.Client(), APIURL: server.URL}

	message, err := notifier.Notice(context.Background(), "v1.1.0")
	require.NoError(t, err)
	require.Equal(t, "アップデート可能です: v1.1.0 -> v1.2.0", message)
}

func TestGitHubReleaseUpdateNotifierNoticeReturnsEmptyWhenSameVersion(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"tag_name":"v1.2.0"}`))
	}))
	t.Cleanup(server.Close)

	notifier := GitHubReleaseUpdateNotifier{Client: server.Client(), APIURL: server.URL}

	message, err := notifier.Notice(context.Background(), "v1.2.0")
	require.NoError(t, err)
	require.Empty(t, message)
}

func TestGitHubReleaseUpdateNotifierNoticeSkipsDevVersion(t *testing.T) {
	t.Parallel()

	notifier := GitHubReleaseUpdateNotifier{}

	message, err := notifier.Notice(context.Background(), "dev")
	require.NoError(t, err)
	require.Empty(t, message)
}
