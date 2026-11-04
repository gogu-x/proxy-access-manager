package key

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogu-x/proxy-access-manager/model"
)

func TestPresentRequiresMatchingEmail(t *testing.T) {
	cases := []struct {
		name    string
		output  string
		present bool
	}{
		{"missing placeholder", `{"users":[{}]}`, false},
		{"empty list", `{"users":[]}`, false},
		{"other user", `{"users":[{"email":"other@keymgr.local"}]}`, false},
		{"matching user", `{"users":[{"email":"user-id@keymgr.local"}]}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "xray")
			if err := os.WriteFile(binary, []byte("#!/bin/sh\nprintf '%s\\n' '"+tc.output+"'\n"), 0700); err != nil {
				t.Fatal(err)
			}
			client := NewProxy(Config{XrayBin: binary, XrayTag: "vless-in", XrayAPI: "127.0.0.1:10085"})
			present, err := client.Present(context.Background(), model.ProxyKey{ID: "user-id"})
			if err != nil || present != tc.present {
				t.Fatalf("Present() = %v, %v; want %v", present, err, tc.present)
			}
		})
	}
}
