package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerProcessExitStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell signals")
	}
	for _, tt := range []struct {
		name, command string
		parentSignal  bool
		code          int
	}{
		{name: "success", command: "exit 0", code: 0},
		{name: "failure", command: "exit 7", code: 7},
		{name: "child signal", command: "kill -TERM $$", code: 143},
		{name: "parent signal", command: "exec /bin/sleep 60", parentSignal: true, code: 130},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			ready := filepath.Join(dir, "ready")
			command := tt.command
			if tt.parentSignal {
				command = "touch " + strconv.Quote(ready) + "; " + command
			}
			cfg := fmt.Sprintf(`app:
  env: prod
http:
  port: 0
database:
  driver: sqlite
  sqlite:
    dbFile: %q
logger:
  level: error
  format: json
upstream:
  enabled: true
  command: /bin/sh
  args: ["-c", %q]
`, filepath.Join(dir, "test.sqlite3"), command)
			cfgPath := filepath.Join(dir, "server.yml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(cfg), 0600))
			binary, err := os.Executable()
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, "-test.run=^TestServerProcessHelper$")
			cmd.Env = append(os.Environ(), "THRUST_PROCESS_TEST_CONFIG="+cfgPath)
			if tt.parentSignal {
				require.NoError(t, cmd.Start())
				require.Eventually(t, func() bool { _, err := os.Stat(ready); return err == nil }, 5*time.Second, time.Millisecond)
				require.NoError(t, cmd.Process.Signal(syscall.SIGINT))
				err = cmd.Wait()
			} else {
				var output []byte
				output, err = cmd.CombinedOutput()
				t.Log(string(output))
			}
			require.NoError(t, ctx.Err(), "server did not shut down")
			if tt.code == 0 {
				require.NoError(t, err)
			} else {
				var exitErr *exec.ExitError
				require.True(t, errors.As(err, &exitErr), "%v", err)
				require.Equal(t, tt.code, exitErr.ExitCode())
			}
		})
	}
}

func TestServerProcessHelper(t *testing.T) {
	configFile := os.Getenv("THRUST_PROCESS_TEST_CONFIG")
	if configFile == "" {
		return
	}
	flag.CommandLine = flag.NewFlagSet("thrustOauth2idServer", flag.ExitOnError)
	os.Args = []string{os.Args[0], "-c", configFile}
	main()
}
