package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSessionSecret(t *testing.T) {
	dir := t.TempDir()

	// Güçlü özel değer olduğu gibi kullanılır
	custom := "0123456789abcdef0123456789abcdef-custom"
	if got := resolveSessionSecret(custom, dir); got != custom {
		t.Fatalf("custom secret not used")
	}

	// Bilinen varsayılan değer reddedilir, rastgele anahtar üretilip kaydedilir
	got := resolveSessionSecret("change-this-to-a-strong-random-secret", dir)
	if knownDefaultSecrets[got] || len(got) < 32 {
		t.Fatalf("default secret must be replaced, got %q", got)
	}
	info, err := os.Stat(filepath.Join(dir, "session.key"))
	if err != nil {
		t.Fatalf("session.key not written: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("session.key permissions = %v, want 0600", info.Mode().Perm())
	}

	// Yeniden başlatmada aynı anahtar okunur (oturumlar korunur)
	if again := resolveSessionSecret("", dir); again != got {
		t.Fatalf("persisted secret not reused")
	}
}
