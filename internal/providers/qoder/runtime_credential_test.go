package qoder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/caigee-cmd/cli2api/internal/accounts"
	"github.com/caigee-cmd/cli2api/internal/translate"
)

func TestDirectUsesRefreshedRuntimeCredentialOnEveryRequest(t *testing.T) {
	for _, region := range []string{"global", "cn"} {
		t.Run(region, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "runtime", "acc")
			authDir := AuthDir(home, region)
			if err := os.MkdirAll(authDir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(authDir, "machine_id"), []byte("0123456789abcdef"), 0o600); err != nil {
				t.Fatal(err)
			}
			// An unusable persisted blob must not override a refreshed runtime blob.
			direct := NewDirect(memoryCredentials{credential: accounts.NativeCredential{UserBlob: []byte("stale")}},
				func(context.Context, string) (string, error) { return region, nil })
			direct.SetRuntimeDir(root)
			for _, blob := range []string{
				`{"uid":"u-first","access_token":"test-first"}`,
				`{"uid":"u-refreshed","access_token":"test-refreshed"}`,
			} {
				if err := os.WriteFile(filepath.Join(authDir, "user"), []byte(blob), 0o600); err != nil {
					t.Fatal(err)
				}
				prepared, err := direct.prepare(context.Background(), "acc", translate.ChatRequest{Model: "auto"})
				if err != nil {
					t.Fatal(err)
				}
				user, err := DecryptCLIUser([]byte(blob), "0123456789abcdef")
				if err != nil {
					t.Fatal(err)
				}
				if prepared.request.Header.Get("Cosy-User") != user.uid() {
					t.Fatal("direct chat used stale credentials")
				}
			}
			if err := os.WriteFile(filepath.Join(authDir, "user"), []byte("invalid-current-credential"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := direct.prepare(context.Background(), "acc", translate.ChatRequest{Model: "auto"}); err == nil {
				t.Fatal("invalid runtime credentials must not silently use stale credentials")
			}
		})
	}
}

func TestDirectFallsBackToStoreBeforeRuntimeHomeExists(t *testing.T) {
	direct := NewDirect(memoryCredentials{credential: accounts.NativeCredential{
		UserBlob: []byte(`{"uid":"stored-user","access_token":"test-stored"}`), MachineID: "0123456789abcdef",
	}}, nil)
	direct.SetRuntimeDir(t.TempDir())
	prepared, err := direct.prepare(context.Background(), "acc", translate.ChatRequest{Model: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.request.Header.Get("Cosy-User") != "stored-user" {
		t.Fatal("stored credential fallback did not work")
	}
}
