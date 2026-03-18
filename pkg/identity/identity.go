// Package identity manages the node's cryptographic identity.
//
// A libp2p PeerID is derived directly from an Ed25519 public key —
// no certificate authority, no registration. The private key IS the identity.
//
// If a key file path is provided:
//   - existing file  → load and reuse (stable PeerID across restarts)
//   - missing file   → generate, persist, then use
//
// If no path is given, a fresh ephemeral key is generated each run.
package identity

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Identity holds the private key and the PeerID derived from it.
type Identity struct {
	PrivKey crypto.PrivKey
	PeerID  peer.ID
}

// Load loads an identity from path, or generates a fresh ephemeral one
// if path is empty. If path is set but the file does not yet exist,
// a new key is generated and persisted there.
func Load(path string) (*Identity, error) {
	priv, err := loadOrCreate(path)
	if err != nil {
		return nil, err
	}

	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("identity: derive PeerID: %w", err)
	}

	return &Identity{PrivKey: priv, PeerID: pid}, nil
}

// ShortID returns the first 12 characters of the PeerID string —
// enough to identify a peer in logs and UI without visual clutter.
func (id *Identity) ShortID() string {
	s := id.PeerID.String()
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}

func loadOrCreate(path string) (crypto.PrivKey, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			priv, err := crypto.UnmarshalPrivateKey(data)
			if err != nil {
				return nil, fmt.Errorf("identity: unmarshal key %s: %w", path, err)
			}
			return priv, nil
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("identity: read key %s: %w", path, err)
		}
	}

	// Ed25519: 32-byte key, fast, deterministic signatures.
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("identity: generate key: %w", err)
	}

	if path != "" {
		raw, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return nil, fmt.Errorf("identity: marshal key: %w", err)
		}
		// 0600 — private key readable only by owner
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			// Non-fatal: warn and continue with ephemeral key
			fmt.Fprintf(os.Stderr, "warn: could not persist key to %s: %v\n", path, err)
		}
	}

	return priv, nil
}
