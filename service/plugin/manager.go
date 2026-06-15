package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var (
	defaultPluginDir = "plugins"
	pluginDirEnv     = "PLUGIN_DIR"
)

// Manager owns the plugin registry and running subprocesses.
type Manager struct {
	mu        sync.RWMutex
	plugins   map[string]*Plugin
	processes map[string]*pluginProcess
	pluginDir string
}

var defaultManager = &Manager{
	plugins:   make(map[string]*Plugin),
	processes: make(map[string]*pluginProcess),
	pluginDir: "",
}

// Init initializes the default plugin manager and loads plugins.
func Init() error {
	return defaultManager.Init()
}

// List returns all discovered plugins with their current enabled state.
func List() []Plugin {
	return defaultManager.List()
}

// SetEnabled updates the enabled state of a plugin and restarts it if needed.
func SetEnabled(pluginId string, enabled bool) error {
	return defaultManager.SetEnabled(pluginId, enabled)
}

// RunPreRequest executes all enabled plugins that subscribe to the pre-request hook.
func RunPreRequest(ctx Context, request []byte) (*Result, []byte, error) {
	return defaultManager.RunHook(HookPreRequest, ctx, request)
}

// RunPostResponse executes all enabled plugins that subscribe to the post-response hook.
func RunPostResponse(ctx Context, response []byte) (*Result, []byte, error) {
	return defaultManager.RunHook(HookPostResponse, ctx, response)
}

// Reload rescans the plugin directory and reloads enabled plugins.
func Reload() error {
	return defaultManager.Reload()
}

// Init discovers the plugin directory, loads manifests, and starts enabled plugins.
func (m *Manager) Init() error {
	m.pluginDir = os.Getenv(pluginDirEnv)
	if m.pluginDir == "" {
		m.pluginDir = defaultPluginDir
	}

	if err := m.scan(); err != nil {
		common.SysLog("failed to scan plugins: " + err.Error())
	}

	m.startEnabled()
	return nil
}

func (m *Manager) scan() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, err := os.Stat(m.pluginDir)
	if err != nil {
		return fmt.Errorf("plugin directory not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("plugin path is not a directory")
	}

	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	dbPlugins, err := model.GetAllPlugins()
	if err != nil {
		common.SysLog("failed to load plugin states from db: " + err.Error())
		dbPlugins = []model.Plugin{}
	}

	enabledMap := make(map[string]bool)
	for _, p := range dbPlugins {
		enabledMap[p.PluginId] = p.Enabled
	}

	newPlugins := make(map[string]*Plugin)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginId := entry.Name()
		pluginPath := filepath.Join(m.pluginDir, pluginId)
		manifestPath := filepath.Join(pluginPath, "plugin.json")

		data, err := os.ReadFile(manifestPath)
		if err != nil {
			common.SysLog(fmt.Sprintf("plugin %s has no manifest: %v", pluginId, err))
			continue
		}

		var manifest Manifest
		if err := common.Unmarshal(data, &manifest); err != nil {
			common.SysLog(fmt.Sprintf("plugin %s has invalid manifest: %v", pluginId, err))
			continue
		}

		if manifest.ID == "" {
			manifest.ID = pluginId
		}

		plugin := &Plugin{
			Manifest: manifest,
			Path:     pluginPath,
			Enabled:  enabledMap[manifest.ID],
		}
		newPlugins[manifest.ID] = plugin
	}

	m.plugins = newPlugins
	return nil
}

func (m *Manager) startEnabled() {
	for _, p := range m.plugins {
		if p.Enabled && p.Entry != "" {
			if err := m.loadPlugin(p); err != nil {
				common.SysLog(fmt.Sprintf("failed to load plugin %s: %v", p.ID, err))
			}
		}
	}
	m.runStartupHooks()
}

func (m *Manager) runStartupHooks() {
	_, _, err := m.RunHook(HookStartup, Context{}, nil)
	if err != nil {
		common.SysLog("failed to run startup hooks: " + err.Error())
	}
}

func (m *Manager) loadPlugin(p *Plugin) error {
	pp, err := newPluginProcess(p, p.Path)
	if err != nil {
		p.Loaded = false
		p.Error = err.Error()
		return err
	}
	m.processes[p.ID] = pp
	p.Loaded = true
	p.Error = ""
	common.SysLog(fmt.Sprintf("plugin %s loaded", p.ID))
	return nil
}

func (m *Manager) unloadPlugin(pluginId string) {
	if pp, ok := m.processes[pluginId]; ok {
		_ = pp.Stop()
		delete(m.processes, pluginId)
	}
	if p, ok := m.plugins[pluginId]; ok {
		p.Loaded = false
	}
}

// List returns a snapshot of all plugins.
func (m *Manager) List() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		result = append(result, *p)
	}
	return result
}

// SetEnabled persists the enabled state and reloads the plugin process.
func (m *Manager) SetEnabled(pluginId string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plugins[pluginId]
	if !ok {
		return fmt.Errorf("plugin not found: %s", pluginId)
	}

	if err := model.SavePlugin(&model.Plugin{
		PluginId: pluginId,
		Enabled:  enabled,
	}); err != nil {
		return fmt.Errorf("failed to save plugin state: %w", err)
	}

	p.Enabled = enabled

	if enabled && p.Entry != "" {
		if pp, ok := m.processes[pluginId]; ok && pp.IsAlive() {
			return nil
		}
		m.unloadPlugin(pluginId)
		return m.loadPlugin(p)
	}

	m.unloadPlugin(pluginId)
	return nil
}

// RunHook executes all enabled plugins subscribed to the given hook.
// It returns the first deny result, or the last modified payload, or nil if all allowed.
func (m *Manager) RunHook(hook string, ctx Context, payload []byte) (*Result, []byte, error) {
	m.mu.RLock()
	plugins := make([]*Plugin, 0, len(m.plugins))
	processes := make(map[string]*pluginProcess, len(m.processes))
	for _, p := range m.plugins {
		if p.Enabled && p.SupportsHook(hook) {
			plugins = append(plugins, p)
		}
	}
	for id, pp := range m.processes {
		processes[id] = pp
	}
	m.mu.RUnlock()

	if len(plugins) == 0 {
		return nil, payload, nil
	}

	var currentPayload []byte
	if len(payload) > 0 {
		currentPayload = append([]byte(nil), payload...)
	}

	for _, p := range plugins {
		pp, ok := m.processes[p.ID]
		if !ok || !pp.IsAlive() {
			common.SysLog(fmt.Sprintf("plugin %s is not running, skipping %s", p.ID, hook))
			continue
		}

		event := &Event{
			Hook:    hook,
			Context: ctx,
			Request: currentPayload,
		}
		result, err := pp.Call(event)
		if err != nil {
			common.SysLog(fmt.Sprintf("plugin %s %s error: %v", p.ID, hook, err))
			continue
		}
		if result == nil {
			continue
		}

		switch result.Action {
		case ActionDeny:
			return result, currentPayload, nil
		case ActionModify:
			if len(result.Request) > 0 {
				currentPayload = append([]byte(nil), result.Request...)
			}
		case ActionAllow:
			// do nothing
		default:
			common.SysLog(fmt.Sprintf("plugin %s returned unknown action: %s", p.ID, result.Action))
		}
	}

	return nil, currentPayload, nil
}

// Reload rescans the plugin directory and reloads enabled plugins.
func (m *Manager) Reload() error {
	m.mu.Lock()
	for id := range m.processes {
		m.unloadPlugin(id)
	}
	m.mu.Unlock()
	return m.Init()
}
