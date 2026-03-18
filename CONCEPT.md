# Jala — Penjelasan Lengkap dari Konsep hingga Kode

> Dokumen ini menjelaskan setiap lapisan proyek **jala** secara berurutan:
> dari "apa itu P2P" hingga baris kode terkecil. Tidak ada asumsi pengetahuan
> sebelumnya tentang jaringan atau Go.

---

## Daftar Isi

1. [Apa Itu P2P dan Mengapa Susah?](#1-apa-itu-p2p-dan-mengapa-susah)
2. [Bagaimana libp2p Memecahkan Masalah Itu](#2-bagaimana-libp2p-memecahkan-masalah-itu)
3. [Gambaran Besar Proyek](#3-gambaran-besar-proyek)
4. [Layer 1 — Identitas: `pkg/identity/identity.go`](#4-layer-1--identitas)
5. [Layer 2 — Node & Transport: `internal/node/node.go`](#5-layer-2--node--transport)
6. [Layer 3 — Pesan & Room: `internal/chat/`](#6-layer-3--pesan--room)
7. [Layer 4 — UI: `internal/ui/`](#7-layer-4--ui)
8. [Titik Masuk: `cmd/jala/main.go`](#8-titik-masuk-cmdjala)
9. [Alur Data Lengkap dari Ketik hingga Terkirim](#9-alur-data-lengkap)
10. [Pola Go yang Digunakan dan Alasannya](#10-pola-go-yang-digunakan)

---

## 1. Apa Itu P2P dan Mengapa Susah?

### Model biasa: Client-Server

Ketika kamu membuka WhatsApp dan mengirim pesan, alurnya seperti ini:

```
Kamu ──→ Server WhatsApp ──→ Temanmu
```

Server WhatsApp adalah perantara. Dia menyimpan pesanmu, memverifikasi
identitasmu, dan memastikan pesan sampai ke tujuan. Modelnya sederhana,
tapi ada beberapa konsekuensi: server bisa mati, server bisa membaca
pesanmu, dan tanpa izin server kamu tidak bisa berkomunikasi.

### Model P2P: Tanpa Perantara

P2P (peer-to-peer) berarti dua komputer berkomunikasi **langsung**:

```
Kamu ──────────────────→ Temanmu
       (koneksi langsung)
```

Terdengar lebih simpel, tapi dalam praktiknya ada tiga masalah besar yang
harus dipecahkan sebelum ini bisa bekerja:

**Masalah 1 — Identitas:** Bagaimana kamu tahu bahwa yang kamu ajak bicara
benar-benar "temanmu" dan bukan orang lain yang mengaku? Di WhatsApp, server
yang menjamin ini. Di P2P, tidak ada server — jadi kita butuh cara lain.

**Masalah 2 — Penemuan (Discovery):** Bagaimana dua komputer saling tahu
alamat IP satu sama lain? Di internet nyata, sebagian besar komputer berada
di balik NAT (router WiFi rumahmu), artinya mereka tidak punya alamat IP
publik yang bisa diakses langsung dari luar.

**Masalah 3 — Komunikasi:** Setelah terhubung, bagaimana pesan dikirim ke
*banyak* orang sekaligus? Dan bagaimana memastikan pesan tidak disadap
di tengah jalan?

Itulah tiga masalah yang proyek jala — dan libp2p di bawahnya — pecahkan.

---

## 2. Bagaimana libp2p Memecahkan Masalah Itu

libp2p adalah *networking stack* modular yang dibuat oleh tim IPFS (InterPlanetary
File System). Bayangkan dia sebagai "toolkit" berisi solusi untuk semua masalah
P2P di atas:

Untuk **identitas**, libp2p menggunakan kriptografi kunci publik. Setiap node
membuat sepasang kunci — kunci privat yang dirahasiakan dan kunci publik yang
dibagikan. PeerID (alamat unik setiap node di jaringan) diturunkan *langsung*
dari kunci publik ini. Artinya: tidak perlu server registrasi, identitasmu
*adalah* kuncimu.

Untuk **penemuan**, libp2p menyediakan dua mekanisme yang bekerja bersamaan.
mDNS (multicast DNS) untuk penemuan di jaringan lokal — seperti bagaimana
printer WiFi ditemukan otomatis oleh laptopmu. Dan Kademlia DHT untuk penemuan
global di internet — sebuah sistem terdistribusi yang memungkinkan jutaan node
saling menemukan tanpa server pusat.

Untuk **komunikasi**, libp2p menyediakan transport terenkripsi (Noise, TLS),
multiplexing (banyak "percakapan" di satu koneksi), dan GossipSub untuk
broadcast pesan ke banyak node sekaligus.

---

## 3. Gambaran Besar Proyek

Sebelum masuk ke kode, penting untuk memahami **bagaimana file-file proyek
ini disusun dan mengapa**.

```
jala/
├── cmd/
│   └── jala/
│       └── main.go          ← "Titik masuk" — program dimulai di sini
│
├── internal/                ← Kode yang HANYA boleh dipakai oleh proyek ini
│   ├── node/
│   │   └── node.go          ← Lapisan jaringan: koneksi, discovery
│   ├── chat/
│   │   ├── message.go       ← Format pesan
│   │   └── room.go          ← Manajemen chat room
│   └── ui/
│       ├── app.go           ← Logika utama tampilan
│       ├── chatview.go      ← Panel pesan & daftar peer
│       └── styles.go        ← Warna dan gaya teks
│
└── pkg/
    └── identity/
        └── identity.go      ← Manajemen kunci kriptografi
```

Folder `internal/` adalah konvensi Go: package di dalamnya tidak bisa
di-import oleh kode di luar proyek ini. Ini memastikan batas-batas
antara lapisan tetap bersih.

Folder `pkg/` berisi kode yang *bisa* dibagikan ke proyek lain — dalam
hal ini, `identity` bisa berguna di aplikasi lain yang butuh manajemen
kunci libp2p.

Cara paling mudah membayangkan proyek ini adalah seperti tumpukan lapisan,
di mana setiap lapisan tidak tahu detail implementasi lapisan di bawahnya:

```
┌─────────────────────────────────────┐
│          main.go                    │  ← "Konduktor": menghubungkan semua
├─────────────────────────────────────┤
│          ui/ (Bubble Tea TUI)       │  ← "Wajah": apa yang kamu lihat
├─────────────────────────────────────┤
│          chat/ (Room + Message)     │  ← "Bahasa": format dan aturan percakapan
├─────────────────────────────────────┤
│          node/ (libp2p Host)        │  ← "Pipa": koneksi dan jaringan
├─────────────────────────────────────┤
│          identity/ (Ed25519 Key)    │  ← "KTP": siapa kamu di jaringan
└─────────────────────────────────────┘
```

---

## 4. Layer 1 — Identitas

**File:** `pkg/identity/identity.go`

### Konsep: Kunci sebagai Identitas

Di dunia nyata, identitas kamu diverifikasi oleh pihak ketiga — KTP
diterbitkan oleh pemerintah, akun Twitter diverifikasi oleh Twitter.
Di P2P, tidak ada pihak ketiga. Solusinya: gunakan kriptografi.

Kita membuat sepasang kunci matematika yang sifatnya unik:
- **Kunci Privat**: seperti tanda tangan kamu — rahasia, hanya kamu punya
- **Kunci Publik**: seperti stempel cap — boleh dibagikan ke siapapun

PeerID (ID unikmu di jaringan P2P) dihitung langsung dari kunci publik
menggunakan fungsi hash kriptografi. Hasilnya terlihat seperti:
`12D3KooWNzeutNDuGHZSDgqpT9NLt8jsGZLiqALBKi14ABdFVrHG`

Ini berarti: selama kamu menyimpan kunci privat yang sama, PeerIDmu akan
selalu sama — di komputer manapun, kapanpun.

Algoritma yang digunakan adalah **Ed25519**, sebuah algoritma tanda tangan
digital modern. Alasan memilih Ed25519: kuncinya hanya 32 byte (sangat kecil),
proses penandatanganan sangat cepat, dan sudah dianggap aman oleh komunitas
kriptografi internasional.

### Kode

```go
// Identity holds the private key and the PeerID derived from it.
type Identity struct {
    PrivKey crypto.PrivKey  // kunci privat — JANGAN dibagikan
    PeerID  peer.ID         // ID publik — boleh dibagikan ke siapapun
}
```

Struct `Identity` hanya menyimpan dua hal: kunci privat dan PeerID yang
diturunkan darinya. Keduanya selalu berpasangan — tidak ada PeerID tanpa
kunci privat yang menghasilkannya.

```go
func Load(path string) (*Identity, error) {
    priv, err := loadOrCreate(path)
    if err != nil {
        return nil, err
    }

    // PeerID diturunkan dari kunci — bukan disimpan terpisah
    pid, err := peer.IDFromPrivateKey(priv)
    if err != nil {
        return nil, fmt.Errorf("identity: derive PeerID: %w", err)
    }

    return &Identity{PrivKey: priv, PeerID: pid}, nil
}
```

Fungsi `Load` adalah *satu-satunya* cara membuat `Identity`. Dia menerima
`path` — lokasi file untuk menyimpan kunci. Jika `path` kosong (`""`), kunci
dibuat baru setiap kali program jalan (ephemeral/sementara). Jika `path`
diisi, kunci disimpan ke file dan dimuat ulang saat program restart — ini
yang membuat PeerIDmu tetap sama antar sesi.

```go
func loadOrCreate(path string) (crypto.PrivKey, error) {
    if path != "" {
        data, err := os.ReadFile(path)
        if err == nil {
            // File ada → muat kunci yang sudah ada
            priv, err := crypto.UnmarshalPrivateKey(data)
            ...
            return priv, nil
        }
        if !os.IsNotExist(err) {
            // Error selain "file tidak ada" → hentikan dengan error
            return nil, fmt.Errorf("identity: read key %s: %w", path, err)
        }
    }

    // Sampai sini berarti: path kosong, ATAU path diisi tapi file belum ada
    // → Generate kunci baru
    priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
    ...

    if path != "" {
        // Simpan ke file dengan permission 0600
        // 0600 artinya: hanya pemilik file yang bisa baca/tulis
        // Orang lain di komputer yang sama tidak bisa membacanya
        if err := os.WriteFile(path, raw, 0o600); err != nil {
            // Gagal simpan bukan error fatal — kita tetap lanjut
            fmt.Fprintf(os.Stderr, "warn: could not persist key...")
        }
    }

    return priv, nil
}
```

Perhatikan pola `if !os.IsNotExist(err)` — ini adalah cara idiomatik Go
untuk membedakan antara "file tidak ada (wajar)" vs "ada error lain seperti
permission denied (tidak wajar)". Pola ini mencegah kita menimpa kunci yang
ada karena kita salah membaca error.

```go
func (id *Identity) ShortID() string {
    s := id.PeerID.String()
    if len(s) <= 12 {
        return s
    }
    return s[:12]
}
```

`ShortID()` hanya mengambil 12 karakter pertama dari PeerID. PeerID penuh
sangat panjang dan tidak praktis ditampilkan di UI. 12 karakter sudah cukup
untuk membedakan peer satu dengan lainnya di antara peer yang sedikit.

---

## 5. Layer 2 — Node & Transport

**File:** `internal/node/node.go`

### Konsep: Host sebagai "Telepon"

Bayangkan `node.Node` seperti sebuah telepon. Telepon itu punya:
- **Nomor telepon** (PeerID — identitas kita di jaringan)
- **Kemampuan menerima dan melakukan panggilan** (transport: TCP dan QUIC)
- **Enkripsi percakapan** (Noise dan TLS)
- **Buku telepon** (peer discovery: mDNS dan DHT)
- **Batas berapa banyak panggilan bisa aktif bersamaan** (Connection Manager)

`host.Host` adalah abstraksi utama di libp2p. Semua komunikasi melaluinya.

### Konfigurasi

```go
type Config struct {
    PrivKey        crypto.PrivKey  // kunci identitas kita
    ListenAddrs    []string        // di alamat mana kita "menunggu koneksi masuk"
    RendezvousNS   string          // "kata sandi" untuk menemukan sesama pengguna jala
    EnableMDNS     bool            // aktifkan discovery lokal
    EnableDHT      bool            // aktifkan discovery global
    BootstrapPeers []string        // "pintu masuk" ke jaringan DHT
}
```

`RendezvousNS` adalah konsep yang menarik. Bayangkan seperti hashtag di Twitter.
Semua node yang menjalankan jala akan menggunakan hashtag `/jala/rendezvous/v1`.
Di DHT, mereka "mengiklankan" diri dengan hashtag ini, dan secara berkala
mencari siapa lagi yang menggunakan hashtag yang sama. Ini memungkinkan dua
instance jala di belahan dunia berbeda untuk saling menemukan.

```go
func DefaultConfig(priv crypto.PrivKey) Config {
    return Config{
        PrivKey: priv,
        ListenAddrs: []string{
            "/ip4/0.0.0.0/tcp/0",       // TCP di semua interface IPv4, port acak
            "/ip4/0.0.0.0/udp/0/quic-v1", // QUIC di semua interface IPv4, port acak
            "/ip6/::/tcp/0",             // TCP di semua interface IPv6
        },
        RendezvousNS:   "/jala/rendezvous/v1",
        EnableMDNS:     true,
        EnableDHT:      true,
        BootstrapPeers: ipfsBootstrapPeers(),
    }
}
```

Format `/ip4/0.0.0.0/tcp/0` adalah **Multiaddr** — format alamat universal
yang digunakan libp2p. `0.0.0.0` berarti "semua interface jaringan" dan port
`0` berarti "minta OS untuk pilih port kosong". Ini idiomatik untuk aplikasi
P2P karena kita tidak perlu tahu port berapa yang tersedia.

### Membangun Host

```go
func buildHost(cfg Config) (host.Host, error) {
    // Connection Manager: jaga jumlah koneksi antara 20 dan 100
    // Jika > 100, mulai tutup koneksi yang "kurang penting"
    // Jika sudah < 20, berhenti menutup
    cm, err := connmgr.NewConnManager(20, 100,
        connmgr.WithGracePeriod(time.Minute))
    ...

    return libp2p.New(
        libp2p.Identity(cfg.PrivKey),        // "ini nomor telepon saya"
        libp2p.ListenAddrs(listenMAs...),     // "di sini saya bisa dihubungi"

        // Transport: cara fisik mengirim data
        libp2p.Transport(tcp.NewTCPTransport),  // TCP — universal, reliable
        libp2p.Transport(quic.NewTransport),    // QUIC — modern, lebih cepat

        // Security: enkripsi semua komunikasi
        libp2p.Security(noise.ID, noise.New),      // Noise — libp2p-native
        libp2p.Security(libp2ptls.ID, libp2ptls.New), // TLS — fallback

        libp2p.ConnectionManager(cm),
        libp2p.EnableRelay(),          // izinkan routing lewat peer lain (untuk NAT)
        libp2p.EnableNATService(),     // deteksi apakah kita di balik NAT
        libp2p.EnableHolePunching(),   // coba buka jalur langsung meski di balik NAT
        libp2p.UserAgent("jala/1.0.0"), // identifikasi diri ke peer lain
    )
}
```

Kenapa ada dua transport (TCP dan QUIC)? TCP adalah protokol yang sudah
ada sejak 1981 — berjalan di mana-mana, reliabel, tapi ada overhead karena
harus *three-way handshake* untuk membuka koneksi. QUIC adalah protokol
modern (dikembangkan Google, sekarang jadi standar RFC) yang berjalan di
atas UDP. QUIC lebih cepat membuka koneksi baru (hanya 1 *round-trip*
vs 3 untuk TCP+TLS), tapi belum didukung semua jaringan. Dengan mendukung
keduanya, jala akan otomatis pilih yang terbaik.

Kenapa ada dua security (Noise dan TLS)? Noise adalah protokol keamanan
yang dirancang *khusus* untuk P2P — tidak butuh sertifikat CA, lebih
sederhana. TLS adalah protokol yang digunakan HTTPS — lebih umum dikenal
dan dibutuhkan untuk interoperabilitas dengan sistem lain. libp2p akan
*negotiate* protokol mana yang digunakan saat koneksi dibuat.

### Penemuan Peer (Discovery)

**mDNS** bekerja di jaringan lokal. Saat kamu menjalankan dua instance
jala di komputer berbeda tapi satu WiFi yang sama, mDNS akan menemukan
mereka satu sama lain dalam hitungan detik tanpa konfigurasi apapun.

```go
func (n *Node) startMDNS(ctx context.Context) error {
    // mdnsNotifee adalah "pendengar" — dia dipanggil setiap kali
    // ada peer baru yang ditemukan via mDNS
    svc := mdns.NewMdnsService(n.Host, n.cfg.RendezvousNS, &mdnsNotifee{
        host: n.Host,
        ctx:  ctx,
    })
    ...
}

// HandlePeerFound dipanggil secara otomatis oleh library mDNS
// setiap kali peer baru ketahuan di jaringan lokal
func (m *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
    log.Debugf("mDNS found: %s", pi.ID)
    // Langsung coba konek
    if err := m.host.Connect(m.ctx, pi); err != nil {
        log.Debugf("mDNS connect %s: %v", pi.ID, err)
    }
}
```

`peer.AddrInfo` adalah struktur yang berisi PeerID dan daftar alamat (multiaddr)
di mana peer bisa dihubungi. Analoginya: nama orang + nomor telepon mereka.

**DHT** (Kademlia Distributed Hash Table) adalah sistem yang lebih kompleks
untuk penemuan global. Ide dasarnya: semua node di jaringan membentuk sebuah
"kamus terdistribusi" yang bisa dicari. Kita menyimpan entri "saya ada di sini"
di kamus ini, dan peer lain bisa mencarinya.

```go
func (n *Node) advertiseAndDiscover(ctx context.Context) {
    rd := drouting.NewRoutingDiscovery(n.dhtNode)

    // "Iklankan" diri kita — simpan entri di DHT:
    // "node dengan RendezvousNS ini ada di sini"
    dutil.Advertise(ctx, rd, n.cfg.RendezvousNS)

    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done(): // program selesai → berhenti
            return
        case <-ticker.C: // setiap 10 detik → cari peer baru
            peerCh, err := rd.FindPeers(ctx, n.cfg.RendezvousNS)
            ...
            for pi := range peerCh {
                // Lewati diri sendiri dan peer tanpa alamat
                if pi.ID == n.Host.ID() || len(pi.Addrs) == 0 {
                    continue
                }
                // Coba konek ke peer yang ditemukan (di goroutine terpisah
                // agar tidak memblok pencarian berikutnya)
                go func(p peer.AddrInfo) {
                    n.Host.Connect(ctx, p)
                }(pi)
            }
        }
    }
}
```

Pola `select { case <-ctx.Done() ... case <-ticker.C ... }` adalah idiom
Go yang sangat umum untuk loop dengan *timeout* atau *cancellation*. `ctx.Done()`
adalah channel yang akan "ditutup" ketika konteks dibatalkan (misalnya saat
user tekan Ctrl+C). `ticker.C` adalah channel yang menerima "sinyal" setiap
10 detik. `select` menunggu salah satu dari keduanya terjadi.

Bootstrap peers adalah "pintu masuk" ke jaringan DHT. Sebelum kita bisa mencari
peer di DHT, kita perlu terhubung ke setidaknya satu node yang sudah ada di
jaringan. IPFS menyediakan beberapa node bootstrap publik yang selalu online
untuk tujuan ini.

```go
func ipfsBootstrapPeers() []string {
    return []string{
        "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoT...",
        // ...
    }
}
```

Format `/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7...` adalah multiaddr
yang menggabungkan DNS hostname dan PeerID. libp2p akan resolve DNS tersebut
ke IP aktual, lalu konek ke node dengan PeerID yang disebutkan.

### Graceful Shutdown dengan `sync.Once`

```go
func (n *Node) Close() {
    // sync.Once memastikan kode di dalamnya hanya berjalan SEKALI
    // meski Close() dipanggil berkali-kali dari goroutine berbeda
    n.stopOnce.Do(func() {
        if n.mdnsSvc != nil { _ = n.mdnsSvc.Close() }
        if n.dhtNode != nil { _ = n.dhtNode.Close() }
        _ = n.Host.Close()
        close(n.stopCh)
    })
}
```

`sync.Once` adalah primitif sinkronisasi di Go yang menjamin sebuah fungsi
hanya dipanggil satu kali, tidak peduli berapa goroutine yang memanggil
`Close()` secara bersamaan. Ini penting karena kita memanggil `defer n.Close()`
di `main()` sekaligus mungkin ada signal handler yang memanggil shutdown
dari goroutine lain.

---

## 6. Layer 3 — Pesan & Room

**File:** `internal/chat/message.go` dan `internal/chat/room.go`

### Konsep: GossipSub sebagai "Megafon Jaringan"

Bayangkan kamu di sebuah aula besar dengan banyak orang. Ada dua cara
menyampaikan pesan:
1. **Berbisik langsung** ke telinga satu orang (direct stream — seperti di `p2p-node` sebelumnya)
2. **Menggunakan megafon** sehingga semua orang di aula mendengar (GossipSub)

Untuk chat room, kita butuh cara kedua. GossipSub adalah implementasi
"megafon" ini di libp2p.

Cara kerja GossipSub secara sederhana: setiap node yang bergabung ke satu
topik (topic = channel = room) membentuk *mesh* dengan beberapa peer lain
(biasanya 6). Ketika kamu publish pesan, pesan diteruskan ke 6 peer tersebut.
Mereka meneruskan ke peer mereka, dan seterusnya sampai semua orang mendapat
pesan — seperti cara gosip menyebar di masyarakat.

### Format Pesan

```go
// MsgType adalah "tipe" pesan — apa jenis event ini?
type MsgType string

const (
    MsgChat  MsgType = "chat"   // pesan teks biasa dari user
    MsgJoin  MsgType = "join"   // seseorang masuk ke room
    MsgLeave MsgType = "leave"  // seseorang keluar dari room
)
```

Kenapa tidak langsung kirim string teks saja? Karena room chat butuh
lebih dari sekadar pesan teks — kita juga perlu tahu siapa yang join
dan leave, kapan pesan dikirim, dari siapa. Dengan mendefinisikan
`MsgType`, kita bisa menangani setiap jenis event dengan cara berbeda
di UI.

```go
type Message struct {
    Type      MsgType   `json:"type"`
    SenderID  string    `json:"sender_id"`  // PeerID penuh pengirim
    Nick      string    `json:"nick"`        // nama tampilan pengirim
    Body      string    `json:"body"`        // isi pesan (kosong untuk join/leave)
    Timestamp time.Time `json:"ts"`
}
```

Tag `json:"type"` dll. adalah instruksi untuk encoder/decoder JSON Go.
Mereka menentukan nama field di JSON (huruf kecil, lebih ringkas) vs nama
field di struct Go (huruf besar, idiomatik). Misalnya field `SenderID`
akan menjadi `"sender_id"` di JSON.

```go
func NewChat(senderID, nick, body string) Message {
    return Message{
        Type:      MsgChat,
        SenderID:  senderID,
        Nick:      nick,
        Body:      body,
        Timestamp: time.Now().UTC(), // selalu simpan waktu dalam UTC
    }
}
```

Constructor function seperti `NewChat`, `NewJoin`, `NewLeave` memastikan
Message selalu dibuat dengan field yang benar — tidak mungkin lupa mengisi
`Timestamp` atau salah mengisi `Type`. Ini adalah pola idiomatik Go.

```go
func (m Message) Encode() ([]byte, error) {
    data, err := json.Marshal(m)
    ...
    return data, nil
}

func Decode(data []byte) (Message, error) {
    var m Message
    if err := json.Unmarshal(data, &m); err != nil {
        ...
    }
    return m, nil
}
```

`Encode()` mengubah struct `Message` menjadi bytes JSON untuk dikirim
lewat GossipSub. `Decode()` membalikkanya — dari bytes JSON ke struct `Message`.
GossipSub tidak tahu apa-apa tentang format pesan kita; dia hanya
mengirim `[]byte` (slice of bytes). Kita yang menentukan format tersebut.

### Manajemen Room

`Room` adalah abstraksi utama di layer chat. Dia menyembunyikan semua
kompleksitas GossipSub di balik API yang sederhana: `Join()`, `Send()`,
`Close()`.

```go
type Room struct {
    // Messages adalah channel yang DIBACA oleh UI
    // UI tidak perlu tahu tentang GossipSub — dia hanya baca dari channel ini
    Messages <-chan Message  // <-chan artinya: hanya bisa dibaca, tidak bisa ditulis

    // Field-field berikut bersifat PRIVAT (huruf kecil)
    // Artinya hanya kode di package `chat` yang bisa mengaksesnya
    host     host.Host
    ps       *pubsub.PubSub
    topic    *pubsub.Topic
    sub      *pubsub.Subscription
    roomName string
    selfID   peer.ID
    nick     string
    msgs     chan Message  // ini yang DITULIS oleh readLoop, menjadi Messages di atas
    once     sync.Once
}
```

Perhatikan perbedaan antara `Messages <-chan Message` (publik, read-only)
dan `msgs chan Message` (privat, read-write). Ini adalah pola enkapsulasi
yang idiomatik di Go. Dari luar, kode hanya bisa *membaca* dari `Messages`.
Tapi dari dalam package `chat`, kode bisa *menulis* ke `msgs` (yang sama
dengan `Messages` tapi tanpa pembatasan arah).

```go
func Join(ctx context.Context, h host.Host, roomName string, nick string) (*Room, error) {
    // 1. Buat GossipSub router
    ps, err := pubsub.NewGossipSub(ctx, h,
        pubsub.WithMessageSigning(true),            // tanda tangani setiap pesan
        pubsub.WithStrictSignatureVerification(true), // tolak pesan tak bertanda tangan
        pubsub.WithFloodPublish(true),              // kirim ke semua peer langsung
    )

    // 2. Join topik GossipSub untuk room ini
    // "/jala/room/v1/lobby" adalah nama topik — unique dan namespaced
    topicName := topicForRoom(roomName)  // → "/jala/room/v1/lobby"
    topic, err := ps.Join(topicName)

    // 3. Subscribe untuk menerima pesan
    sub, err := topic.Subscribe()

    // 4. Buat channel buffer 64 untuk pesan masuk
    // Buffer 64 artinya: bisa menampung 64 pesan sebelum pengirim perlu menunggu
    msgs := make(chan Message, 64)

    r := &Room{
        Messages: msgs, // publik (read-only)
        msgs:     msgs, // privat (read-write)
        ...
    }

    // 5. Mulai goroutine yang membaca dari GossipSub → menulis ke msgs
    go r.readLoop(ctx)

    // 6. Umumkan kehadiran kita ke room
    r.Publish(NewJoin(h.ID().String(), nick))

    return r, nil
}
```

Kenapa `msgs := make(chan Message, 64)` dan bukan unbuffered? Jika unbuffered,
setiap kali `readLoop` mendapat pesan dari jaringan, dia harus *menunggu* UI
membacanya sebelum bisa melanjutkan. Jika UI sedang sibuk render, ini bisa
memblok goroutine jaringan. Buffer 64 memberi ruang napas — jaringan bisa
menerima hingga 64 pesan sebelum perlu menunggu UI.

```go
func (r *Room) readLoop(ctx context.Context) {
    for {
        // Tunggu pesan berikutnya dari GossipSub
        // Ini blocking — goroutine "tidur" di sini sampai ada pesan atau ctx selesai
        rawMsg, err := r.sub.Next(ctx)
        if err != nil {
            return // ctx dibatalkan atau sub ditutup → keluar dari loop
        }

        // GossipSub mengembalikan pesan kita sendiri! Kita skip.
        // (Kita echo pesan sendiri secara manual di UI)
        if rawMsg.ReceivedFrom == r.selfID {
            continue
        }

        msg, err := Decode(rawMsg.Data)
        if err != nil {
            log.Debugf("malformed message...")
            continue // pesan rusak → skip, lanjut ke berikutnya
        }

        select {
        case r.msgs <- msg:   // kirim ke channel — kalau channel penuh...
        case <-ctx.Done():    // ...atau program selesai → keluar
            return
        default:              // ...channel penuh DAN ctx belum selesai → DROP
            log.Warn("message buffer full — dropping message")
        }
    }
}
```

`select` dengan `default` di sini sangat penting: dia mencegah `readLoop`
pernah memblok lebih dari sesaat. Kalau channel `msgs` penuh (UI tidak cukup
cepat membaca), pesan baru akan di-*drop* daripada memblok goroutine jaringan.
Ini adalah trade-off yang disengaja: lebih baik kehilangan beberapa pesan
daripada membuat jaringan macet.

---

## 7. Layer 4 — UI

**File:** `internal/ui/styles.go`, `chatview.go`, dan `app.go`

### Konsep: Elm Architecture

Bubble Tea menggunakan pola yang disebut **Elm Architecture**, dinamai dari
bahasa pemrograman Elm (sebuah bahasa untuk UI web). Idenya sederhana tapi
powerful:

```
┌──────────────────────────────────────────────────┐
│                                                    │
│   Event (keypress, network msg, resize)            │
│         │                                          │
│         ▼                                          │
│   Update(Model, Event) → (Model baru, Cmd)         │
│         │                                          │
│         ▼                                          │
│   View(Model) → string yang ditampilkan            │
│                                                    │
└──────────────────────────────────────────────────┘
```

- **Model** adalah *snapshot* lengkap dari semua state UI pada satu momen.
- **Update** adalah fungsi murni: diberi Model lama + Event, menghasilkan Model
  baru. Tidak ada *side effect* langsung.
- **View** adalah fungsi murni: diberi Model, menghasilkan string yang
  akan ditampilkan. Tidak ada logika di sini.
- **Cmd** adalah "perintah" yang akan dieksekusi Bubble Tea di goroutine
  terpisah, hasilnya akan masuk sebagai Event ke Update berikutnya.

Mengapa pola ini bagus? Karena Update selalu dipanggil secara *serial* (satu
per satu) di satu goroutine. Tidak ada race condition. Tidak ada mutex.
Kamu bisa membaca Update dan *tahu pasti* state program tanpa khawatir
ada goroutine lain yang mengubahnya bersamaan.

### Styles

```go
// AdaptiveColor otomatis memilih warna berdasarkan apakah terminal
// background gelap atau terang
var (
    colorPrimary = lipgloss.AdaptiveColor{
        Light: "#5A4FCF",  // ungu gelap untuk terminal putih
        Dark:  "#9D8FFF",  // ungu muda untuk terminal hitam
    }
    colorSelf = lipgloss.AdaptiveColor{
        Light: "#0055CC",  // biru tua untuk terminal putih
        Dark:  "#8BE9FD",  // biru muda untuk terminal hitam
    }
    ...
)
```

`lipgloss.AdaptiveColor` adalah fitur Lipgloss yang secara otomatis memilih
warna berdasarkan latar belakang terminal. Ini penting karena pengguna
ada yang memakai terminal hitam, ada yang putih.

```go
// Style dibangun dengan method chaining — setiap method mengembalikan style baru
styleTitle = lipgloss.NewStyle().
    Bold(true).
    Foreground(colorPrimary).
    Padding(0, 1)  // padding vertikal 0, horizontal 1

// Style bisa "inherit" dari style lain
styleStatusKey = lipgloss.NewStyle().
    Inherit(styleStatusBar).  // ambil semua property dari statusBar
    Foreground(colorPrimary). // lalu timpa warna teks
    Bold(true)
```

Pola *method chaining* (`Bold().Foreground().Padding()`) adalah cara idiomatik
Lipgloss untuk mendefinisikan style. Setiap panggilan method menghasilkan
style baru (immutable) — tidak mengubah style sebelumnya.

### ChatView

`ChatView` bertanggung jawab menyimpan dan merender *history* pesan.

```go
type ChatView struct {
    messages []renderedMsg  // semua pesan yang pernah diterima
    selfID   string         // PeerID kita — untuk bedakan warna pesan sendiri
    width    int            // lebar panel dalam karakter
    height   int            // tinggi panel dalam baris
}

// renderedMsg menyimpan pesan ASLI dan VERSI YANG SUDAH DIRENDER
// Kenapa disimpan dua-duanya? Karena saat terminal di-resize,
// kita perlu render ulang semua pesan dengan lebar baru
type renderedMsg struct {
    raw      chat.Message
    rendered string
}
```

```go
func (v *ChatView) View() string {
    lines := make([]string, len(v.messages))
    for i, m := range v.messages {
        lines[i] = m.rendered
    }

    // Ambil hanya `height` baris terakhir (scroll to bottom)
    if len(lines) > v.height {
        lines = lines[len(lines)-v.height:]
    }

    // Tambah baris kosong di atas agar pesan "menempel" di bawah
    // (seperti WhatsApp — pesan terbaru di bawah)
    for len(lines) < v.height {
        lines = append([]string{""}, lines...)  // prepend baris kosong
    }

    return strings.Join(lines, "\n")
}
```

Ide `prepend baris kosong` mungkin tidak intuitif. Bayangkan terminal
dengan tinggi 20 baris, dan kamu baru punya 3 pesan. Tanpa padding,
pesan akan muncul di atas dan sisanya kosong di bawah. Dengan padding
baris kosong di *atas*, 3 pesan tersebut muncul di *bawah* — lebih
natural untuk aplikasi chat.

```go
func (v *ChatView) renderMsg(msg chat.Message) string {
    ts := msg.Timestamp.Local().Format("15:04")

    switch msg.Type {
    case chat.MsgJoin:
        // Join/leave: miring, warna abu-abu
        return styleMsgSystem.Render(fmt.Sprintf("  ─── %s joined ───", msg.Nick))

    case chat.MsgChat:
        // Warna berbeda untuk pesan sendiri vs orang lain
        var nickStr string
        if msg.SenderID == v.selfID {
            nickStr = styleMsgNickSelf.Render(msg.Nick)   // biru
        } else {
            nickStr = styleMsgNickOther.Render(msg.Nick)  // oranye
        }

        // Word-wrap: hitung lebar yang tersisa setelah prefix "[15:04] nick: "
        prefixLen := len(ts) + 3 + len(msg.Nick) + 2
        bodyWidth := v.width - prefixLen - 2
        body := wrapText(msg.Body, bodyWidth)

        // Jika pesan multi-baris setelah wrap, indent baris ke-2 dst
        // agar teks tetap rata dengan baris pertama
        indent := strings.Repeat(" ", prefixLen)
        lines := strings.Split(body, "\n")
        for i := 1; i < len(lines); i++ {
            lines[i] = indent + lines[i]
        }
        ...
    }
}
```

Word-wrapping adalah detail kecil tapi penting. Tanpanya, pesan panjang
akan "meluber" ke baris berikutnya tanpa indent yang benar, membuat tampilan
menjadi berantakan. Kalkulasi `prefixLen` menghitung berapa karakter yang
sudah "terpakai" oleh timestamp dan nickname, sehingga body-wrap tahu
sampai mana dia boleh menulis sebelum pindah baris.

### App (Bubble Tea Model)

```go
// Semua state UI ada di sini
type Model struct {
    room    *chat.Room      // referensi ke room jaringan
    selfID  string

    input    textinput.Model  // komponen input teks dari library bubbles
    chatView *ChatView

    peers     []string  // daftar peer saat ini
    showPeers bool      // apakah panel peer ditampilkan?
    statusMsg string    // pesan error sementara
    quitting  bool      // apakah user sedang keluar?

    width, height int   // dimensi terminal
}
```

**Msg types** — inilah "event" yang bisa terjadi:

```go
// Pesan dari jaringan — dibungkus agar bisa masuk ke event loop Bubble Tea
type incomingMsg chat.Message

// Snapshot terbaru daftar peer
type peerListMsg []string

// Error yang perlu ditampilkan di UI (tapi tidak perlu crash program)
type errMsg struct{ err error }
```

Kenapa kita perlu mendefinisikan tipe-tipe ini? Karena `Update` harus
bisa membedakan sumber event yang berbeda. `tea.KeyMsg` untuk keyboard,
`tea.WindowSizeMsg` untuk resize terminal, `incomingMsg` untuk pesan
dari jaringan, dst.

**Init** — dijalankan sekali saat program mulai:

```go
func (m *Model) Init() tea.Cmd {
    return tea.Batch(
        textinput.Blink,         // mulai animasi cursor berkedip
        m.listenForMessages(),   // mulai mendengarkan pesan dari jaringan
        m.refreshPeerList(),     // ambil daftar peer saat ini
    )
}
```

`tea.Batch` menjalankan beberapa `Cmd` secara bersamaan. Semua tiga
Cmd ini dijalankan di goroutine terpisah oleh Bubble Tea.

**`listenForMessages`** adalah pattern yang sangat penting dan idiomatik:

```go
func (m *Model) listenForMessages() tea.Cmd {
    // Cmd adalah fungsi yang mengembalikan tea.Msg
    return func() tea.Msg {
        // Blok di sini sampai ada pesan di channel (atau channel ditutup)
        msg, ok := <-m.room.Messages
        if !ok {
            return tea.Quit()  // channel ditutup → program selesai
        }
        // Kembalikan pesan sebagai event ke loop Update
        return incomingMsg(msg)
    }
}
```

Dan di `Update`, setelah menerima `incomingMsg`, kita langsung *re-arm*
listener:

```go
case incomingMsg:
    m.chatView.Push(chat.Message(msg))  // tampilkan di UI
    // Daftarkan listener BARU untuk pesan berikutnya
    // Ini menciptakan "rantai" listener yang terus berjalan
    cmds = append(cmds, m.listenForMessages())
```

Mengapa tidak cukup satu goroutine yang terus loop? Karena Bubble Tea
tidak mengizinkan state dimodifikasi dari goroutine lain. Dengan pola
"satu Cmd per pesan + re-arm", kita menggunakan mekanisme Bubble Tea
sendiri untuk mengantarkan pesan ke event loop dengan aman.

**Update** — jantung dari logika UI:

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {

    case tea.WindowSizeMsg:
        // Terminal di-resize → update dimensi dan render ulang
        m.width = msg.Width
        m.height = msg.Height
        m.chatView.Resize(m.chatPanelWidth(), m.chatPanelHeight())

    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyEsc, tea.KeyCtrlC:
            m.quitting = true
            return m, tea.Quit  // signal ke Bubble Tea untuk keluar

        case tea.KeyEnter:
            if text := strings.TrimSpace(m.input.Value()); text != "" {
                m.input.Reset()
                // sendMessage adalah Cmd — dijalankan di goroutine terpisah
                // hasilnya (incomingMsg atau errMsg) akan kembali ke Update
                cmds = append(cmds, m.sendMessage(text))
            }

        case tea.KeyTab:
            m.showPeers = !m.showPeers  // toggle panel peer

        case tea.KeyCtrlL:
            m.chatView = NewChatView(...)  // clear history

        default:
            // Tombol lain → teruskan ke komponen textinput
            var inputCmd tea.Cmd
            m.input, inputCmd = m.input.Update(msg)
            cmds = append(cmds, inputCmd)
        }
    ...
    }
    return m, tea.Batch(cmds...)
}
```

Perhatikan bahwa `Update` tidak pernah memanggil fungsi yang blocking
atau melakukan network call langsung. Semua operasi async (kirim pesan,
dengarkan pesan) dilakukan via `Cmd`. `Update` sendiri harus selesai
sangat cepat karena UI tidak bisa render sampai `Update` selesai.

**`sendMessage`** — mengirim pesan ke jaringan:

```go
func (m *Model) sendMessage(text string) tea.Cmd {
    return func() tea.Msg {
        if err := m.room.Send(text); err != nil {
            return errMsg{err}  // kirim error ke Update untuk ditampilkan
        }
        // GossipSub TIDAK mengembalikan pesan kita sendiri
        // Jadi kita echo secara manual agar muncul di layar kita sendiri
        selfMsg := chat.NewChat(m.selfID, m.room.Nick(), text)
        return incomingMsg(selfMsg)
    }
}
```

Detail kecil tapi penting: GossipSub tidak mengirimkan pesan kita ke
diri kita sendiri (sudah kita skip di `readLoop` pun). Jadi setelah
berhasil mengirim, kita buat `incomingMsg` untuk pesan kita sendiri
agar muncul di layar kita. Hasilnya identitas sender adalah `m.selfID`
sehingga nama kita akan muncul dengan warna biru (warna "self").

**View** — render semua ke string:

```go
func (m *Model) View() string {
    // ...
    var sections []string

    // Title bar: "jala #lobby"
    sections = append(sections, title)

    // Panel utama: chat + (opsional) daftar peer berdampingan
    if m.showPeers {
        content := lipgloss.JoinHorizontal(lipgloss.Top,
            styleBorder.Width(chatW).Height(h).Render(chatPanel),
            peerPanel,
        )
        sections = append(sections, content)
    } else {
        sections = append(sections, ...)
    }

    // Input area
    sections = append(sections, inputLine)

    // Status bar
    sections = append(sections, status)

    // Help bar
    sections = append(sections, HelpView(m.width))

    // Gabungkan semua section dengan newline
    return strings.Join(sections, "\n")
}
```

`View` hanya merakit string dari komponen-komponen yang sudah dirender.
`lipgloss.JoinHorizontal` menempatkan dua panel berdampingan — seperti
`display: flex` di CSS. View tidak punya logika sama sekali — hanya
"terjemahkan Model menjadi teks".

---

## 8. Titik Masuk: `cmd/jala/main.go`

`main.go` adalah "konduktor" — dia tidak berisi logika bisnis apapun,
hanya menghubungkan semua layer di atas.

```go
func main() {
    // 1. Parse flags dari command line
    nick    := flag.String("nick",  "",      "display name")
    room    := flag.String("room",  "lobby", "chat room")
    keyPath := flag.String("key",   "",      "key file")
    ...
    flag.Parse()

    // 2. Setup logging (diam di TUI mode, verbose di debug mode)
    logLevel := "error"
    if *debug { logLevel = "debug" }
    logging.SetLogLevel("*", logLevel)

    // 3. Load atau generate identitas
    id, err := identity.Load(*keyPath)
    ...

    // 4. Buat context yang akan dibatalkan saat SIGINT/SIGTERM
    // signal.NotifyContext adalah cara idiomatik Go untuk handle OS signals
    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    // 5. Bangun node libp2p
    n, err := node.New(ctx, nodeCfg)
    defer n.Close()  // pastikan node di-close saat main() selesai

    // 6. Join chat room
    r, err := chat.Join(ctx, n.Host, *room, displayNick)
    defer r.Close()  // pastikan "leave" terkirim saat keluar

    // 7. Jalankan TUI (blocking sampai user quit)
    ui.Run(ctx, r)
}
```

Penggunaan `defer` di sini membentuk urutan shutdown yang benar:
ketika `main()` selesai (karena `ui.Run()` return), Go akan menjalankan
semua `defer` dalam urutan **terbalik** dari pendefinisiannya:
1. `r.Close()` — kirim pesan "leave", unsubscribe dari room
2. `n.Close()` — tutup DHT, mDNS, dan host
3. `stop()` — batalkan context signal

Ini memastikan shutdown berjalan dari lapisan atas ke bawah, bukan
sebaliknya.

```go
// randomNick membuat nickname seperti "silentOtter" atau "boldFalcon"
func randomNick() string {
    adj := adjectives[rand.Intn(len(adjectives))]
    animal := animals[rand.Intn(len(animals))]
    // Kapitalisasi huruf pertama animal untuk format camelCase
    // animal[0]-32 mengubah huruf kecil ke huruf besar di ASCII
    // ('b' adalah 98, 'B' adalah 66, selisih 32)
    return adj + string(animal[0]-32) + animal[1:]
}
```

Trick `animal[0]-32` adalah cara mengubah huruf kecil ke huruf besar
di ASCII tanpa import `strings` atau `unicode`. Ini bekerja karena di
tabel ASCII, semua huruf besar dan kecil selisih persis 32 posisi.

---

## 9. Alur Data Lengkap

Mari kita trace alur lengkap dari "user mengetik pesan" hingga "pesan
muncul di layar pengguna lain":

```
PENGIRIM (alice):

  Keyboard "h-e-l-l-o" + Enter
       │
       ▼
  tea.KeyMsg{Enter} masuk ke Update()
       │
       ▼
  sendMessage("hello") dipanggil sebagai Cmd
  (dijalankan di goroutine Bubble Tea)
       │
       ▼
  room.Send("hello")
       │  → NewChat(selfID, "alice", "hello") dibuat
       │  → JSON encode: {"type":"chat","nick":"alice","body":"hello",...}
       ▼
  topic.Publish([]byte(json))
       │
       ▼
  GossipSub menyebarkan ke mesh peer

  sendMessage juga return incomingMsg (echo ke diri sendiri)
       │
       ▼
  Update() menerima incomingMsg
       │
       ▼
  chatView.Push(msg) → tampil di layar alice dengan warna BIRU

─────────────────────────────────────────────────

PENERIMA (bob):

  GossipSub menerima bytes dari jaringan
       │
       ▼
  r.sub.Next(ctx) di readLoop() return rawMsg
       │
       ▼
  Decode(rawMsg.Data) → Message{type:chat, nick:"alice", body:"hello"}
       │
       ▼
  r.msgs <- msg  (kirim ke channel)
       │
       ▼
  listenForMessages() Cmd di UI bob mendapat msg dari channel
       │
       ▼
  return incomingMsg(msg) → masuk ke event loop Bubble Tea bob
       │
       ▼
  Update() menerima incomingMsg
       │
       ▼
  chatView.Push(msg) → tampil di layar bob dengan warna ORANYE
       │
       ▼
  listenForMessages() baru di-arm untuk pesan selanjutnya
```

---

## 10. Pola Go yang Digunakan

### Context dan Cancellation

`context.Context` adalah cara standar Go untuk mengelola "lifetime" dari
operasi async. Setiap fungsi yang berjalan lama (network call, goroutine
jangka panjang) menerima `ctx context.Context` sebagai argumen pertama.
Ketika context dibatalkan (via `cancel()` atau saat program menerima
SIGINT), semua fungsi yang menerima context tersebut akan berhenti.

```go
// Ini membuat context yang otomatis dibatalkan saat SIGINT/SIGTERM diterima
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
```

### Error Wrapping

```go
return nil, fmt.Errorf("identity: derive PeerID: %w", err)
//                      ^^^^^^^^^^^^^^^^^^^^^^^^^^  ^^
//                      konteks tambahan           error asli
```

`%w` (bukan `%v`) membungkus error asli sehingga bisa di-*unwrap* dengan
`errors.Is()` dan `errors.As()`. Prefix seperti `"identity: derive PeerID: "`
membantu saat debugging — kamu bisa langsung tahu error ini terjadi di
fungsi mana, bahkan tanpa stack trace.

### Goroutines dan Channels

Go menggunakan goroutines (bukan threads OS) dan channels untuk concurrency.
Goroutines sangat ringan — kamu bisa spawn ribuan tanpa masalah memori.

```go
// Pattern: goroutine + channel untuk komunikasi async
msgs := make(chan Message, 64)  // channel dengan buffer

// Producer goroutine: menulis ke channel
go func() {
    for {
        msg := getMessageFromNetwork()
        msgs <- msg  // kirim ke channel
    }
}()

// Consumer (di goroutine lain): membaca dari channel
msg := <-msgs  // tunggu sampai ada pesan
```

### `sync.Once` untuk Idempotent Cleanup

```go
var once sync.Once

func cleanup() {
    once.Do(func() {
        // Kode ini hanya jalan SEKALI, tidak peduli cleanup() dipanggil berapa kali
        closeConnection()
        releaseResources()
    })
}
```

### Interface untuk Decoupling

libp2p menggunakan interfaces secara ekstensif. Misalnya `host.Host` adalah
interface (bukan struct konkret), sehingga kode yang menggunakannya tidak
perlu tahu implementasi spesifiknya. Ini memudahkan testing (bisa mock)
dan memungkinkan penggantian implementasi tanpa mengubah kode yang bergantung.

---

## Penutup

Proyek jala membangun sebuah *layer cake* yang rapi:

```
Identitas (Ed25519)
    ↓ memberikan: kunci privat & PeerID
Node (libp2p)
    ↓ memberikan: host.Host yang terenkripsi & terhubung ke peers
Room (GossipSub)
    ↓ memberikan: channel Messages yang siap dibaca
UI (Bubble Tea)
    ↓ memberikan: tampilan terminal yang interaktif
```

Setiap lapisan hanya bergantung pada lapisan di bawahnya melalui interface
yang sempit dan terdefinisi dengan baik. `ui` tidak tahu tentang libp2p.
`chat` tidak tahu tentang rendering. `node` tidak tahu tentang format pesan.

Itulah mengapa kode ini disebut "idiomatic" — bukan karena menggunakan
fitur-fitur canggih, tapi karena setiap bagian punya tanggung jawab yang
jelas, kesalahan ditangani dengan eksplesit, dan goroutines digunakan
dengan pola yang aman.
