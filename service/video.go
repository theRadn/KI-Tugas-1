package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/utils"
)

const MaxVideoChunkBytes = 256 << 10 // maks 256 kb per potongan
const maxVideoChunks = 400 // maks 100 mb

var ErrVideoNotFound = errors.New("video tidak ditemukan")
var ErrInvalidVideoChunk = errors.New("potongan video tidak valid")

type VideoService struct{ root string }

type videoManifest struct {
	ChunkCount int   `json:"chunk_count"`
	SizeBytes  int64 `json:"size_bytes"`
}

type videoCipher struct {
	name      string
	algorithm utils.Algorithm
	key       []byte
}

func NewVideoService(root string) *VideoService { return &VideoService{root: root} }

func videoCiphers() ([]videoCipher, error) {
	ciphers := []videoCipher{
		{"AES", utils.AlgorithmAES256CTR, []byte(os.Getenv("DATABASE_SECRET_AES256"))},
		{"DES", utils.AlgorithmDESCTR, []byte(os.Getenv("DATABASE_SECRET_DES"))},
		{"RC4", utils.AlgorithmRC4, []byte(os.Getenv("DATABASE_SECRET_RC4"))},
	}
	if len(ciphers[0].key) != 32 || len(ciphers[1].key) != 8 || len(ciphers[2].key) == 0 {
		return nil, errors.New("kunci enkripsi video belum dikonfigurasi dengan benar")
	}
	return ciphers, nil
}

// satu cut video disimpan dalam beberapa versi terenkripsi
func packVideoData(plaintext []byte) ([]byte, error) {
	ciphers, err := videoCiphers()
	if err != nil {
		return nil, err
	}
	versions := make(map[string]utils.EncryptedData, len(ciphers))
	for _, cipher := range ciphers {
		versions[cipher.name], err = utils.Encrypt(cipher.algorithm, cipher.key, cipher.key, plaintext)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(versions)
}

func unpackVideoData(data []byte, algorithm string) ([]byte, error) {
	ciphers, err := videoCiphers()
	if err != nil {
		return nil, err
	}
	var versions map[string]utils.EncryptedData
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, err
	}
	if len(versions) != len(ciphers) {
		return nil, ErrInvalidVideoChunk
	}
	for _, cipher := range ciphers {
		encrypted, ok := versions[cipher.name]
		if !ok || encrypted.Algorithm != cipher.algorithm {
			return nil, ErrInvalidVideoChunk
		}
		if cipher.name == algorithm {
			return utils.Decrypt(encrypted, cipher.key, cipher.key)
		}
	}
	return nil, ErrInvalidVideoChunk
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
	data, err := packVideoData(manifest)
	if err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	dir := s.videoDir(userID, id)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return uuid.Nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600); err != nil {
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
	data, err := packVideoData(chunk)
	if err != nil {
		return err
	}
	return os.WriteFile(s.chunkPath(userID, videoID, index), data, 0600)
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
	chunk, err := unpackVideoData(data, algorithm)
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
	data, err = unpackVideoData(data, "AES")
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

func (s *VideoService) videoDir(userID, videoID uuid.UUID) string {
	return filepath.Join(s.root, userID.String(), videoID.String())
}

func (s *VideoService) chunkPath(userID, videoID uuid.UUID, index int) string {
	return filepath.Join(s.videoDir(userID, videoID), fmt.Sprintf("%06d.json", index))
}
