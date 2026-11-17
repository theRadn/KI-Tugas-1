package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/utils"
	"gorm.io/gorm"
)

const (
	MaxVideoChunkBytes = 512 << 10 // maks 512 kb per potongan (amplop sekitar 910 KB, di bawah 1 MiB)
	maxVideoChunks     = 400       // maks 100 mb
)

var (
	ErrVideoNotFound     = errors.New("video tidak ditemukan")
	ErrInvalidVideoChunk = errors.New("potongan video tidak valid")
)

type VideoService struct {
	root string
	db   *gorm.DB
}

type videoManifest struct {
	ChunkCount int   `json:"chunk_count"`
	SizeBytes  int64 `json:"size_bytes"`
}

type videoCipher struct {
	name      string
	algorithm utils.Algorithm
	size      int
}

type storedVideoCipher struct {
	utils.EncryptedData
	WrappedDEK []byte `json:"wrapped_dek,omitempty"`
}

func NewVideoService(root string, db *gorm.DB) *VideoService {
	return &VideoService{root: root, db: db}
}

func videoCiphers() []videoCipher {
	return []videoCipher{
		{"AES", utils.AlgorithmAES256CTR, 32},
		{"DES", utils.AlgorithmDESCTR, 8},
		{"RC4", utils.AlgorithmRC4, 32},
	}
}

// ciphertextPotongan membaca berkas potongan dan mengembalikan ciphertext
// algoritma yang diminta tanpa mendekripsinya.
func ciphertextPotongan(path, algorithm string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrVideoNotFound
	}
	if err != nil {
		return nil, err
	}
	var versions map[string]utils.EncryptedData
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, err
	}
	encrypted, ok := versions[algorithm]
	if !ok {
		return nil, ErrInvalidVideoChunk
	}
	return encrypted.Ciphertext, nil
}

// CiphertextChunk mengembalikan ciphertext satu potongan video untuk
// fitur "lihat ciphertext", beserta jumlah potongannya.
func (s *VideoService) CiphertextChunk(userID, videoID uuid.UUID, index int, algorithm string) ([]byte, int, error) {
	manifest, err := s.manifest(userID, videoID)
	if err != nil {
		return nil, 0, err
	}
	if index < 0 || index >= manifest.ChunkCount {
		return nil, 0, ErrInvalidVideoChunk
	}
	ciphertext, err := ciphertextPotongan(s.chunkPath(userID, videoID, index), algorithm)
	return ciphertext, manifest.ChunkCount, err
}

func (s *VideoService) Start(userID uuid.UUID, chunkCount int, sizeBytes int64) (uuid.UUID, error) {
	if chunkCount < 1 || chunkCount > maxVideoChunks ||
		sizeBytes <= int64(chunkCount-1)*MaxVideoChunkBytes ||
		sizeBytes > int64(chunkCount)*MaxVideoChunkBytes {
		return uuid.Nil, ErrInvalidVideoChunk
	}
	manifest, err := json.Marshal(videoManifest{chunkCount, sizeBytes})
	if err != nil {
		return uuid.Nil, err // ini banyak error handling biar file selanjutnya ga korup
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	data, err := s.packVideoData(manifest, userID, id, "video-manifest")
	if err != nil {
		return uuid.Nil, err
	}
	dir := s.videoDir(userID, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return uuid.Nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return uuid.Nil, err
	}
	return id, nil
}

func (s *VideoService) PutChunk(userID, videoID uuid.UUID, index int, chunk []byte) error {
	manifest, err := s.manifest(userID, videoID)
	if err != nil {
		return err
	}
	if index < 0 || index >= manifest.ChunkCount || len(chunk) == 0 || len(chunk) > MaxVideoChunkBytes {
		return ErrInvalidVideoChunk
	}
	expected := MaxVideoChunkBytes
	if index == manifest.ChunkCount-1 {
		expected = int(manifest.SizeBytes - int64(index)*MaxVideoChunkBytes)
	}
	if len(chunk) != expected {
		return ErrInvalidVideoChunk
	}
	data, err := s.packVideoData(chunk, userID, videoID, fmt.Sprintf("chunk:%d", index))
	if err != nil {
		return err
	}
	return os.WriteFile(s.chunkPath(userID, videoID, index), data, 0o600)
}

func (s *VideoService) GetChunk(userID, videoID uuid.UUID, index int, algorithm string) ([]byte, int, error) {
	manifest, err := s.manifest(userID, videoID)
	if err != nil {
		return nil, 0, err
	}
	if index < 0 || index >= manifest.ChunkCount {
		return nil, 0, ErrInvalidVideoChunk
	}
	data, err := os.ReadFile(s.chunkPath(userID, videoID, index))
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, ErrVideoNotFound
	}
	if err != nil {
		return nil, 0, err
	}
	chunk, err := s.unpackVideoData(data, userID, videoID, fmt.Sprintf("chunk:%d", index), algorithm)
	if err != nil {
		return nil, 0, err
	}
	expected := MaxVideoChunkBytes
	if index == manifest.ChunkCount-1 {
		expected = int(manifest.SizeBytes - int64(index)*MaxVideoChunkBytes)
	}
	if len(chunk) != expected {
		return nil, 0, ErrInvalidVideoChunk
	}
	return chunk, manifest.ChunkCount, nil
}

func (s *VideoService) manifest(userID, videoID uuid.UUID) (videoManifest, error) {
	data, err := os.ReadFile(filepath.Join(s.videoDir(userID, videoID), "manifest.json"))
	if errors.Is(err, os.ErrNotExist) {
		return videoManifest{}, ErrVideoNotFound
	}
	if err != nil {
		return videoManifest{}, err
	}
	data, err = s.unpackVideoData(data, userID, videoID, "video-manifest", "AES")
	if err != nil {
		return videoManifest{}, err
	}
	var manifest videoManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return videoManifest{}, err
	}
	if manifest.ChunkCount < 1 || manifest.ChunkCount > maxVideoChunks ||
		manifest.SizeBytes <= int64(manifest.ChunkCount-1)*MaxVideoChunkBytes ||
		manifest.SizeBytes > int64(manifest.ChunkCount)*MaxVideoChunkBytes {
		return videoManifest{}, ErrInvalidVideoChunk
	}
	return manifest, nil
}

// Folder mengembalikan lokasi potongan video, dipakai saat video dicatat
// di tabel berkas supaya ikut terhapus bersama barisnya.
func (s *VideoService) Folder(userID, videoID uuid.UUID) string {
	return s.videoDir(userID, videoID)
}

// Lengkap bernilai true bila semua potongan sudah tersimpan.
func (s *VideoService) Lengkap(userID, videoID uuid.UUID) (bool, error) {
	manifest, err := s.manifest(userID, videoID)
	if err != nil {
		return false, err
	}
	for i := 0; i < manifest.ChunkCount; i++ {
		if _, err := os.Stat(s.chunkPath(userID, videoID, i)); err != nil {
			return false, nil
		}
	}
	return true, nil
}

func (s *VideoService) videoDir(userID, videoID uuid.UUID) string {
	return filepath.Join(s.root, userID.String(), videoID.String())
}

func (s *VideoService) chunkPath(userID, videoID uuid.UUID, index int) string {
	return filepath.Join(s.videoDir(userID, videoID), fmt.Sprintf("%06d.json", index))
}
