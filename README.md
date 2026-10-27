# KI-Tugas-1

Backend SIPMABA (Go, Gin, GORM, SQLite).

## Setup

```
git clone https://github.com/theRadn/KI-Tugas-1.git
cd KI-Tugas-1
cp .env.example .env    # lalu isi semua nilai, perhatikan panjang kunci
go mod download
go run main.go --migrate --seed
go run main.go          # server di http://localhost:8080
```

Butuh CGO (gcc) karena memakai `go-sqlite3`.

## Keamanan

- **Enkripsi in-transit.** Klien memanggil `POST /api/handshake` (X25519),
  lalu setiap permintaan dan respons dibungkus amplop AES-128-CTR + HMAC-SHA256
  dengan `request_id` sekali pakai (replay ditolak 409). Untuk GET, amplop
  dikirim di header `X-Encrypted-Envelope` (JSON amplop dalam base64).
  Batas body amplop 1 MiB.
- **Enkripsi berkas.** Setiap berkas dienkripsi dengan AES-256-CTR, DES-CTR,
  dan RC4 (kunci per berkas lewat HKDF), lalu hanya ciphertext yang disimpan di
  `storage/berkas/<id>/`. Isi berkas diperiksa (`DetectContentType`, ukuran,
  dimensi foto) sebelum disimpan.
- **Enkripsi biodata.** 101 kolom data pribadi (NIK, alamat, kesehatan,
  penghasilan, dan lain-lain) disimpan sebagai `enc:v1:` + ciphertext
  AES-256-CTR + HMAC lewat serializer GORM `enkripsi` (`entity/enkripsi.go`).
  Kunci relasi, nomor pendaftaran, dan tanggal dibiarkan polos untuk pencarian.
- **Autentikasi.** JWT (120 menit) di header `Authorization: Bearer`.
  Kata sandi di-hash dengan bcrypt.
- **Rate limit.** Global 100/menit, masuk dan daftar 5/menit, handshake
  5 lalu 1/menit per IP.

## Endpoint

Semua di bawah `/api` dan terenkripsi, kecuali `/handshake`.

| Metode | Jalur | Login |
| --- | --- | --- |
| POST | `/handshake` | - |
| POST | `/auth/daftar`, `/auth/masuk`, `/auth/keluar`, `/auth/lupa_katasandi` | - |
| PUT | `/auth/ganti-katasandi` | ya |
| GET, PUT | `/biodata/:tab` (`data-diri`, `pendidikan`, `keluarga`, `rumah`, `catatan-khusus`, `pendukung`) | ya |
| GET | `/berkas` | ya |
| POST | `/berkas/:kode` (body `nama_asli`, `tipe_mime`, `isi_base64`) | ya |
| GET | `/berkas/:id/unduh?algoritma=AES\|DES\|RC4` | ya |
| GET | `/berkas/:id/ciphertext?algoritma=AES\|DES\|RC4` | ya |
| DELETE | `/berkas/:id` | ya |
| POST | `/videos` (body `chunk_count`, `size_bytes`, `nama_asli`, `tipe_mime`) | ya |
| PUT, GET | `/videos/:id/chunks/:index` (`data` base64, GET dengan `?algoritma=`) | ya |
| GET | `/beranda` | ya |
| GET | `/analisis/berkas` | ya |
| POST | `/analisis/uji-dekripsi` (body `berkas_id`, `jumlah_percobaan` 1-100) | ya |

## Batasan yang diketahui

- DES dan RC4 tidak aman; dipakai karena diwajibkan tugas.
- Satu set kunci berkas untuk semua pengguna, disimpan di `.env`.
  Kunci yang sama juga dipakai sebagai kunci HMAC berkas.
- `keluar` belum mencabut JWT; token tetap berlaku sampai kedaluwarsa.
- Video MP4/WebM maks 10 MB, dikirim per potongan 256 KB lewat `/videos`
  (batas ini menjaga jumlah permintaan di bawah limiter global 100/menit).
- Dokumen (PDF/DOC/XLS) belum didukung.
