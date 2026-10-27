package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	MaxDocumentChunkBytes = 512 << 10 // amplop sekitar 910 KB, di bawah 1 MiB
	maxDocumentChunks     = 400
	maxDocumentBytes      = int64(MaxDocumentChunkBytes) * maxDocumentChunks
)

var (
	ErrDocumentNotFound        = errors.New("document not found")
	ErrInvalidDocument         = errors.New("invalid document")
	ErrUnsupportedDocumentType = errors.New("unsupported document type")
)

type DocumentService struct {
	root string
}

type documentManifest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	ChunkCount  int    `json:"chunk_count"`
	SizeBytes   int64  `json:"size_bytes"`
}

var allowedDocumentExtensions = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".ppt":  true,
	".pptx": true,
	".txt":  true,
	".csv":  true,
}

func NewDocumentService(root string) *DocumentService {
	return &DocumentService{root: root}
}

func (s *DocumentService) Start(
	userID uuid.UUID,
	filename string,
	sizeBytes int64,
) (uuid.UUID, int, error) {
	if filename == "" ||
		filename != filepath.Base(filename) ||
		strings.ContainsAny(filename, `/\`) ||
		sizeBytes < 1 ||
		sizeBytes > maxDocumentBytes {
		return uuid.Nil, 0, ErrInvalidDocument
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedDocumentExtensions[ext] {
		return uuid.Nil, 0, ErrUnsupportedDocumentType
	}

	chunkCount := int((sizeBytes + MaxDocumentChunkBytes - 1) /
		MaxDocumentChunkBytes)

	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	manifest := documentManifest{
		Filename:    filename,
		ContentType: contentType,
		ChunkCount:  chunkCount,
		SizeBytes:   sizeBytes,
	}

	plainManifest, err := json.Marshal(manifest)
	if err != nil {
		return uuid.Nil, 0, err
	}

	encryptedManifest, err := packVideoData(plainManifest)
	if err != nil {
		return uuid.Nil, 0, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, 0, err
	}

	dir := s.documentDir(userID, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return uuid.Nil, 0, err
	}

	if err := os.WriteFile(
		filepath.Join(dir, "manifest.json"),
		encryptedManifest,
		0o600,
	); err != nil {
		_ = os.RemoveAll(dir)
		return uuid.Nil, 0, err
	}

	return id, chunkCount, nil
}

func (s *DocumentService) PutChunk(
	userID, documentID uuid.UUID,
	index int,
	chunk []byte,
) error {
	manifest, err := s.manifest(userID, documentID)
	if err != nil {
		return err
	}

	if index < 0 || index >= manifest.ChunkCount {
		return ErrInvalidDocument
	}

	expected := MaxDocumentChunkBytes
	if index == manifest.ChunkCount-1 {
		expected = int(
			manifest.SizeBytes -
				int64(index)*MaxDocumentChunkBytes,
		)
	}

	if len(chunk) != expected {
		return ErrInvalidDocument
	}

	encrypted, err := packVideoData(chunk)
	if err != nil {
		return err
	}

	return os.WriteFile(
		s.chunkPath(userID, documentID, index),
		encrypted,
		0o600,
	)
}

func (s *DocumentService) GetChunk(
	userID, documentID uuid.UUID,
	index int,
	algorithm string,
) ([]byte, documentManifest, error) {
	manifest, err := s.manifest(userID, documentID)
	if err != nil {
		return nil, documentManifest{}, err
	}

	if index < 0 || index >= manifest.ChunkCount {
		return nil, documentManifest{}, ErrInvalidDocument
	}

	data, err := os.ReadFile(
		s.chunkPath(userID, documentID, index),
	)
	if errors.Is(err, os.ErrNotExist) {
		return nil, documentManifest{}, ErrDocumentNotFound
	}
	if err != nil {
		return nil, documentManifest{}, err
	}

	chunk, err := unpackVideoData(data, algorithm)
	if err != nil {
		return nil, documentManifest{}, err
	}

	expected := MaxDocumentChunkBytes
	if index == manifest.ChunkCount-1 {
		expected = int(
			manifest.SizeBytes -
				int64(index)*MaxDocumentChunkBytes,
		)
	}

	if len(chunk) != expected {
		return nil, documentManifest{}, ErrInvalidDocument
	}

	return chunk, manifest, nil
}

func (s *DocumentService) manifest(
	userID, documentID uuid.UUID,
) (documentManifest, error) {
	data, err := os.ReadFile(
		filepath.Join(s.documentDir(userID, documentID), "manifest.json"),
	)
	if errors.Is(err, os.ErrNotExist) {
		return documentManifest{}, ErrDocumentNotFound
	}
	if err != nil {
		return documentManifest{}, err
	}

	data, err = unpackVideoData(data, "AES")
	if err != nil {
		return documentManifest{}, err
	}

	var manifest documentManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return documentManifest{}, err
	}

	ext := strings.ToLower(filepath.Ext(manifest.Filename))
	expectedCount := int(
		(manifest.SizeBytes + MaxDocumentChunkBytes - 1) /
			MaxDocumentChunkBytes,
	)

	if !allowedDocumentExtensions[ext] ||
		manifest.SizeBytes < 1 ||
		manifest.SizeBytes > maxDocumentBytes ||
		manifest.ChunkCount < 1 ||
		manifest.ChunkCount > maxDocumentChunks ||
		manifest.ChunkCount != expectedCount {
		return documentManifest{}, ErrInvalidDocument
	}

	return manifest, nil
}

// Folder mengembalikan lokasi potongan dokumen, dipakai saat dokumen
// dicatat di tabel berkas supaya ikut terhapus bersama barisnya.
func (s *DocumentService) Folder(userID, documentID uuid.UUID) string {
	return s.documentDir(userID, documentID)
}

// Info mengembalikan nama dan tipe dokumen dari manifest.
func (s *DocumentService) Info(userID, documentID uuid.UUID) (string, string, error) {
	manifest, err := s.manifest(userID, documentID)
	if err != nil {
		return "", "", err
	}
	return manifest.Filename, manifest.ContentType, nil
}

// Lengkap bernilai true bila semua potongan sudah tersimpan.
func (s *DocumentService) Lengkap(userID, documentID uuid.UUID) (bool, error) {
	manifest, err := s.manifest(userID, documentID)
	if err != nil {
		return false, err
	}
	for i := 0; i < manifest.ChunkCount; i++ {
		if _, err := os.Stat(s.chunkPath(userID, documentID, i)); err != nil {
			return false, nil
		}
	}
	return true, nil
}

func (s *DocumentService) documentDir(
	userID, documentID uuid.UUID,
) string {
	return filepath.Join(s.root, userID.String(), documentID.String())
}

func (s *DocumentService) chunkPath(
	userID, documentID uuid.UUID,
	index int,
) string {
	return filepath.Join(
		s.documentDir(userID, documentID),
		fmt.Sprintf("%06d.json", index),
	)
}
