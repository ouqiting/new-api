package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginProcessAllow(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "main.js")
	if err := os.WriteFile(script, []byte(`
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin, output: process.stdout, terminal: false });
rl.on('line', () => {
  console.log(JSON.stringify({ action: 'allow' }));
});
`), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Plugin{
		Manifest: Manifest{ID: "allow-test", Entry: "main.js"},
		Path:     dir,
	}

	pp, err := newPluginProcess(p, dir)
	if err != nil {
		t.Fatalf("failed to start plugin: %v", err)
	}
	defer pp.Stop()

	result, err := pp.Call(&Event{Hook: HookPreRequest})
	if err != nil {
		t.Fatalf("failed to call plugin: %v", err)
	}
	if result == nil || result.Action != ActionAllow {
		t.Fatalf("expected allow, got %v", result)
	}
}

func TestPluginProcessDeny(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "main.js")
	if err := os.WriteFile(script, []byte(`
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin, output: process.stdout, terminal: false });
rl.on('line', () => {
  console.log(JSON.stringify({ action: 'deny', code: 403, error: 'denied' }));
});
`), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Plugin{
		Manifest: Manifest{ID: "deny-test", Entry: "main.js"},
		Path:     dir,
	}

	pp, err := newPluginProcess(p, dir)
	if err != nil {
		t.Fatalf("failed to start plugin: %v", err)
	}
	defer pp.Stop()

	result, err := pp.Call(&Event{Hook: HookPreRequest})
	if err != nil {
		t.Fatalf("failed to call plugin: %v", err)
	}
	if result == nil || result.Action != ActionDeny {
		t.Fatalf("expected deny, got %v", result)
	}
	if result.Code != 403 || result.Error != "denied" {
		t.Fatalf("unexpected deny result: %v", result)
	}
}
