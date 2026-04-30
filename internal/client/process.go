package client

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type ProcessManager struct {
	mu        sync.Mutex
	processes map[string]*managedProcess
	logPath   string
	detached  bool
}

type managedProcess struct {
	cmd        *exec.Cmd
	pid        int
	frpcPath   string
	configPath string
	logPath    string
	restartKey string
	lastErr    string
	detached   bool
	exited     bool
	done       chan struct{}
}

type runtimeFrpcConfig struct {
	NodeID     string
	ConfigPath string
	LogPath    string
	RestartKey string
	Raw        string
}

type ProcessManagerOption func(*ProcessManager)

func WithDetachedProcesses() ProcessManagerOption {
	return func(pm *ProcessManager) {
		pm.detached = true
	}
}

func NewProcessManager(logPath string, options ...ProcessManagerOption) *ProcessManager {
	pm := &ProcessManager{
		processes: map[string]*managedProcess{},
		logPath:   logPath,
	}
	for _, option := range options {
		option(pm)
	}
	return pm
}

func (pm *ProcessManager) Start(ctx context.Context, frpcPath, configPath string) error {
	return pm.startKey(ctx, defaultProcessKey, frpcPath, configPath, pm.logPath, "")
}

func (pm *ProcessManager) startKey(ctx context.Context, key, frpcPath, configPath, logPath, restartKey string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if proc := pm.processes[key]; proc != nil && runningLocked(proc) {
		return nil
	}
	if frpcPath == "" {
		return errors.New("frpc path is empty")
	}
	if logPath == "" {
		logPath = pm.logPath
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		_ = logFile.Close()
		return err
	}
	// frpc is a long-running child process. Do not bind it to the HTTP request
	// context, otherwise a successful create/reload request kills frpc as soon as
	// the response is returned.
	cmd := exec.Command(frpcPath, "-c", configPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if pm.detached {
		configureDetachedCommand(cmd)
	}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}
	proc := &managedProcess{
		cmd:        cmd,
		pid:        cmd.Process.Pid,
		frpcPath:   frpcPath,
		configPath: configPath,
		logPath:    logPath,
		restartKey: restartKey,
		detached:   pm.detached,
		done:       make(chan struct{}),
	}
	pm.processes[key] = proc
	if pm.detached {
		if err := cmd.Process.Release(); err != nil {
			delete(pm.processes, key)
			_ = logFile.Close()
			return err
		}
		_ = logFile.Close()
		return nil
	}
	go func() {
		err := cmd.Wait()
		_ = logFile.Close()
		pm.mu.Lock()
		defer pm.mu.Unlock()
		proc.exited = true
		if current := pm.processes[key]; current == proc && err != nil {
			proc.lastErr = err.Error()
		}
		close(proc.done)
	}()
	return nil
}

func (pm *ProcessManager) Reload(ctx context.Context, frpcPath, configPath string) error {
	return pm.reloadKey(ctx, defaultProcessKey, frpcPath, runtimeFrpcConfig{
		NodeID:     defaultProcessKey,
		ConfigPath: configPath,
		LogPath:    pm.logPath,
	})
}

func (pm *ProcessManager) Apply(ctx context.Context, frpcPath string, configs []runtimeFrpcConfig) error {
	desired := make(map[string]runtimeFrpcConfig, len(configs))
	for _, cfg := range configs {
		key := processKey(cfg.NodeID)
		desired[key] = cfg
	}

	pm.mu.Lock()
	var stale []string
	for key, proc := range pm.processes {
		if _, ok := desired[key]; !ok && runningLocked(proc) {
			stale = append(stale, key)
		}
	}
	pm.mu.Unlock()
	sort.Strings(stale)
	for _, key := range stale {
		pm.stopKey(key)
	}

	sort.Slice(configs, func(i, j int) bool {
		return configs[i].NodeID < configs[j].NodeID
	})
	for _, cfg := range configs {
		if err := pm.reloadKey(ctx, processKey(cfg.NodeID), frpcPath, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (pm *ProcessManager) reloadKey(ctx context.Context, key, frpcPath string, cfg runtimeFrpcConfig) error {
	pm.mu.Lock()
	proc := pm.processes[key]
	running := proc != nil && runningLocked(proc)
	needsRestart := running && (proc.frpcPath != frpcPath || proc.configPath != cfg.ConfigPath || proc.restartKey != cfg.RestartKey)
	pm.mu.Unlock()
	if needsRestart {
		pm.stopKey(key)
		running = false
	}
	if !running {
		return pm.startKey(ctx, key, frpcPath, cfg.ConfigPath, cfg.LogPath, cfg.RestartKey)
	}

	reloadCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(reloadCtx, frpcPath, "reload", "-c", cfg.ConfigPath)
	out, err := cmd.CombinedOutput()
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		if proc := pm.processes[key]; proc != nil {
			proc.lastErr = msg
		}
		return errors.New(msg)
	}
	if proc := pm.processes[key]; proc != nil {
		proc.lastErr = ""
		proc.configPath = cfg.ConfigPath
		proc.logPath = cfg.LogPath
		proc.restartKey = cfg.RestartKey
	}
	return nil
}

func (pm *ProcessManager) Verify(ctx context.Context, frpcPath, configPath string) error {
	verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(verifyCtx, frpcPath, "verify", "-c", configPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(msg)
	}
	return nil
}

func (pm *ProcessManager) Status(configPath string) FrpcStatus {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	proc := pm.processes[defaultProcessKey]
	return statusForProcess(defaultProcessKey, configPath, pm.logPath, proc).toFrpcStatus()
}

func (pm *ProcessManager) StatusAll(configs []runtimeFrpcConfig) FrpcStatus {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if len(configs) == 0 {
		return FrpcStatus{}
	}
	nodes := make([]FrpcNodeStatus, 0, len(configs))
	var errs []string
	for _, cfg := range configs {
		node := statusForProcess(cfg.NodeID, cfg.ConfigPath, cfg.LogPath, pm.processes[processKey(cfg.NodeID)])
		nodes = append(nodes, node)
		if node.Running {
			// Keep the legacy top-level fields useful when only one process exists.
		}
		if node.LastError != "" {
			errs = append(errs, node.NodeID+": "+node.LastError)
		}
	}
	status := FrpcStatus{
		ConfigPath: configs[0].ConfigPath,
		Nodes:      nodes,
	}
	for _, node := range nodes {
		if node.Running {
			status.Running = true
			if status.PID == 0 && len(nodes) == 1 {
				status.PID = node.PID
			}
		}
	}
	if len(errs) > 0 {
		status.LastError = strings.Join(errs, "\n")
	}
	if len(nodes) == 1 {
		status.ConfigPath = nodes[0].ConfigPath
		status.LastError = nodes[0].LastError
		status.PID = nodes[0].PID
	}
	return status
}

func (pm *ProcessManager) Logs() string {
	raw, err := os.ReadFile(pm.logPath)
	if err != nil {
		return ""
	}
	const max = 64 * 1024
	if len(raw) > max {
		raw = raw[len(raw)-max:]
	}
	return string(raw)
}

func (pm *ProcessManager) LogsFor(configs []runtimeFrpcConfig) string {
	if len(configs) == 0 {
		return pm.Logs()
	}
	var b strings.Builder
	for _, cfg := range configs {
		logPath := cfg.LogPath
		if logPath == "" {
			logPath = pm.logPath
		}
		raw, err := os.ReadFile(logPath)
		if err != nil {
			continue
		}
		const max = 64 * 1024
		if len(raw) > max {
			raw = raw[len(raw)-max:]
		}
		if len(configs) > 1 {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("===== frpc node: ")
			b.WriteString(cfg.NodeID)
			b.WriteString(" =====\n")
		}
		b.Write(raw)
		if !strings.HasSuffix(b.String(), "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (pm *ProcessManager) StopAll() {
	pm.mu.Lock()
	keys := make([]string, 0, len(pm.processes))
	for key := range pm.processes {
		keys = append(keys, key)
	}
	pm.mu.Unlock()
	sort.Strings(keys)
	for _, key := range keys {
		pm.stopKey(key)
	}
}

func (pm *ProcessManager) stopKey(key string) {
	pm.mu.Lock()
	proc := pm.processes[key]
	delete(pm.processes, key)
	pm.mu.Unlock()
	if proc == nil || proc.cmd == nil || proc.cmd.Process == nil {
		return
	}
	if proc.detached {
		if proc.pid != 0 {
			if process, err := os.FindProcess(proc.pid); err == nil {
				_ = process.Kill()
				_ = process.Release()
			}
		}
		return
	}
	if runningLocked(proc) {
		_ = proc.cmd.Process.Kill()
	}
	select {
	case <-proc.done:
	case <-time.After(3 * time.Second):
	}
}

func runningLocked(proc *managedProcess) bool {
	if proc != nil && proc.detached {
		if proc.exited || proc.pid == 0 {
			if proc.lastErr == "" {
				proc.lastErr = "process exited"
			}
			return false
		}
		if !processExists(proc.pid) {
			proc.exited = true
			if proc.lastErr == "" {
				proc.lastErr = "process exited"
			}
			return false
		}
		return true
	}
	return proc != nil && !proc.exited && proc.cmd != nil && proc.cmd.Process != nil && proc.cmd.ProcessState == nil
}

func statusForProcess(nodeID, configPath, logPath string, proc *managedProcess) FrpcNodeStatus {
	status := FrpcNodeStatus{
		NodeID:     nodeID,
		ConfigPath: configPath,
		LogPath:    logPath,
	}
	if proc != nil {
		if status.ConfigPath == "" {
			status.ConfigPath = proc.configPath
		}
		if status.LogPath == "" {
			status.LogPath = proc.logPath
		}
		if runningLocked(proc) {
			status.Running = true
			status.PID = proc.pid
			if status.PID == 0 && proc.cmd != nil && proc.cmd.Process != nil {
				status.PID = proc.cmd.Process.Pid
			}
		}
		status.LastError = proc.lastErr
	}
	return status
}

func (s FrpcNodeStatus) toFrpcStatus() FrpcStatus {
	return FrpcStatus{
		Running:    s.Running,
		PID:        s.PID,
		ConfigPath: s.ConfigPath,
		LastError:  s.LastError,
	}
}

func processKey(nodeID string) string {
	if strings.TrimSpace(nodeID) == "" {
		return defaultProcessKey
	}
	return nodeID
}

const defaultProcessKey = "default"
