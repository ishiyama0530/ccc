package installer_test

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallScriptInstallsLatestRelease(t *testing.T) {
	t.Parallel()

	archiveName := "ccc_linux_amd64.tar.gz"
	version := "v1.2.3"
	binaryBody := "#!/bin/sh\necho installed\n"

	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, archiveName)
	writeArchive(t, archivePath, "ccc", []byte(binaryBody))

	archiveBytes, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	sum := sha256.Sum256(archiveBytes)
	checksums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), archiveName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/ccc/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"tag_name":%q}`, version)
		case "/download/" + version + "/" + archiveName:
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(archiveBytes)
		case "/download/" + version + "/checksums.txt":
			_, _ = w.Write([]byte(checksums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	installDir := filepath.Join(tmpDir, "bin")
	cmd := exec.Command("bash", installScriptPath(t))
	cmd.Env = append(os.Environ(),
		"HOME="+tmpDir,
		"CCC_INSTALL_DIR="+installDir,
		"CCC_INSTALL_OS=linux",
		"CCC_INSTALL_ARCH=amd64",
		"CCC_INSTALL_GITHUB_API_BASE="+server.URL+"/repos/test/ccc/releases",
		"CCC_INSTALL_GITHUB_DOWNLOAD_BASE="+server.URL+"/download",
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	installedPath := filepath.Join(installDir, "ccc")
	body, err := os.ReadFile(installedPath)
	require.NoError(t, err)
	require.Equal(t, binaryBody, string(body))

	info, err := os.Stat(installedPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), info.Mode().Perm())

	text := string(output)
	require.Contains(t, text, "Installing ccc "+version)
	require.Contains(t, text, installedPath)
}

func installScriptPath(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "install.sh"))
}

func writeArchive(t *testing.T, path string, name string, body []byte) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	header := &tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(body)),
	}
	require.NoError(t, tw.WriteHeader(header))
	_, err = tw.Write(body)
	require.NoError(t, err)
}

func TestInstallScriptRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()

	archiveName := "ccc_linux_amd64.tar.gz"
	version := "v9.9.9"

	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, archiveName)
	writeArchive(t, archivePath, "ccc", []byte("#!/bin/sh\necho broken\n"))

	archiveBytes, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/download/" + version + "/" + archiveName:
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(archiveBytes)
		case "/download/" + version + "/checksums.txt":
			_, _ = w.Write([]byte("deadbeef  " + archiveName + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	installDir := filepath.Join(tmpDir, "bin")
	cmd := exec.Command("bash", installScriptPath(t))
	cmd.Env = append(os.Environ(),
		"HOME="+tmpDir,
		"CCC_INSTALL_VERSION="+version,
		"CCC_INSTALL_DIR="+installDir,
		"CCC_INSTALL_OS=linux",
		"CCC_INSTALL_ARCH=amd64",
		"CCC_INSTALL_GITHUB_DOWNLOAD_BASE="+server.URL+"/download",
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "checksum")
	_, err = os.Stat(filepath.Join(installDir, "ccc"))
	require.True(t, os.IsNotExist(err))
}

func TestInstallScriptPathExists(t *testing.T) {
	t.Parallel()

	path := installScriptPath(t)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.False(t, info.IsDir())
	require.True(t, strings.HasSuffix(path, "/install.sh"))
}
