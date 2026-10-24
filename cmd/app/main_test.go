package main

import (
	"bufio"
	"context"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainReportsStartupFailure(t *testing.T) {
	if os.Getenv("MYJOURNAL_TEST_MAIN_FAILURE") == "1" {
		main()
		return
	}
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestMainReportsStartupFailure$")
	// Use a controlled environment so developer credentials never enter the child.
	command.Env = []string{"MYJOURNAL_TEST_MAIN_FAILURE=1", "PORT=invalid"}
	output, err := command.CombinedOutput()
	require.NoError(t, ctx.Err())
	var exitError *exec.ExitError
	require.ErrorAs(t, err, &exitError)
	assert.Equal(t, 1, exitError.ExitCode())
	assert.Contains(t, string(output), "application failed")
	assert.Contains(t, string(output), "level=ERROR")
	assert.NotContains(t, string(output), "starting server")
}

func TestMainStartsAndShutsDown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	require.NoError(t, listener.Close())
	for key, value := range map[string]string{
		"PORT":                 port,
		"APP_BASE_URL":         "http://127.0.0.1:" + port,
		"SESSION_KEY":          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"PROVIDERS":            "local",
		"LOCAL_DATA_DIR":       t.TempDir(),
		"COOKIE_SECURE":        "false",
		"GOOGLE_TEST_MODE":     "false",
		"LOG_LEVEL":            "info",
		"GOOGLE_CLIENT_ID":     "",
		"GOOGLE_CLIENT_SECRET": "",
	} {
		t.Setenv(key, value)
	}
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	originalOutput, originalLogger := os.Stdout, slog.Default()
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = originalOutput
		slog.SetDefault(originalLogger)
		assert.NoError(t, writer.Close())
		assert.NoError(t, reader.Close())
	})
	started := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(reader)
		assert.NoError(t, scanner.Err())

		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "starting server") {
				close(started)
				return
			}
		}
	}()
	done := make(chan struct{})
	go func() {
		main()
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("application did not start")
	}
	process, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	// The startup log follows NotifyContext registration. Signal only this test
	// process, after its application shutdown handler is installed.
	require.NoError(t, process.Signal(os.Interrupt))
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("application did not stop after interrupt")
	}
}
