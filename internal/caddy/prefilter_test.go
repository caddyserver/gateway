// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2026 Wildcard

package caddy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPreFilterRoute(t *testing.T) {
	// Unset env -> no pre-filter, no error.
	t.Setenv(PreFilterPathEnv, "")
	if r, err := preFilterRoute(); err != nil || r != nil {
		t.Fatalf("unset env: want (nil,nil), got (%v,%v)", r, err)
	}

	// Empty array -> no route.
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(empty, []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(PreFilterPathEnv, empty)
	if r, err := preFilterRoute(); err != nil || r != nil {
		t.Fatalf("empty array: want (nil,nil), got (%v,%v)", r, err)
	}

	// Real handler list -> a matcher-less, non-terminal route that marshals
	// the handlers verbatim under "handle".
	p := filepath.Join(dir, "prefilter.json")
	content := `[{"handler":"blocker","allow_countries":["US"]},{"handler":"rate_limit","rate_limits":{"per_ip":{"key":"{http.request.remote.host}","window":"1m","max_events":600}}}]`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(PreFilterPathEnv, p)
	r, err := preFilterRoute()
	if err != nil {
		t.Fatalf("valid file: unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("valid file: expected a route, got nil")
	}
	if r.Terminal {
		t.Error("pre-filter route must not be terminal")
	}
	if len(r.MatcherSets) != 0 {
		t.Errorf("pre-filter route must have no matchers, got %d", len(r.MatcherSets))
	}
	if len(r.Handlers) != 2 {
		t.Fatalf("want 2 handlers, got %d", len(r.Handlers))
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	got := string(b)
	for _, want := range []string{`"handler":"blocker"`, `"handler":"rate_limit"`, `"max_events":600`} {
		if !contains(got, want) {
			t.Errorf("marshalled route missing %q; got %s", want, got)
		}
	}

	// Configured but unreadable -> error (fail loud).
	t.Setenv(PreFilterPathEnv, filepath.Join(dir, "does-not-exist.json"))
	if _, err := preFilterRoute(); err == nil {
		t.Error("missing file: expected error, got nil")
	}

	// Configured but invalid JSON -> error.
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{not an array}"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(PreFilterPathEnv, bad)
	if _, err := preFilterRoute(); err == nil {
		t.Error("invalid json: expected error, got nil")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
