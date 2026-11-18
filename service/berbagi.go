package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/entity"
	"github.com/theRadn/KI-Tugas-1/repository"
	"github.com/theRadn/KI-Tugas-1/utils"
	"gorm.io/gorm"
)

var (
	ErrBerbagiTidakDitemukan = errors.New("permintaan berbagi tidak ditemukan")
	ErrBerbagiTidakDiizinkan = errors.New("tidak berhak mengakses berbagi ini")
	ErrBerbagiTidakAktif     = errors.New("permintaan berbagi tidak aktif")
	ErrBerbagiKedaluwarsa    = errors.New("berbagi sudah kedaluwarsa")
	ErrBerbagiCakupanSalah   = errors.New("cakupan data tidak dikenal")
	ErrBerbagiDiriSendiri    = errors.New("tidak bisa berbagi ke diri sendiri")
	ErrBerbagiDurasiSalah    = errors.New("durasi harus antara 1 dan 30 hari")
	ErrBerbagiBerkasSalah    = errors.New("berkas tidak termasuk dalam berbagi")
)

// tab yang boleh diminta, dipetakan ke scope enkripsi internal.
var tabKeScope = map[string][]string{
	"data-diri":      {"biodata"},
	"pendidikan":     {"pendidikan", "nilai_tka"},
	"keluarga":       {"biodata_keluarga"},
	"rumah":          {"rumah"},
	"catatan-khusus": {"catatan_khusus"},
	"pendukung":      {"pendukung_ekonomi"},
}

type BerbagiService interface {
	BuatPermintaan(ctx context.Context, pemintaID uuid.UUID, req dto.BuatPermintaanBerbagiRequest) (dto.PermintaanBerbagiResponse, error)
	DaftarMasuk(ctx context.Context, pemilikID uuid.UUID) ([]dto.PermintaanBerbagiResponse, error)
	DaftarKeluar(ctx context.Context, pemintaID uuid.UUID) ([]dto.PermintaanBerbagiResponse, error)
	Detail(ctx context.Context, userID, id uuid.UUID) (dto.PermintaanBerbagiResponse, error)
	Jawab(ctx context.Context, pemilikID, id uuid.UUID, req dto.JawabPermintaanBerbagiRequest) error
	Cabut(ctx context.Context, pemilikID, id uuid.UUID) error
	LihatBiodata(ctx context.Context, pemintaID, id uuid.UUID, tab string) (any, error)
	LihatBerkas(ctx context.Context, pemintaID, id, berkasID uuid.UUID) (dto.UnduhBerkasResponse, error)
	LihatChunk(ctx context.Context, pemintaID, id, berkasID uuid.UUID, index int) (dto.BerbagiChunkResponse, error)
}

type berbagiService struct {
	db   *gorm.DB
	repo *repository.BerbagiRepository
}

func NewBerbagiService(db *gorm.DB) BerbagiService {
	return &berbagiService{db: db, repo: repository.NewBerbagiRepository(db)}
}

// ---------- AAD & helper kecil ----------

func scopeAADBerbagi(userID uuid.UUID, scope, algorithm string) []byte {
	return []byte(fmt.Sprintf("user:%s:scope:%s:%s-dek", userID, scope, algorithm))
}

func shareAAD(permintaanID uuid.UUID, kind, refID, algorithm string, expiry time.Time) []byte {
	return []byte(fmt.Sprintf(
		"share:%s:%s:%s:%s:%d",
		permintaanID, kind, refID, algorithm, expiry.Unix(),
	))
}

func parseStringList(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func mustJSONList(v []string) string {
	if v == nil {
		v = []string{}
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func manifestPurpose(kode string) string {
	if kode == KodeDokumen {
		return "document-manifest"
	}
	return "video-manifest"
}

func chunkPurpose(kode string, i int) string {
	if kode == KodeDokumen {
		return fmt.Sprintf("document-chunk:%d", i)
	}
	return fmt.Sprintf("chunk:%d", i)
}

// ---------- Buat & jawab permintaan ----------

func (s *berbagiService) BuatPermintaan(
	ctx context.Context,
	pemintaID uuid.UUID,
	req dto.BuatPermintaanBerbagiRequest,
) (dto.PermintaanBerbagiResponse, error) {
	if req.DurasiHari < 1 || req.DurasiHari > 30 {
		return dto.PermintaanBerbagiResponse{}, ErrBerbagiDurasiSalah
	}
	for _, tab := range req.Cakupan {
		if _, ok := tabKeScope[tab]; !ok {
			return dto.PermintaanBerbagiResponse{}, ErrBerbagiCakupanSalah
		}
	}
	pemilik, err := s.repo.ByUsername(ctx, req.NamaPenggunaPemilik)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.PermintaanBerbagiResponse{}, ErrBerbagiTidakDitemukan
		}
		return dto.PermintaanBerbagiResponse{}, err
	}
	if pemilik.ID == pemintaID {
		return dto.PermintaanBerbagiResponse{}, ErrBerbagiDiriSendiri
	}

	p := &entity.PermintaanBerbagi{
		PemintaID:      pemintaID,
		PemilikID:      pemilik.ID,
		Status:         entity.StatusBerbagiMenunggu,
		CakupanDiminta: mustJSONList(req.Cakupan),
		BerkasDiminta:  mustJSONList(req.BerkasIDs),
		PesanPeminta:   req.Pesan,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return dto.PermintaanBerbagiResponse{}, err
	}
	return s.toResponse(ctx, p), nil
}

func (s *berbagiService) Jawab(
	ctx context.Context,
	pemilikID, id uuid.UUID,
	req dto.JawabPermintaanBerbagiRequest,
) error {
	perm, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBerbagiTidakDitemukan
		}
		return err
	}
	if perm.PemilikID != pemilikID {
		return ErrBerbagiTidakDiizinkan
	}
	if perm.Status != entity.StatusBerbagiMenunggu {
		return ErrBerbagiTidakAktif
	}

	if !req.Setuju {
		_, err := s.repo.UpdateStatus(ctx, id, entity.StatusBerbagiMenunggu, map[string]any{
			"status":        entity.StatusBerbagiDitolak,
			"pesan_pemilik": req.Pesan,
		})
		if err != nil {
			return err
		}
		return nil
	}

	if req.DurasiHari < 1 || req.DurasiHari > 30 {
		return ErrBerbagiDurasiSalah
	}

	diminta := parseStringList(perm.CakupanDiminta)
	disetujui := req.Cakupan
	if len(disetujui) == 0 {
		disetujui = diminta
	}
	for _, tab := range disetujui {
		if !contains(diminta, tab) {
			return ErrBerbagiCakupanSalah
		}
	}

	berkasDiminta := parseStringList(perm.BerkasDiminta)
	berkasDisetujui := req.BerkasIDs
	if len(berkasDisetujui) == 0 {
		berkasDisetujui = berkasDiminta
	}
	for _, bid := range berkasDisetujui {
		if !contains(berkasDiminta, bid) {
			return ErrBerbagiTidakDiizinkan
		}
	}

	expiry := time.Now().Add(time.Duration(req.DurasiHari) * 24 * time.Hour)

	// Ambil KEK peminta & pemilik di memori.
	pemintaKEK, err := unwrapUserKEK(s.db.WithContext(ctx), perm.PemintaID)
	if err != nil {
		return err
	}
	defer clear(pemintaKEK)

	// Siapkan semua kunci baru dulu, baru simpan dalam satu transaksi.
	var kunciBaru []entity.KunciBerbagi
	tabBerhasil := make([]string, 0, len(disetujui))

	for _, tab := range disetujui {
		adaSatu := false
		for _, scope := range tabKeScope[tab] {
			keys, err := s.transferScopeKeys(ctx, perm.PemilikID, perm.ID, scope, expiry, pemintaKEK)
			if err != nil {
				return err
			}
			if len(keys) > 0 {
				kunciBaru = append(kunciBaru, keys...)
				adaSatu = true
			}
		}
		if adaSatu {
			tabBerhasil = append(tabBerhasil, tab)
		}
	}

	berkasBerhasil := make([]string, 0, len(berkasDisetujui))
	for _, rawID := range berkasDisetujui {
		bid, err := uuid.Parse(rawID)
		if err != nil {
			continue
		}
		b, err := s.loadOwnerBerkas(ctx, perm.PemilikID, bid)
		if err != nil {
			if errors.Is(err, ErrBerbagiTidakDitemukan) {
				continue
			}
			return err
		}
		if b.Status != "terunggah" {
			continue
		}
		keys, err := s.transferBerkasKeys(ctx, perm.PemilikID, perm.ID, b, expiry, pemintaKEK)
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			kunciBaru = append(kunciBaru, keys...)
			berkasBerhasil = append(berkasBerhasil, bid.String())
		}
	}

	if len(tabBerhasil) == 0 && len(berkasBerhasil) == 0 {
		return errors.New("tidak ada data yang bisa dibagikan untuk cakupan tersebut")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := repository.NewBerbagiRepository(tx)
		rows, err := repoTx.UpdateStatus(ctx, perm.ID, entity.StatusBerbagiMenunggu, map[string]any{
			"status":            entity.StatusBerbagiDisetujui,
			"cakupan_disetujui": mustJSONList(tabBerhasil),
			"berkas_disetujui":  mustJSONList(berkasBerhasil),
			"pesan_pemilik":     req.Pesan,
			"berlaku_sampai":    &expiry,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrBerbagiTidakAktif
		}
		return repoTx.SimpanKunci(ctx, kunciBaru)
	})
}

// transferScopeKeys membungkus ulang DEK scope milik pemilik memakai KEK peminta.
// Return kosong bila scope tersebut belum pernah dibuat (belum ada data).
func (s *berbagiService) transferScopeKeys(
	ctx context.Context,
	pemilikID, permintaanID uuid.UUID,
	scope string,
	expiry time.Time,
	pemintaKEK []byte,
) ([]entity.KunciBerbagi, error) {
	var row entity.EncryptionKeySet
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND scope = ?", pemilikID, scope).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	pemilikKEK, err := unwrapUserKEK(s.db.WithContext(ctx), pemilikID)
	if err != nil {
		return nil, err
	}
	defer clear(pemilikKEK)

	wrappedMap := map[string][]byte{
		"AES": row.WrappedAES, "DES": row.WrappedDES, "RC4": row.WrappedRC4,
	}
	result := make([]entity.KunciBerbagi, 0, 3)
	for algo, wb := range wrappedMap {
		if len(wb) == 0 {
			continue
		}
		wrapped, err := utils.UnmarshalWrappedKey(wb)
		if err != nil {
			return nil, err
		}
		ownerDEK, err := utils.UnwrapKey(pemilikKEK, wrapped, scopeAADBerbagi(pemilikID, scope, algo))
		if err != nil {
			return nil, err
		}
		aad := shareAAD(permintaanID, "scope", scope, algo, expiry)
		rewrapped, err := utils.WrapKey(pemintaKEK, ownerDEK, aad)
		clear(ownerDEK)
		if err != nil {
			return nil, err
		}
		rewrappedBytes, err := utils.MarshalWrappedKey(rewrapped)
		if err != nil {
			return nil, err
		}
		result = append(result, entity.KunciBerbagi{
			PermintaanID:    permintaanID,
			Jenis:           "scope",
			RefID:           scope,
			Algoritma:       algo,
			KunciTerbungkus: rewrappedBytes,
		})
	}
	return result, nil
}

// transferBerkasKeys membungkus ulang kunci berkas (reguler atau potongan).
func (s *berbagiService) transferBerkasKeys(
	ctx context.Context,
	pemilikID, permintaanID uuid.UUID,
	berkas entity.Berkas,
	expiry time.Time,
	pemintaKEK []byte,
) ([]entity.KunciBerbagi, error) {
	if berkas.KodeBerkas == KodeVideo || berkas.KodeBerkas == KodeDokumen {
		return s.transferChunkKeys(ctx, pemilikID, permintaanID, berkas, expiry, pemintaKEK)
	}
	return s.transferSingleBerkasKeys(ctx, pemilikID, permintaanID, berkas, expiry, pemintaKEK)
}

func (s *berbagiService) transferSingleBerkasKeys(
	ctx context.Context,
	pemilikID, permintaanID uuid.UUID,
	berkas entity.Berkas,
	expiry time.Time,
	pemintaKEK []byte,
) ([]entity.KunciBerbagi, error) {
	pemilikKEK, err := unwrapUserKEK(s.db.WithContext(ctx), pemilikID)
	if err != nil {
		return nil, err
	}
	defer clear(pemilikKEK)

	result := make([]entity.KunciBerbagi, 0, len(berkas.Enkripsi))
	refID := berkas.ID.String()
	for _, e := range berkas.Enkripsi {
		if len(e.WrappedDEK) == 0 {
			continue // legacy, tidak bisa dibagikan
		}
		wrapped, err := utils.UnmarshalWrappedKey(e.WrappedDEK)
		if err != nil {
			return nil, err
		}
		ownerDEK, err := utils.UnwrapKey(pemilikKEK, wrapped, dekAAD(berkas.ID, "berkas", e.Algoritma))
		if err != nil {
			return nil, err
		}
		aad := shareAAD(permintaanID, "berkas", refID, e.Algoritma, expiry)
		rewrapped, err := utils.WrapKey(pemintaKEK, ownerDEK, aad)
		clear(ownerDEK)
		if err != nil {
			return nil, err
		}
		rewrappedBytes, err := utils.MarshalWrappedKey(rewrapped)
		if err != nil {
			return nil, err
		}
		result = append(result, entity.KunciBerbagi{
			PermintaanID:    permintaanID,
			Jenis:           "berkas",
			RefID:           refID,
			Algoritma:       e.Algoritma,
			KunciTerbungkus: rewrappedBytes,
		})
	}
	return result, nil
}

func (s *berbagiService) transferChunkKeys(
	ctx context.Context,
	pemilikID, permintaanID uuid.UUID,
	berkas entity.Berkas,
	expiry time.Time,
	pemintaKEK []byte,
) ([]entity.KunciBerbagi, error) {
	pemilikKEK, err := unwrapUserKEK(s.db.WithContext(ctx), pemilikID)
	if err != nil {
		return nil, err
	}
	defer clear(pemilikKEK)

	mpurpose := manifestPurpose(berkas.KodeBerkas)
	manifestFile, err := os.ReadFile(filepath.Join(berkas.FilePath, "manifest.json"))
	if err != nil {
		return nil, err
	}
	manifestPlain, err := unpackEncryptedPayload(
		s.db.WithContext(ctx), manifestFile, pemilikID, berkas.ID, mpurpose, "AES",
	)
	if err != nil {
		return nil, err
	}

	var count int
	if berkas.KodeBerkas == KodeDokumen {
		var m documentManifest
		if err := json.Unmarshal(manifestPlain, &m); err != nil {
			return nil, err
		}
		count = m.ChunkCount
	} else {
		var m videoManifest
		if err := json.Unmarshal(manifestPlain, &m); err != nil {
			return nil, err
		}
		count = m.ChunkCount
	}

	result := make([]entity.KunciBerbagi, 0)
	for i := 0; i < count; i++ {
		file, err := os.ReadFile(filepath.Join(berkas.FilePath, fmt.Sprintf("%06d.json", i)))
		if err != nil {
			return nil, err
		}
		var versions map[string]storedVideoCipher
		if err := json.Unmarshal(file, &versions); err != nil {
			return nil, err
		}
		purpose := chunkPurpose(berkas.KodeBerkas, i)
		refID := fmt.Sprintf("%s:%d", berkas.ID, i)
		for algo, v := range versions {
			if len(v.WrappedDEK) == 0 {
				continue
			}
			wrapped, err := utils.UnmarshalWrappedKey(v.WrappedDEK)
			if err != nil {
				return nil, err
			}
			ownerDEK, err := utils.UnwrapKey(pemilikKEK, wrapped, dekAAD(berkas.ID, purpose, algo))
			if err != nil {
				return nil, err
			}
			aad := shareAAD(permintaanID, "chunk", refID, algo, expiry)
			rewrapped, err := utils.WrapKey(pemintaKEK, ownerDEK, aad)
			clear(ownerDEK)
			if err != nil {
				return nil, err
			}
			rewrappedBytes, err := utils.MarshalWrappedKey(rewrapped)
			if err != nil {
				return nil, err
			}
			result = append(result, entity.KunciBerbagi{
				PermintaanID:    permintaanID,
				Jenis:           "chunk",
				RefID:           refID,
				Algoritma:       algo,
				KunciTerbungkus: rewrappedBytes,
			})
		}
	}
	return result, nil
}

// ---------- Cabut ----------

func (s *berbagiService) Cabut(ctx context.Context, pemilikID, id uuid.UUID) error {
	perm, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBerbagiTidakDitemukan
		}
		return err
	}
	if perm.PemilikID != pemilikID {
		return ErrBerbagiTidakDiizinkan
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := repository.NewBerbagiRepository(tx).HapusKunci(ctx, id); err != nil {
			return err
		}
		return tx.Model(&entity.PermintaanBerbagi{}).
			Where("id = ?", id).
			Update("status", entity.StatusBerbagiDicabut).Error
	})
}

// ---------- Daftar & detail ----------

func (s *berbagiService) DaftarMasuk(ctx context.Context, pemilikID uuid.UUID) ([]dto.PermintaanBerbagiResponse, error) {
	list, err := s.repo.DaftarMasuk(ctx, pemilikID)
	if err != nil {
		return nil, err
	}
	return s.mapList(ctx, list), nil
}

func (s *berbagiService) DaftarKeluar(ctx context.Context, pemintaID uuid.UUID) ([]dto.PermintaanBerbagiResponse, error) {
	list, err := s.repo.DaftarKeluar(ctx, pemintaID)
	if err != nil {
		return nil, err
	}
	return s.mapList(ctx, list), nil
}

func (s *berbagiService) Detail(ctx context.Context, userID, id uuid.UUID) (dto.PermintaanBerbagiResponse, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.PermintaanBerbagiResponse{}, ErrBerbagiTidakDitemukan
		}
		return dto.PermintaanBerbagiResponse{}, err
	}
	if p.PemintaID != userID && p.PemilikID != userID {
		return dto.PermintaanBerbagiResponse{}, ErrBerbagiTidakDiizinkan
	}
	return s.toResponse(ctx, p), nil
}

func (s *berbagiService) mapList(ctx context.Context, list []entity.PermintaanBerbagi) []dto.PermintaanBerbagiResponse {
	out := make([]dto.PermintaanBerbagiResponse, 0, len(list))
	for i := range list {
		out = append(out, s.toResponse(ctx, &list[i]))
	}
	return out
}

func (s *berbagiService) toResponse(ctx context.Context, p *entity.PermintaanBerbagi) dto.PermintaanBerbagiResponse {
	// Cek kedaluwarsa secara pasif (tanpa mengubah DB di jalur baca).
	status := p.Status
	aktif := false
	if status == entity.StatusBerbagiDisetujui {
		if p.BerlakuSampai != nil && !time.Now().Before(*p.BerlakuSampai) {
			status = entity.StatusBerbagiKedaluwarsa
		} else {
			aktif = true
		}
	}
	resp := dto.PermintaanBerbagiResponse{
		ID:               p.ID.String(),
		PemintaID:        p.PemintaID.String(),
		PemilikID:        p.PemilikID.String(),
		Status:           status,
		CakupanDiminta:   parseStringList(p.CakupanDiminta),
		CakupanDisetujui: parseStringList(p.CakupanDisetujui),
		BerkasDiminta:    parseStringList(p.BerkasDiminta),
		BerkasDisetujui:  parseStringList(p.BerkasDisetujui),
		PesanPeminta:     p.PesanPeminta,
		PesanPemilik:     p.PesanPemilik,
		BerlakuSampai:    p.BerlakuSampai,
		DibuatPada:       p.DibuatPada,
		DiubahPada:       p.DiubahPada,
		Aktif:            aktif,
	}
	if u, err := s.repo.ByID(ctx, p.PemintaID); err == nil {
		resp.PemintaNama = u.NamaPengguna
	}
	if u, err := s.repo.ByID(ctx, p.PemilikID); err == nil {
		resp.PemilikNama = u.NamaPengguna
	}
	return resp
}

// ---------- Akses data ----------

// loadActiveShare memastikan permintaan disetujui, bukan milik sendiri,
// dan belum kedaluwarsa. Bila sudah lewat waktu, baris kunci dihapus.
func (s *berbagiService) loadActiveShare(
	ctx context.Context, pemintaID, permintaanID uuid.UUID,
) (*entity.PermintaanBerbagi, error) {
	p, err := s.repo.Get(ctx, permintaanID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBerbagiTidakDitemukan
		}
		return nil, err
	}
	if p.PemintaID != pemintaID {
		return nil, ErrBerbagiTidakDiizinkan
	}
	if p.Status != entity.StatusBerbagiDisetujui {
		if p.Status == entity.StatusBerbagiKedaluwarsa {
			return nil, ErrBerbagiKedaluwarsa
		}
		return nil, ErrBerbagiTidakAktif
	}
	if p.BerlakuSampai == nil || !time.Now().Before(*p.BerlakuSampai) {
		_ = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := repository.NewBerbagiRepository(tx).HapusKunci(ctx, p.ID); err != nil {
				return err
			}
			return tx.Model(&entity.PermintaanBerbagi{}).
				Where("id = ?", p.ID).
				Update("status", entity.StatusBerbagiKedaluwarsa).Error
		})
		return nil, ErrBerbagiKedaluwarsa
	}
	return p, nil
}

// muatScopeKeys membuka bungkus semua kunci scope yang disetujui dan
// menyiapkannya sebagai entity.KeyMaterial milik pemilik asli.
func (s *berbagiService) muatScopeKeys(
	ctx context.Context, perm *entity.PermintaanBerbagi, scopes []string,
) (map[string]entity.KeyMaterial, []byte, error) {
	pemintaKEK, err := unwrapUserKEK(s.db.WithContext(ctx), perm.PemintaID)
	if err != nil {
		return nil, nil, err
	}

	materials := map[string]entity.KeyMaterial{}
	for _, scope := range scopes {
		rows, err := s.repo.KunciScope(ctx, perm.ID, scope)
		if err != nil {
			clear(pemintaKEK)
			return nil, nil, err
		}
		if len(rows) == 0 {
			clear(pemintaKEK)
			return nil, nil, ErrBerbagiTidakDiizinkan
		}
		keyMap := map[string][]byte{}
		for _, k := range rows {
			wrapped, err := utils.UnmarshalWrappedKey(k.KunciTerbungkus)
			if err != nil {
				clear(pemintaKEK)
				return nil, nil, err
			}
			aad := shareAAD(perm.ID, "scope", scope, k.Algoritma, *perm.BerlakuSampai)
			dek, err := utils.UnwrapKey(pemintaKEK, wrapped, aad)
			if err != nil {
				clear(pemintaKEK)
				return nil, nil, err
			}
			keyMap[k.Algoritma] = dek
		}
		materials[scope] = entity.KeyMaterial{
			UserID: perm.PemilikID,
			Scope:  scope,
			Keys:   keyMap,
		}
	}
	return materials, pemintaKEK, nil
}

func (s *berbagiService) LihatBiodata(
	ctx context.Context, pemintaID, id uuid.UUID, tab string,
) (any, error) {
	scopes, ok := tabKeScope[tab]
	if !ok {
		return nil, ErrBerbagiCakupanSalah
	}
	perm, err := s.loadActiveShare(ctx, pemintaID, id)
	if err != nil {
		return nil, err
	}
	disetujui := parseStringList(perm.CakupanDisetujui)
	if !contains(disetujui, tab) {
		return nil, ErrBerbagiTidakDiizinkan
	}

	materials, pemintaKEK, err := s.muatScopeKeys(ctx, perm, scopes)
	if err != nil {
		return nil, err
	}
	defer func() {
		clear(pemintaKEK)
		for _, m := range materials {
			for _, k := range m.Keys {
				clear(k)
			}
		}
	}()

	// Query langsung dengan material pemilik → serializer memakai kunci asli.
	qctx := entity.WithKeyMaterial(ctx, materials)
	var b entity.Biodata
	err = s.db.WithContext(qctx).
		Preload("Pendidikan.NilaiTKA").
		Preload("Keluarga").
		Preload("Rumah").
		Preload("CatatanKhusus").
		Preload("PendukungEkonomi").
		Where("user_id = ?", perm.PemilikID).
		First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBerbagiTidakDitemukan
		}
		return nil, err
	}

	switch tab {
	case "data-diri":
		return map[string]any{
			"no_pendaftaran": b.NoPendaftaran,
			"nama_ijazah":    b.NamaIjazah,
			"nama_ktm":       b.NamaKTM,
			"program_studi":  b.ProgramStudi,
			"jenis_kelamin":  b.JenisKelamin,
			"agama":          b.Agama,
			"tanggal_lahir":  b.TanggalLahir,
			"email":          b.Email,
			"no_hp":          b.NoHP,
			"alamat_ktp":     b.AlamatKTP,
			"alamat_domisili": b.AlamatDomisili,
			"diubah_pada":    b.DiubahPada,
		}, nil
	case "pendidikan":
		return b.Pendidikan, nil
	case "keluarga":
		return b.Keluarga, nil
	case "rumah":
		return b.Rumah, nil
	case "catatan-khusus":
		return b.CatatanKhusus, nil
	case "pendukung":
		return b.PendukungEkonomi, nil
	}
	return nil, ErrBerbagiCakupanSalah
}

func (s *berbagiService) loadOwnerBerkas(
	ctx context.Context, ownerID, berkasID uuid.UUID,
) (entity.Berkas, error) {
	var b entity.Berkas
	err := s.db.WithContext(ctx).
		Joins("JOIN biodata ON biodata.id = berkas.biodata_id").
		Where("biodata.user_id = ? AND berkas.id = ?", ownerID, berkasID).
		Preload("Enkripsi").First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Berkas{}, ErrBerbagiTidakDitemukan
	}
	return b, err
}

// LihatBerkas mengembalikan isi berkas reguler (gambar) sebagai base64.
func (s *berbagiService) LihatBerkas(
	ctx context.Context, pemintaID, id, berkasID uuid.UUID,
) (dto.UnduhBerkasResponse, error) {
	perm, err := s.loadActiveShare(ctx, pemintaID, id)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	if !contains(parseStringList(perm.BerkasDisetujui), berkasID.String()) {
		return dto.UnduhBerkasResponse{}, ErrBerbagiBerkasSalah
	}
	b, err := s.loadOwnerBerkas(ctx, perm.PemilikID, berkasID)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	if b.KodeBerkas == KodeVideo || b.KodeBerkas == KodeDokumen {
		return dto.UnduhBerkasResponse{}, errors.New("berkas potongan, gunakan endpoint chunk")
	}

	var enc entity.EnkripsiBerkas
	for _, e := range b.Enkripsi {
		if e.Algoritma == "AES" {
			enc = e
			break
		}
	}
	if len(enc.WrappedDEK) == 0 {
		return dto.UnduhBerkasResponse{}, errors.New("kunci berkas tidak tersedia")
	}

	rows, err := s.repo.KunciRef(ctx, perm.ID, "berkas", berkasID.String())
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	var kunciAES *entity.KunciBerbagi
	for i := range rows {
		if rows[i].Algoritma == "AES" {
			kunciAES = &rows[i]
			break
		}
	}
	if kunciAES == nil {
		return dto.UnduhBerkasResponse{}, ErrBerbagiBerkasSalah
	}

	pemintaKEK, err := unwrapUserKEK(s.db.WithContext(ctx), pemintaID)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	defer clear(pemintaKEK)

	wrapped, err := utils.UnmarshalWrappedKey(kunciAES.KunciTerbungkus)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	aad := shareAAD(perm.ID, "berkas", berkasID.String(), "AES", *perm.BerlakuSampai)
	ownerDEK, err := utils.UnwrapKey(pemintaKEK, wrapped, aad)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	defer clear(ownerDEK)

	macKey, err := deriveFileMACKey(ownerDEK, dekAAD(b.ID, "berkas", "AES"))
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	defer clear(macKey)

	raw, err := os.ReadFile(enc.CiphertextPath)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	var data utils.EncryptedData
	if err := json.Unmarshal(raw, &data); err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	plaintext, err := utils.Decrypt(data, ownerDEK, macKey)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}

	return dto.UnduhBerkasResponse{
		NamaAsli:  b.NamaAsli,
		TipeMIME:  b.TipeMIME,
		Algoritma: "AES",
		IsiBase64: base64.StdEncoding.EncodeToString(plaintext),
	}, nil
}

// LihatChunk mendekripsi satu potongan video/dokumen.
func (s *berbagiService) LihatChunk(
	ctx context.Context, pemintaID, id, berkasID uuid.UUID, index int,
) (dto.BerbagiChunkResponse, error) {
	perm, err := s.loadActiveShare(ctx, pemintaID, id)
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	if !contains(parseStringList(perm.BerkasDisetujui), berkasID.String()) {
		return dto.BerbagiChunkResponse{}, ErrBerbagiBerkasSalah
	}
	b, err := s.loadOwnerBerkas(ctx, perm.PemilikID, berkasID)
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	if b.KodeBerkas != KodeVideo && b.KodeBerkas != KodeDokumen {
		return dto.BerbagiChunkResponse{}, errors.New("berkas bukan potongan")
	}

	mpurpose := manifestPurpose(b.KodeBerkas)
	manifestFile, err := os.ReadFile(filepath.Join(b.FilePath, "manifest.json"))
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	manifestPlain, err := unpackEncryptedPayload(
		s.db.WithContext(ctx), manifestFile, perm.PemilikID, b.ID, mpurpose, "AES",
	)
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	var count int
	if b.KodeBerkas == KodeDokumen {
		var m documentManifest
		if err := json.Unmarshal(manifestPlain, &m); err != nil {
			return dto.BerbagiChunkResponse{}, err
		}
		count = m.ChunkCount
	} else {
		var m videoManifest
		if err := json.Unmarshal(manifestPlain, &m); err != nil {
			return dto.BerbagiChunkResponse{}, err
		}
		count = m.ChunkCount
	}
	if index < 0 || index >= count {
		return dto.BerbagiChunkResponse{}, errors.New("index chunk tidak valid")
	}

	refID := fmt.Sprintf("%s:%d", b.ID, index)
	rows, err := s.repo.KunciRef(ctx, perm.ID, "chunk", refID)
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	var kunci *entity.KunciBerbagi
	for i := range rows {
		if rows[i].Algoritma == "AES" {
			kunci = &rows[i]
			break
		}
	}
	if kunci == nil {
		return dto.BerbagiChunkResponse{}, ErrBerbagiBerkasSalah
	}

	chunkFile, err := os.ReadFile(filepath.Join(b.FilePath, fmt.Sprintf("%06d.json", index)))
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	var versions map[string]storedVideoCipher
	if err := json.Unmarshal(chunkFile, &versions); err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	v, ok := versions["AES"]
	if !ok {
		return dto.BerbagiChunkResponse{}, errors.New("chunk tidak tersedia")
	}

	pemintaKEK, err := unwrapUserKEK(s.db.WithContext(ctx), pemintaID)
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	defer clear(pemintaKEK)

	wrapped, err := utils.UnmarshalWrappedKey(kunci.KunciTerbungkus)
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	aad := shareAAD(perm.ID, "chunk", refID, "AES", *perm.BerlakuSampai)
	ownerDEK, err := utils.UnwrapKey(pemintaKEK, wrapped, aad)
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	defer clear(ownerDEK)

	macKey, err := deriveFileMACKey(ownerDEK, dekAAD(b.ID, chunkPurpose(b.KodeBerkas, index), "AES"))
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}
	defer clear(macKey)

	plaintext, err := utils.Decrypt(v.EncryptedData, ownerDEK, macKey)
	if err != nil {
		return dto.BerbagiChunkResponse{}, err
	}

	return dto.BerbagiChunkResponse{
		BerkasID:   b.ID.String(),
		Index:      index,
		TipeMIME:   b.TipeMIME,
		UkuranByte: len(plaintext),
		IsiBase64:  base64.StdEncoding.EncodeToString(plaintext),
	}, nil
}
