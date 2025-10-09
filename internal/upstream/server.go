package upstream

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-dev-frame/sponge/pkg/logger"

	"thrust_oauth2id/internal/config"
)

const defaultStopTimeout = 10 * time.Second

// Server supervises an upstream command, relaying logs and signals.
type Server struct {
	cfg      config.Upstream
	mu       sync.Mutex
	cmd      *exec.Cmd
	done     chan struct{}
	stopping bool
}

// NewServer creates a supervisor for the configured upstream command.
func NewServer(cfg config.Upstream) *Server {
	return &Server{cfg: cfg}
}

// Start launches the upstream command and blocks until it exits.
func (s *Server) Start() error {
	if s.cfg.Enabled && s.cfg.Command == "" {
		return errors.New("upstream command is empty")
	}

	cmd := exec.Command(s.cfg.Command, s.cfg.Args...)
	if s.cfg.Enabled && s.cfg.WorkingDirectory != "" {
		if _, err := os.Stat(s.cfg.WorkingDirectory); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("working directory does not exist: %s", s.cfg.WorkingDirectory)
			}
			return fmt.Errorf("cannot access working directory %s: %w", s.cfg.WorkingDirectory, err)
		}
		cmd.Dir = s.cfg.WorkingDirectory
	}

	cmd.Env = s.buildEnv()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan struct{})

	s.mu.Lock()
	s.cmd = cmd
	s.done = done
	s.stopping = false
	s.mu.Unlock()

	defer func() {
		close(done)
		s.mu.Lock()
		s.cmd = nil
		s.done = nil
		s.mu.Unlock()
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start upstream command: %w", err)
	}

	logger.Info("upstream process started",
		logger.String("command", s.cfg.Command),
		logger.Any("args", s.cfg.Args),
		logger.Int("pid", cmd.Process.Pid),
		logger.String("working_dir", cmd.Dir))

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode := exitErr.ExitCode()

			s.mu.Lock()
			stopping := s.stopping
			s.mu.Unlock()

			if stopping {
				logger.Info("upstream process exited",
					logger.Int("pid", cmd.Process.Pid),
					logger.Int("exit_code", exitCode))
				return nil
			}

			return fmt.Errorf("upstream process exited with code %d", exitCode)
		}

		return fmt.Errorf("wait upstream command: %w", err)
	}

	logger.Info("upstream process exited",
		logger.Int("pid", cmd.Process.Pid),
		logger.Int("exit_code", 0))

	return nil
}

// Stop attempts to gracefully stop the upstream process.
func (s *Server) Stop() error {
	s.mu.Lock()
	cmd := s.cmd
	done := s.done
	s.stopping = true
	s.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	sig, err := parseSignal(s.cfg.StopSignal)
	if err != nil {
		logger.Warn("invalid stop signal; defaulting to SIGTERM", logger.String("signal", s.cfg.StopSignal), logger.Err(err))
		sig = syscall.SIGTERM
	}

	logger.Info("sending signal to upstream process",
		logger.Int("pid", cmd.Process.Pid),
		logger.String("signal", sig.String()))

	if err := cmd.Process.Signal(sig); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("signal upstream process: %w", err)
	}

	if done != nil {
		select {
		case <-done:
		case <-time.After(defaultStopTimeout):
			logger.Warn("upstream process did not exit within timeout; killing",
				logger.Int("pid", cmd.Process.Pid),
				logger.Duration("timeout", defaultStopTimeout))
			if killErr := cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
				return fmt.Errorf("kill upstream process: %w", killErr)
			}
			if done != nil {
				<-done
			}
		}
	}

	return nil
}

// String implements app.IServer for logging purposes.
func (s *Server) String() string {
	return "upstream process supervisor"
}

func (s *Server) buildEnv() []string {
	merged := map[string]string{}

	for _, kv := range os.Environ() {
		if idx := strings.Index(kv, "="); idx != -1 {
			merged[kv[:idx]] = kv[idx+1:]
		}
	}

	// Only export PORT when not using a UNIX socket binding to avoid conflicts.
	if s.cfg.TargetBindSocket == "" && s.cfg.TargetPort > 0 {
		merged["PORT"] = strconv.Itoa(s.cfg.TargetPort)
	}

	for key, value := range s.cfg.Env {
		merged[key] = value
	}

	env := make([]string, 0, len(merged))
	for key, value := range merged {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

func parseSignal(name string) (syscall.Signal, error) {
	if name == "" {
		return syscall.SIGTERM, nil
	}

	switch strings.ToUpper(name) {
	case "SIGTERM", "TERM":
		return syscall.SIGTERM, nil
	case "SIGINT", "INT":
		return syscall.SIGINT, nil
	case "SIGQUIT", "QUIT":
		return syscall.SIGQUIT, nil
	case "SIGKILL", "KILL":
		return syscall.SIGKILL, nil
	}

	return 0, fmt.Errorf("unsupported signal %q", name)
}
