package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // daftarkan dekoder untuk cek dimensi foto
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/entity"
	"github.com/theRadn/KI-Tugas-1/repository"
	"github.com/theRadn/KI-Tugas-1/utils"
	"gorm.io/gorm"
)

var (
	ErrKodeBerkas      = errors.New("kode berkas tidak dikenal")
	ErrBerkasTidakSah  = errors.New("berkas tidak sesuai ketentuan")
	ErrAlgoritma       = errors.New("algoritma harus AES, DES, atau RC4")
	ErrKunciBerkas     = errors.New("kunci enkripsi berkas belum diatur di server")
	ErrBerkasNotFound  = errors.New("berkas tidak ditemukan")
	folderPenyimpanan  = filepath.Join("storage", "berkas")
	tipeGambar         = []string{"image/jpeg", "image/png"}
	tipeVideo          = []string{"video/mp4", "video/webm"}
	algoritmaBerkas    = []string{"AES", "DES", "RC4"}
	panjangKunciBerkas = map[string]int{"AES": 32, "DES": 8}
)

type aturanBerkas struct {
	tipe         []string
	minKB, maxKB int64
	persegi      int // dimensi wajib (px), 0 berarti bebas
}

// Aturan mengikuti kontrak-data.md bagian 5.2, 5.5, dan 6.
// Dokumen dan video (bagian "Lainnya") belum didukung karena melebihi
// batas 1 MiB amplop enkripsi in-transit.
func aturanUntuk(kode string) (aturanBerkas, bool) {
	standar := aturanBerkas{tipe: tipeGambar, minKB: 200, maxKB: 500}
	if n, ok := strings.CutPrefix(kode, "rapor_sem_"); ok {
		return standar, len(n) == 1 && n >= "1" && n <= "7"
	}
	switch kode {
	case "foto":
		return aturanBerkas{tipe: tipeGambar, maxKB: 500, persegi: 400}, true
	case "ijazah_skl", "kartu_keluarga", "akta_kelahiran", "ktp", "bpjs":
		return aturanBerkas{tipe: tipeGambar, maxKB: 500}, true
	case "foto_kipk_sktm", "surat_tidak_menarik_biaya", "surat_tidak_memalsukan",
		"surat_kode_etik", "sertifikat_nilai_tka",
		"bukti_penghasilan_ortu", "surat_keterangan_pekerjaan", "foto_aktivitas_ortu",
		"tagihan_listrik", "bukti_pkb_roda_dua", "bukti_pkb_roda_empat", "pbb",
		"foto_rumah_depan", "foto_ruang_tamu", "foto_kamar_mandi",
		"foto_kamar_1", "foto_kamar_2":
		return standar, true
	}
	return aturanBerkas{}, false
}

func kunciBerkas(algoritma string) (utils.Algorithm, []byte, error) {
	alg, namaEnv := algoritmaDanSecret(algoritma)
	if alg == "" {
		return "", nil, ErrAlgoritma
	}
	kunci := []byte(os.Getenv(namaEnv))
	if want, ok := panjangKunciBerkas[algoritma]; (ok && len(kunci) != want) || len(kunci) == 0 {
		return "", nil, ErrKunciBerkas
	}
	return alg, kunci, nil
}

type BerkasService struct {
	db          *gorm.DB
	biodataRepo *repository.BiodataRepository
}

func NewBerkasService(db *gorm.DB) *BerkasService {
	return &BerkasService{db: db, biodataRepo: repository.NewBiodataRepository(db)}
}

func keResponse(b entity.Berkas) dto.BerkasResponse {
	return dto.BerkasResponse{
		ID: b.ID.String(), KodeBerkas: b.KodeBerkas, NamaAsli: b.NamaAsli,
		UkuranByte: b.UkuranByte, TipeMIME: b.TipeMIME, Status: b.Status,
		DiunggahPada: b.DiunggahPada,
	}
}

func (s *BerkasService) Daftar(ctx context.Context, userID uuid.UUID) (dto.BerkasListResponse, error) {
	var daftar []entity.Berkas
	err := s.db.WithContext(ctx).Model(&entity.Berkas{}).
		Joins("JOIN biodata ON biodata.id = berkas.biodata_id").
		Where("biodata.user_id = ?", userID).
		Order("berkas.kode_berkas").Find(&daftar).Error
	if err != nil {
		return dto.BerkasListResponse{}, err
	}
	res := dto.BerkasListResponse{Berkas: make([]dto.BerkasResponse, 0, len(daftar))}
	for _, b := range daftar {
		res.Berkas = append(res.Berkas, keResponse(b))
		res.TotalUkuranByte += b.UkuranByte
	}
	res.Jumlah = len(res.Berkas)
	return res, nil
}

// periksaIsi memastikan isi berkas benar-benar sesuai aturan, tidak hanya
// percaya pada nama atau tipe yang dikirim klien.
func periksaIsi(aturan aturanBerkas, req dto.UnggahBerkasRequest) ([]byte, error) {
	isi, err := base64.StdEncoding.DecodeString(req.IsiBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: isi berkas tidak valid", ErrBerkasTidakSah)
	}
	kb := int64(len(isi)) / 1024
	if len(isi) == 0 || kb < aturan.minKB || int64(len(isi)) > aturan.maxKB*1024 {
		return nil, fmt.Errorf("%w: ukuran harus %d sampai %d KB", ErrBerkasTidakSah, aturan.minKB, aturan.maxKB)
	}
	asli := http.DetectContentType(isi)
	if !slices.Contains(aturan.tipe, asli) || asli != req.TipeMIME {
		return nil, fmt.Errorf("%w: format berkas tidak sesuai", ErrBerkasTidakSah)
	}
	if aturan.persegi > 0 {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(isi))
		if err != nil || cfg.Width != aturan.persegi || cfg.Height != aturan.persegi {
			return nil, fmt.Errorf("%w: dimensi foto harus %dx%d px", ErrBerkasTidakSah, aturan.persegi, aturan.persegi)
		}
	}
	return isi, nil
}

// Unggah mengenkripsi berkas dengan ketiga algoritma dan hanya menyimpan
// ciphertext-nya. Berkas asli tidak pernah ditulis ke disk. Unggahan
// ulang dengan kode yang sama menggantikan berkas lama.
func (s *BerkasService) Unggah(ctx context.Context, userID uuid.UUID, kode string, req dto.UnggahBerkasRequest) (dto.UnggahBerkasResponse, error) {
	aturan, ok := aturanUntuk(kode)
	if !ok {
		return dto.UnggahBerkasResponse{}, ErrKodeBerkas
	}
	isi, err := periksaIsi(aturan, req)
	if err != nil {
		return dto.UnggahBerkasResponse{}, err
	}

	type hasilEnkripsi struct {
		alg  string
		mode utils.Algorithm
		data utils.EncryptedData
	}
	var semua []hasilEnkripsi
	for _, a := range algoritmaBerkas {
		alg, kunci, err := kunciBerkas(a)
		if err != nil {
			return dto.UnggahBerkasResponse{}, err
		}
		data, err := utils.Encrypt(alg, kunci, kunci, isi)
		if err != nil {
			return dto.UnggahBerkasResponse{}, err
		}
		semua = append(semua, hasilEnkripsi{alg: a, mode: alg, data: data})
	}

	biodata, err := s.biodataRepo.PastikanBiodata(ctx, userID)
	if err != nil {
		return dto.UnggahBerkasResponse{}, err
	}

	sekarang := time.Now()
	berkas := entity.Berkas{
		BiodataID: biodata.ID, KodeBerkas: kode, NamaAsli: filepath.Base(req.NamaAsli),
		UkuranByte: int64(len(isi)), TipeMIME: req.TipeMIME,
		Status: "terunggah", DiunggahPada: &sekarang,
	}
	berkas.ID, err = uuid.NewV7()
	if err != nil {
		return dto.UnggahBerkasResponse{}, err
	}
	folder := filepath.Join(folderPenyimpanan, berkas.ID.String())

	var lama []entity.Berkas
	res := dto.UnggahBerkasResponse{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("biodata_id = ? AND kode_berkas = ?", biodata.ID, kode).Find(&lama).Error; err != nil {
			return err
		}
		for _, b := range lama {
			if err := hapusBaris(tx, b.ID); err != nil {
				return err
			}
		}
		if err := tx.Create(&berkas).Error; err != nil {
			return err
		}
		if err := os.MkdirAll(folder, 0o700); err != nil {
			return err
		}
		for _, h := range semua {
			isiJSON, err := json.Marshal(h.data)
			if err != nil {
				return err
			}
			jalur := filepath.Join(folder, h.alg+".json")
			if err := os.WriteFile(jalur, isiJSON, 0o600); err != nil {
				return err
			}
			var iv *string
			if len(h.data.IV) > 0 {
				v := base64.StdEncoding.EncodeToString(h.data.IV)
				iv = &v
			}
			enk := entity.EnkripsiBerkas{
				BerkasID: berkas.ID, Algoritma: h.alg, Mode: string(h.mode), IV: iv,
				CiphertextPath: jalur, UkuranCiphertextByte: int64(len(h.data.Ciphertext)),
			}
			if err := tx.Create(&enk).Error; err != nil {
				return err
			}
			res.Enkripsi = append(res.Enkripsi, dto.EnkripsiBerkasResponse{
				Algoritma: enk.Algoritma, Mode: enk.Mode, UkuranCiphertextByte: enk.UkuranCiphertextByte,
			})
		}
		return nil
	})
	if err != nil {
		_ = os.RemoveAll(folder)
		return dto.UnggahBerkasResponse{}, err
	}
	for _, b := range lama {
		hapusFolder(b)
	}
	res.BerkasResponse = keResponse(berkas)
	return res, nil
}

// hapusFolder membuang ciphertext di disk: gambar di storage/berkas/<id>,
// video di folder potongan yang tercatat di FilePath.
func hapusFolder(b entity.Berkas) {
	_ = os.RemoveAll(filepath.Join(folderPenyimpanan, b.ID.String()))
	if b.FilePath != "" {
		_ = os.RemoveAll(b.FilePath)
	}
}

// Video diunggah per potongan lewat /api/videos. Batas 10 MB menjaga
// jumlah potongan di bawah limiter global (100 permintaan per menit).
const (
	KodeVideo     = "video_dokumentasi"
	MaksVideoByte = 10 << 20
)

func ValidasiVideo(tipe string, ukuran int64) error {
	if !slices.Contains(tipeVideo, tipe) {
		return fmt.Errorf("%w: format video harus MP4 atau WebM", ErrBerkasTidakSah)
	}
	if ukuran <= 0 || ukuran > MaksVideoByte {
		return fmt.Errorf("%w: ukuran video maksimal 10 MB", ErrBerkasTidakSah)
	}
	return nil
}

// CatatVideo membuat baris berkas untuk video baru (status "mengunggah")
// dan menggantikan video sebelumnya milik pengguna yang sama.
func (s *BerkasService) CatatVideo(ctx context.Context, userID, videoID uuid.UUID, nama, tipe string, ukuran int64, folder string) error {
	biodata, err := s.biodataRepo.PastikanBiodata(ctx, userID)
	if err != nil {
		return err
	}
	var lama []entity.Berkas
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("biodata_id = ? AND kode_berkas = ?", biodata.ID, KodeVideo).Find(&lama).Error; err != nil {
			return err
		}
		for _, b := range lama {
			if err := hapusBaris(tx, b.ID); err != nil {
				return err
			}
		}
		berkas := entity.Berkas{
			BiodataID: biodata.ID, KodeBerkas: KodeVideo, NamaAsli: filepath.Base(nama),
			UkuranByte: ukuran, TipeMIME: tipe, Status: "mengunggah", FilePath: folder,
		}
		berkas.ID = videoID
		return tx.Create(&berkas).Error
	})
	if err != nil {
		return err
	}
	for _, b := range lama {
		hapusFolder(b)
	}
	return nil
}

func (s *BerkasService) videoMilik(ctx context.Context, userID, videoID uuid.UUID) (entity.Berkas, error) {
	b, err := s.milikPengguna(ctx, userID, videoID.String())
	if err == nil && b.KodeBerkas != KodeVideo {
		return entity.Berkas{}, ErrBerkasNotFound
	}
	return b, err
}

// CekAwalVideo memeriksa isi asli potongan pertama, bukan hanya tipe yang
// diakui klien.
func (s *BerkasService) CekAwalVideo(ctx context.Context, userID, videoID uuid.UUID, potongan []byte) error {
	b, err := s.videoMilik(ctx, userID, videoID)
	if err != nil {
		return err
	}
	if http.DetectContentType(potongan) != b.TipeMIME {
		return fmt.Errorf("%w: isi berkas bukan video %s", ErrBerkasTidakSah, b.TipeMIME)
	}
	return nil
}

func (s *BerkasService) TandaiVideoTerunggah(ctx context.Context, userID, videoID uuid.UUID) error {
	if _, err := s.videoMilik(ctx, userID, videoID); err != nil {
		return err
	}
	sekarang := time.Now()
	return s.db.WithContext(ctx).Model(&entity.Berkas{}).Where("id = ?", videoID).
		Updates(map[string]any{"status": "terunggah", "diunggah_pada": &sekarang}).Error
}

// hapusBaris menghapus berkas beserta enkripsi dan hasil analisisnya.
func hapusBaris(tx *gorm.DB, berkasID uuid.UUID) error {
	var idHasil []uuid.UUID
	if err := tx.Model(&entity.HasilAnalisis{}).Where("berkas_id = ?", berkasID).Pluck("id", &idHasil).Error; err != nil {
		return err
	}
	if len(idHasil) > 0 {
		if err := tx.Where("hasil_analisis_id IN ?", idHasil).Delete(&entity.HasilAnalisisAlgoritma{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", idHasil).Delete(&entity.HasilAnalisis{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("berkas_id = ?", berkasID).Delete(&entity.EnkripsiBerkas{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", berkasID).Delete(&entity.Berkas{}).Error
}

func (s *BerkasService) milikPengguna(ctx context.Context, userID uuid.UUID, rawID string) (entity.Berkas, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return entity.Berkas{}, ErrBerkasNotFound
	}
	var b entity.Berkas
	err = s.db.WithContext(ctx).Joins("JOIN biodata ON biodata.id = berkas.biodata_id").
		Where("biodata.user_id = ? AND berkas.id = ?", userID, id).
		Preload("Enkripsi").First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Berkas{}, ErrBerkasNotFound
	}
	return b, err
}

func bacaEnkripsi(b entity.Berkas, algoritma string) (utils.EncryptedData, error) {
	for _, e := range b.Enkripsi {
		if e.Algoritma != algoritma {
			continue
		}
		isi, err := os.ReadFile(e.CiphertextPath)
		if err != nil {
			return utils.EncryptedData{}, err
		}
		var data utils.EncryptedData
		return data, json.Unmarshal(isi, &data)
	}
	return utils.EncryptedData{}, ErrBerkasNotFound
}

func (s *BerkasService) Unduh(ctx context.Context, userID uuid.UUID, rawID, algoritma string) (dto.UnduhBerkasResponse, error) {
	alg, kunci, err := kunciBerkas(algoritma)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	b, err := s.milikPengguna(ctx, userID, rawID)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	data, err := bacaEnkripsi(b, algoritma)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	if data.Algorithm != alg {
		return dto.UnduhBerkasResponse{}, errors.New("metadata ciphertext tidak sesuai")
	}
	isi, err := utils.Decrypt(data, kunci, kunci)
	if err != nil {
		return dto.UnduhBerkasResponse{}, err
	}
	return dto.UnduhBerkasResponse{
		NamaAsli: b.NamaAsli, TipeMIME: b.TipeMIME, Algoritma: algoritma,
		IsiBase64: base64.StdEncoding.EncodeToString(isi),
	}, nil
}

func (s *BerkasService) Ciphertext(ctx context.Context, userID uuid.UUID, rawID, algoritma string) (dto.CiphertextResponse, error) {
	if !slices.Contains(algoritmaBerkas, algoritma) {
		return dto.CiphertextResponse{}, ErrAlgoritma
	}
	b, err := s.milikPengguna(ctx, userID, rawID)
	if err != nil {
		return dto.CiphertextResponse{}, err
	}
	data, err := bacaEnkripsi(b, algoritma)
	if err != nil {
		return dto.CiphertextResponse{}, err
	}
	return dto.CiphertextResponse{
		BerkasID: b.ID.String(), Algoritma: algoritma, Encoding: "base64",
		Ciphertext: base64.StdEncoding.EncodeToString(data.Ciphertext),
	}, nil
}

func (s *BerkasService) Hapus(ctx context.Context, userID uuid.UUID, rawID string) error {
	b, err := s.milikPengguna(ctx, userID, rawID)
	if err != nil {
		return err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return hapusBaris(tx, b.ID)
	})
	if err != nil {
		return err
	}
	hapusFolder(b)
	return nil
}
