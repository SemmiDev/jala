package identity_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/semmidev/jala/pkg/identity"
)

func TestLoadEphemeral(t *testing.T) {
	// Empty path → ephemeral key, no file written
	id, err := identity.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") error: %v", err)
	}
	if id.PrivKey == nil {
		t.Error("PrivKey should not be nil")
	}
	if id.PeerID == "" {
		t.Error("PeerID should not be empty")
	}
}

func TestLoadAndPersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "identity.key")

	// First load: generates and saves
	id1, err := identity.Load(path)
	if err != nil {
		t.Fatalf("first Load: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("key file should have been created")
	}

	// Second load: reads from file — PeerID must match
	id2, err := identity.Load(path)
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}

	if id1.PeerID != id2.PeerID {
		t.Errorf("PeerID changed across loads: %s != %s", id1.PeerID, id2.PeerID)
	}
}

func TestShortID(t *testing.T) {
	id, err := identity.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	short := id.ShortID()
	full := id.PeerID.String()

	if len(short) > len(full) {
		t.Errorf("ShortID longer than full ID")
	}
	if len(full) > 12 && len(short) != 12 {
		t.Errorf("ShortID should be 12 chars for long IDs, got %d", len(short))
	}
}

func TestTwoEphemeralIDsAreDifferent(t *testing.T) {
	id1, _ := identity.Load("")
	id2, _ := identity.Load("")
	if id1.PeerID == id2.PeerID {
		t.Error("two ephemeral identities should have different PeerIDs")
	}
}
