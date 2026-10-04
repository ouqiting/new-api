package passkey

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/go-webauthn/webauthn/protocol"
)

const testPasskeyHost = "api.ouqiting.shop"

func newTestRequest(t *testing.T, host string, forwardedProto string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "https://"+host+"/api/user/passkey/verify/begin", nil)
	req.Host = host
	if forwardedProto != "" {
		req.Header.Set("X-Forwarded-Proto", forwardedProto)
	}
	return req
}

// 浏览器上报的 Origin 必须能在允许列表里匹配到，否则 go-webauthn 会返回 "Error validating origin"
func requireBrowserOriginMatched(t *testing.T, origins []string, browserOrigin string) {
	t.Helper()
	if !protocol.IsOriginInHaystack(browserOrigin, origins) {
		t.Fatalf("browser origin %s not matched by allowed origins %v", browserOrigin, origins)
	}
}

func TestResolveOriginsAppendsMissingScheme(t *testing.T) {
	req := newTestRequest(t, testPasskeyHost, "https")
	settings := &system_setting.PasskeySettings{Origins: testPasskeyHost}

	origins, err := resolveOrigins(req, settings)
	if err != nil {
		t.Fatalf("resolveOrigins() error = %v", err)
	}
	if len(origins) != 1 || origins[0] != "https://"+testPasskeyHost {
		t.Fatalf("resolveOrigins() = %v, want [https://%s]", origins, testPasskeyHost)
	}
	requireBrowserOriginMatched(t, origins, "https://"+testPasskeyHost)
}

func TestResolveOriginsAcceptsMultipleEntries(t *testing.T) {
	cases := map[string][]string{
		"comma":        {"https://a.example.com,https://b.example.com"},
		"comma+space":  {"https://a.example.com, https://b.example.com"},
		"newline":      {"https://a.example.com\nhttps://b.example.com"},
		"crlf":         {"https://a.example.com\r\nhttps://b.example.com"},
		"json array":   {`["https://a.example.com","https://b.example.com"]`},
		"mixed scheme": {"a.example.com,https://b.example.com"},
	}

	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			req := newTestRequest(t, testPasskeyHost, "https")
			origins, err := resolveOrigins(req, &system_setting.PasskeySettings{Origins: entries[0]})
			if err != nil {
				t.Fatalf("resolveOrigins() error = %v", err)
			}
			want := []string{"https://a.example.com", "https://b.example.com"}
			if len(origins) != len(want) {
				t.Fatalf("resolveOrigins() = %v, want %v", origins, want)
			}
			for i := range want {
				if origins[i] != want[i] {
					t.Fatalf("resolveOrigins() = %v, want %v", origins, want)
				}
			}
			requireBrowserOriginMatched(t, origins, "https://a.example.com")
		})
	}
}

func TestNormalizeOriginCleansUpEntry(t *testing.T) {
	cases := map[string]struct {
		entry string
		want  string
	}{
		"bare host":       {entry: testPasskeyHost, want: "https://" + testPasskeyHost},
		"trailing slash":  {entry: "https://" + testPasskeyHost + "/", want: "https://" + testPasskeyHost},
		"with path":       {entry: "https://" + testPasskeyHost + "/channels", want: "https://" + testPasskeyHost},
		"quoted":          {entry: `"https://` + testPasskeyHost + `"`, want: "https://" + testPasskeyHost},
		"with port":       {entry: testPasskeyHost + ":8443", want: "https://" + testPasskeyHost + ":8443"},
		"uppercase":       {entry: "HTTPS://" + testPasskeyHost, want: "https://" + testPasskeyHost},
		"blank":           {entry: "   ", want: ""},
		"localhost dev":   {entry: "localhost:3000", want: "https://localhost:3000"},
		"wildcard origin": {entry: "https://*.example.com", want: "https://*.example.com"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := normalizeOrigin(tc.entry, "https")
			if err != nil {
				t.Fatalf("normalizeOrigin(%q) error = %v", tc.entry, err)
			}
			if got != tc.want {
				t.Fatalf("normalizeOrigin(%q) = %q, want %q", tc.entry, got, tc.want)
			}
		})
	}
}

func TestNormalizeOriginRejectsInvalidEntry(t *testing.T) {
	for _, entry := range []string{"ftp://example.com", "https://", "://example.com"} {
		if got, err := normalizeOrigin(entry, "https"); err == nil {
			t.Fatalf("normalizeOrigin(%q) = %q, want error", entry, got)
		}
	}
}

func TestResolveOriginsInsecureOrigin(t *testing.T) {
	req := newTestRequest(t, "127.0.0.1:3000", "http")
	req.Host = "127.0.0.1:3000"

	if _, err := resolveOrigins(req, &system_setting.PasskeySettings{Origins: "127.0.0.1:3000"}); err == nil {
		t.Fatal("resolveOrigins() should reject insecure origin when not allowed")
	}

	origins, err := resolveOrigins(req, &system_setting.PasskeySettings{
		Origins:             "127.0.0.1:3000",
		AllowInsecureOrigin: true,
	})
	if err != nil {
		t.Fatalf("resolveOrigins() error = %v", err)
	}
	if len(origins) != 1 || origins[0] != "http://127.0.0.1:3000" {
		t.Fatalf("resolveOrigins() = %v, want [http://127.0.0.1:3000]", origins)
	}
	requireBrowserOriginMatched(t, origins, "http://127.0.0.1:3000")
}

func TestResolveOriginsLegacyEmptyArrayFallsBackToRequest(t *testing.T) {
	req := newTestRequest(t, testPasskeyHost, "https")

	origins, err := resolveOrigins(req, &system_setting.PasskeySettings{Origins: "[]"})
	if err != nil {
		t.Fatalf("resolveOrigins() error = %v", err)
	}
	if len(origins) != 1 || origins[0] != "https://"+testPasskeyHost {
		t.Fatalf("resolveOrigins() = %v, want [https://%s]", origins, testPasskeyHost)
	}
	requireBrowserOriginMatched(t, origins, "https://"+testPasskeyHost)
}

func TestResolveOriginsAutoDetectsFromRequest(t *testing.T) {
	req := newTestRequest(t, testPasskeyHost, "https")

	origins, err := resolveOrigins(req, &system_setting.PasskeySettings{})
	if err != nil {
		t.Fatalf("resolveOrigins() error = %v", err)
	}
	if len(origins) != 1 || origins[0] != "https://"+testPasskeyHost {
		t.Fatalf("resolveOrigins() = %v, want [https://%s]", origins, testPasskeyHost)
	}
}

func TestResolveRPIDAcceptsHostPortAndURL(t *testing.T) {
	req := newTestRequest(t, testPasskeyHost, "https")
	origins := []string{"https://" + testPasskeyHost}

	cases := map[string]string{
		"":                                     testPasskeyHost,
		testPasskeyHost:                        testPasskeyHost,
		"https://" + testPasskeyHost:           testPasskeyHost,
		testPasskeyHost + ":8443":              testPasskeyHost,
		"https://" + testPasskeyHost + ":8443": testPasskeyHost,
	}

	for configured, want := range cases {
		got, err := resolveRPID(req, &system_setting.PasskeySettings{RPID: configured}, origins)
		if err != nil {
			t.Fatalf("resolveRPID(%q) error = %v", configured, err)
		}
		if got != want {
			t.Fatalf("resolveRPID(%q) = %q, want %q", configured, got, want)
		}
	}
}
