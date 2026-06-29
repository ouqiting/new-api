package plugin

import (
	"encoding/json"
)

const (
	HookStartup     = "startup"
	HookPreRequest  = "pre-request"
	HookPostResponse = "post-response"
)

const (
	ActionAllow  = "allow"
	ActionDeny   = "deny"
	ActionModify = "modify"
)

// Manifest is the plugin.json schema.
type Manifest struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Author       string                 `json:"author,omitempty"`
	Entry        string                 `json:"entry,omitempty"`
	Hooks        []string               `json:"hooks,omitempty"`
	Capabilities []string               `json:"capabilities,omitempty"`
	// Log declares whether this plugin writes a usage log entry when it is
	// triggered during a request. The actual per-trigger decision can still be
	// overridden by the plugin via Result.Log.
	Log    bool                   `json:"log,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// Plugin represents a loaded plugin, including its disk path and runtime state.
type Plugin struct {
	Manifest
	Path    string `json:"-"`
	Enabled bool   `json:"enabled"`
	Loaded  bool   `json:"loaded"`
	Error   string `json:"error,omitempty"`
}

// Context carries information about the current request/user.
type Context struct {
	UserID    int    `json:"userId"`
	Role      string `json:"role"`
	Group     string `json:"group,omitempty"`
	TokenName string `json:"tokenName,omitempty"`
	Model     string `json:"model,omitempty"`
}

// Event is the payload sent to a plugin.
type Event struct {
	Hook    string          `json:"hook"`
	Context Context         `json:"context"`
	Request json.RawMessage `json:"request,omitempty"`
}

// Result is the response expected from a plugin.
type Result struct {
	Action string          `json:"action"`
	Code   int             `json:"code,omitempty"`
	Error  string          `json:"error,omitempty"`
	Request json.RawMessage `json:"request,omitempty"`
	// Log lets a plugin decide, per trigger, whether to write a usage log entry.
	// nil => fall back to the manifest's Log flag; non-nil => explicit override.
	Log *bool `json:"log,omitempty"`
	// LogContent is the human-readable message stored in the log entry. When
	// empty a default message is generated from the plugin title and action.
	LogContent string `json:"logContent,omitempty"`
	// LogDetail is arbitrary extra information stored in the log's Other field
	// (visible to admin/root only).
	LogDetail map[string]interface{} `json:"logDetail,omitempty"`
}

// IsLoaded returns true if the plugin has an entry and the process is running.
func (p *Plugin) IsLoaded() bool {
	return p.Loaded
}

// SupportsHook checks if the plugin subscribes to a given hook.
func (p *Plugin) SupportsHook(hook string) bool {
	for _, h := range p.Hooks {
		if h == hook {
			return true
		}
	}
	return false
}

// HasCapability checks if the plugin declares a capability.
func (p *Plugin) HasCapability(cap string) bool {
	for _, c := range p.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}
