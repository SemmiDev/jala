// Package node owns the libp2p Host and all peer-discovery mechanisms.
//
// It is kept separate from the chat package so that the two concerns —
// "how do I connect to peers?" and "what do I say to them?" — stay decoupled.
// A future version could add file transfer or video without touching this package.
package node

import (
	"context"
	"fmt"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("jala/node")

// Config holds all parameters needed to construct a Node.
type Config struct {
	PrivKey        crypto.PrivKey // required — use identity.Load()
	ListenAddrs    []string       // defaults used if empty
	RendezvousNS   string         // DHT rendezvous namespace
	EnableMDNS     bool
	EnableDHT      bool
	BootstrapPeers []string
}

// DefaultConfig returns a production-sensible Config with both
// mDNS and DHT enabled.
func DefaultConfig(priv crypto.PrivKey) Config {
	return Config{
		PrivKey: priv,
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip6/::/tcp/0",
		},
		RendezvousNS:   "jala-rendezvous-v1",
		EnableMDNS:     true,
		EnableDHT:      true,
		BootstrapPeers: ipfsBootstrapPeers(),
	}
}

// Node wraps a libp2p Host with peer discovery already running.
type Node struct {
	Host host.Host

	cfg     Config
	dhtNode *dht.IpfsDHT
	mdnsSvc mdns.Service

	stopOnce sync.Once
	stopCh   chan struct{}
}

// New builds the libp2p host and starts peer discovery.
// Call Close() when done to release resources.
func New(ctx context.Context, cfg Config) (*Node, error) {
	h, err := buildHost(cfg)
	if err != nil {
		return nil, fmt.Errorf("node: build host: %w", err)
	}

	n := &Node{
		Host:   h,
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}

	if cfg.EnableDHT {
		if err := n.startDHT(ctx); err != nil {
			_ = h.Close()
			return nil, fmt.Errorf("node: DHT: %w", err)
		}
		go n.bootstrapPeers(ctx)
		go n.advertiseAndDiscover(ctx)
	}

	if cfg.EnableMDNS {
		if err := n.startMDNS(ctx); err != nil {
			// mDNS failure is non-fatal; common in containers / CI
			log.Warnf("mDNS unavailable: %v", err)
		}
	}

	log.Infof("node ready — PeerID: %s", h.ID())
	for _, a := range h.Addrs() {
		log.Infof("  ↳ %s/p2p/%s", a, h.ID())
	}
	return n, nil
}

// Close shuts down discovery and the host.
func (n *Node) Close() {
	n.stopOnce.Do(func() {
		if n.mdnsSvc != nil {
			_ = n.mdnsSvc.Close()
		}
		if n.dhtNode != nil {
			_ = n.dhtNode.Close()
		}
		_ = n.Host.Close()
		close(n.stopCh)
		log.Info("node closed")
	})
}

// ── Host construction ─────────────────────────────────────────────────────────

func buildHost(cfg Config) (host.Host, error) {
	cm, err := connmgr.NewConnManager(20, 100,
		connmgr.WithGracePeriod(time.Minute))
	if err != nil {
		return nil, err
	}

	listenMAs := make([]multiaddr.Multiaddr, 0, len(cfg.ListenAddrs))
	for _, a := range cfg.ListenAddrs {
		ma, err := multiaddr.NewMultiaddr(a)
		if err != nil {
			return nil, fmt.Errorf("bad listen addr %q: %w", a, err)
		}
		listenMAs = append(listenMAs, ma)
	}

	return libp2p.New(
		libp2p.Identity(cfg.PrivKey),
		libp2p.ListenAddrs(listenMAs...),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.ConnectionManager(cm),
		libp2p.EnableRelay(),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.UserAgent("jala/1.0.0"),
	)
}

// ── DHT ───────────────────────────────────────────────────────────────────────

func (n *Node) startDHT(ctx context.Context) error {
	d, err := dht.New(ctx, n.Host,
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix("/jala"),
	)
	if err != nil {
		return err
	}
	if err := d.Bootstrap(ctx); err != nil {
		return fmt.Errorf("DHT bootstrap: %w", err)
	}
	n.dhtNode = d
	log.Info("DHT started (mode=auto)")
	return nil
}

func (n *Node) bootstrapPeers(ctx context.Context) {
	var wg sync.WaitGroup
	for _, addr := range n.cfg.BootstrapPeers {
		addr := addr
		wg.Add(1)
		go func() {
			defer wg.Done()
			pi, err := peer.AddrInfoFromString(addr)
			if err != nil {
				log.Debugf("bootstrap: bad addr %q: %v", addr, err)
				return
			}
			if err := n.Host.Connect(ctx, *pi); err != nil {
				log.Debugf("bootstrap: %s: %v", pi.ID, err)
				return
			}
			log.Debugf("bootstrap: connected %s", pi.ID)
		}()
	}
	wg.Wait()
	log.Info("bootstrap complete")
}

func (n *Node) advertiseAndDiscover(ctx context.Context) {
	rd := drouting.NewRoutingDiscovery(n.dhtNode)
	dutil.Advertise(ctx, rd, n.cfg.RendezvousNS)
	log.Infof("advertising rendezvous %q", n.cfg.RendezvousNS)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peerCh, err := rd.FindPeers(ctx, n.cfg.RendezvousNS)
			if err != nil {
				log.Debugf("FindPeers: %v", err)
				continue
			}
			for pi := range peerCh {
				if pi.ID == n.Host.ID() || len(pi.Addrs) == 0 {
					continue
				}
				go func(p peer.AddrInfo) {
					// Use an independent context with timeout so a
					// cancelled outer ctx doesn't kill in-flight connects.
					cctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()
					if err := n.Host.Connect(cctx, p); err != nil {
						log.Debugf("connect %s: %v", p.ID, err)
					}
				}(pi)
			}
		}
	}
}

// ── mDNS ──────────────────────────────────────────────────────────────────────

func (n *Node) startMDNS(ctx context.Context) error {
	svc := mdns.NewMdnsService(n.Host, n.cfg.RendezvousNS, &mdnsNotifee{
		host: n.Host,
		ctx:  ctx,
	})
	if err := svc.Start(); err != nil {
		return err
	}
	n.mdnsSvc = svc
	log.Infof("mDNS started (tag=%q)", n.cfg.RendezvousNS)
	return nil
}

type mdnsNotifee struct {
	host host.Host
	ctx  context.Context
}

func (m *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Debugf("mDNS found: %s", pi.ID)
	if err := m.host.Connect(m.ctx, pi); err != nil {
		log.Debugf("mDNS connect %s: %v", pi.ID, err)
	}
}

// ── Bootstrap peers ───────────────────────────────────────────────────────────

func ipfsBootstrapPeers() []string {
	return []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}
}
