# KI-Tugas-1

Backend for [SIPMABA-V2](https://github.com/setiAjiid/SIPMABA-v2.0/) (Go, Gin, GORM, SQLite)

| Name           | NRP        | Class     |
| ---            | ---        | ----------|
| Dimas Setiaji | 5025241056 | Information Security (I) |
| Addien Zafriyan Al Akhsan | 5025241058 | Information Security (I) |
| Abdullah Sultan Barizy | 5025241092 | Information Security (I) |
| Raden Kurniawan Agung Fitrianto | 5025241104 | Information Security (I) |

## Setup

```bash
git clone https://github.com/theRadn/KI-Tugas-1.git
cd KI-Tugas-1
cp .env.example .env    # then fill in all values, pay attention to key lengths
go mod download
go run main.go --migrate
go run main.go          # server at http://localhost:8080

```

CGO (`gcc`) is required because it uses `go-sqlite3`.

## Security

* **In-transit encryption.** Clients call `POST /api/handshake` (X25519),
then every request and response is wrapped in an AES-128-CTR + HMAC-SHA256 envelope
with a one-time `request_id` (replays are rejected with 409). For GET requests, the envelope
is sent in the `X-Encrypted-Envelope` header (JSON envelope in base64).
The envelope body limit is 1 MiB.
* **File encryption.** Every file is encrypted using AES-256-CTR, DES-CTR,
and RC4 (per-file keys via HKDF), and only the ciphertext is stored in
`storage/berkas/<id>/`. File contents are inspected (`DetectContentType`, size,
photo dimensions) before being stored.
* **Biodata encryption.** 101 personal data columns (NIK, address, health,
income, etc.) are stored as `enc:v2:` containing three
ciphertexts (AES-256-CTR, DES-CTR, RC4) with an AES-keyed HMAC tag, via the GORM `enkripsi`
serializer (`entity/enkripsi.go`). `GET /biodata/:tab?algoritma=`
selects the decrypted version (default is AES).
Relation keys, registration numbers, and dates are left in plaintext for searchability.
* **Authentication.** JWT (120 minutes) in the `Authorization: Bearer` header.
Passwords are hashed with bcrypt.
* **Rate limiting.** Global limit of 100/minute, login and registration 5/minute, handshake
5 then 1/minute per IP. Video chunk and document routes have their own quota of
300/minute and are excluded from the global limiter.

## Endpoints

All endpoints are under `/api` and encrypted, except for `/handshake`.

| Method | Path | Login Required |
| --- | --- | --- |
| POST | `/handshake` | - |
| POST | `/auth/daftar`, `/auth/masuk`, `/auth/keluar`, `/auth/lupa_katasandi` | - |
| PUT | `/auth/ganti-katasandi` | yes |
| GET, PUT | `/biodata/:tab` (`data-diri`, `pendidikan`, `keluarga`, `rumah`, `catatan-khusus`, `pendukung`) | yes |
| GET | `/berkas` | yes |
| POST | `/berkas/:kode` (body `nama_asli`, `tipe_mime`, `isi_base64`) | yes |
| GET | `/berkas/:id/unduh?algoritma=AES|DES|RC4` | yes |
| GET | `/berkas/:id/ciphertext?algoritma=AES|DES|RC4` | yes |
| DELETE | `/berkas/:id` | yes |
| POST | `/videos` (body `chunk_count`, `size_bytes`, `nama_asli`, `tipe_mime`) | yes |
| PUT, GET | `/videos/:id/chunks/:index` (`data` base64, GET with `?algoritma=`) | yes |
| POST | `/documents` (body `filename`, `size_bytes`) | yes |
| PUT, GET | `/documents/:id/chunks/:index` (`data` base64, GET with `?algoritma=`) | yes |
| GET | `/videos/:id/chunks/:index/ciphertext`, `/documents/:id/chunks/:index/ciphertext` (`?algoritma=`) | yes |
| GET | `/beranda` | yes |
| GET | `/analisis/berkas` | yes |
| POST | `/analisis/uji-dekripsi` (body `berkas_id`, `jumlah_percobaan` 1-100) | yes |

## Known Limitations

* DES and RC4 are insecure, they are used because they were required by the assignment.
* A single set of file keys is used for all users, stored in `.env`. The same key is also used as the file HMAC key.
* `keluar` (logout) does not revoke the JWT in the server, only remove it in the client local storage, the token remains valid until it expires.
* MP4/WebM videos up to 50 MB and documents up to 10 MB are uploaded in 512 KB chunks (envelope size around 910 KB). Both are recorded in the files table, but are excluded from Analysis due to the chunk-based format.