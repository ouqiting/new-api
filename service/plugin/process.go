package plugin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const defaultTimeout = 5 * time.Second

// pluginProcess wraps a running plugin subprocess.
type pluginProcess struct {
	plugin *Plugin
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser
	mu     sync.Mutex
	alive  bool
}

func buildCommand(entryPath string) *exec.Cmd {
	ext := strings.ToLower(filepath.Ext(entryPath))
	switch ext {
	case ".js":
		return exec.Command("node", entryPath)
	case ".py":
		return exec.Command("python", entryPath)
	case ".sh":
		return exec.Command("bash", entryPath)
	default:
		return exec.Command(entryPath)
	}
}

func newPluginProcess(p *Plugin, pluginDir string) (*pluginProcess, error) {
	entryPath := filepath.Join(pluginDir, p.Entry)
	info, err := os.Stat(entryPath)
	if err != nil {
		return nil, fmt.Errorf("plugin entry not found: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("plugin entry is a directory")
	}

	cmd := buildCommand(entryPath)
	cmd.Dir = pluginDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start plugin: %w", err)
	}

	pp := &pluginProcess{
		plugin: p,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		stderr: stderr,
		alive:  true,
	}

	go pp.consumeStderr()
	go pp.waitExit()

	return pp, nil
}

func (pp *pluginProcess) consumeStderr() {
	scanner := bufio.NewScanner(pp.stderr)
	scanner.Buffer(make([]byte, 1024), 64*1024)
	for scanner.Scan() {
		common.SysLog(fmt.Sprintf("plugin %s stderr: %s", pp.plugin.ID, scanner.Text()))
	}
}

func (pp *pluginProcess) waitExit() {
	if err := pp.cmd.Wait(); err != nil {
		common.SysLog(fmt.Sprintf("plugin %s exited: %v", pp.plugin.ID, err))
	} else {
		common.SysLog(fmt.Sprintf("plugin %s exited normally", pp.plugin.ID))
	}
	pp.mu.Lock()
	pp.alive = false
	pp.mu.Unlock()
}

func (pp *pluginProcess) IsAlive() bool {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	return pp.alive
}

func (pp *pluginProcess) Call(event *Event) (*Result, error) {
	pp.mu.Lock()
	if !pp.alive {
		pp.mu.Unlock()
		return nil, fmt.Errorf("plugin process is not running")
	}
	pp.mu.Unlock()

	payload, err := common.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	payload = append(payload, '\n')

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	type callResult struct {
		result *Result
		err    error
	}
	ch := make(chan callResult, 1)

	go func() {
		pp.mu.Lock()
		_, err := pp.stdin.Write(payload)
		pp.mu.Unlock()
		if err != nil {
			ch <- callResult{nil, fmt.Errorf("failed to write to plugin: %w", err)}
			return
		}

		line, err := pp.stdout.ReadBytes('\n')
		if err != nil {
			ch <- callResult{nil, fmt.Errorf("failed to read from plugin: %w", err)}
			return
		}

		var result Result
		if err := common.Unmarshal(line, &result); err != nil {
			ch <- callResult{nil, fmt.Errorf("failed to decode plugin response: %w", err)}
			return
		}
		ch <- callResult{&result, nil}
	}()

	select {
	case <-ctx.Done():
		_ = pp.cmd.Process.Kill()
		pp.mu.Lock()
		pp.alive = false
		pp.mu.Unlock()
		return nil, fmt.Errorf("plugin call timed out")
	case res := <-ch:
		return res.result, res.err
	}
}

func (pp *pluginProcess) Stop() error {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if !pp.alive {
		return nil
	}
	pp.alive = false
	if pp.cmd.Process != nil {
		_ = pp.cmd.Process.Kill()
	}
	return pp.cmd.Wait()
}
