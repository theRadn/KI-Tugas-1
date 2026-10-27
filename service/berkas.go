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
	tipeVideo          = []string{"video/mp4", "video/webm", "video/x-matroska"}
	algoritmaBerkas    = []string{"AES", "DES", "RC4"}
	panjangKunciBerkas = map[string]int{"AES": 32, "DES": 8}
)

type aturanBerkas struct {
	tipe    []string
	maxKB   int64
	persegi int // dimensi wajib (px), 0 berarti bebas
}

// Aturan mengikuti kontrak-data.md bagian 5.2, 5.5, dan 6.
// Dokumen dan video (bagian "Lainnya") diunggah per potongan lewat
// /api/documents dan /api/videos, bukan lewat aturan di sini.
func aturanUntuk(kode string) (aturanBerkas, bool) {
	// Tanpa batas minimum: pengguna tidak bisa menyesuaikan ukuran berkas
	// yang terlalu kecil, jadi cukup dibatasi atasnya.
	standar := aturanBerkas{tipe: tipeGambar, maxKB: 500}
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
	if len(isi) == 0 || int64(len(isi)) > aturan.maxKB*1024 {
		return nil, fmt.Errorf("%w: ukuran berkas maksimal %d KB", ErrBerkasTidakSah, aturan.maxKB)
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

// Video dan dokumen diunggah per potongan 512 KB lewat /api/videos dan
// /api/documents. Rute potongan memakai limiter sendiri (lihat routes),
// jadi tidak terbatas oleh limiter global 100 permintaan per menit.
const (
	KodeVideo       = "video_dokumentasi"
	MaksVideoByte   = 50 << 20
	KodeDokumen     = "dokumen_lain"
	MaksDokumenByte = 10 << 20
)

func ValidasiVideo(tipe string, ukuran int64) error {
	if !slices.Contains(tipeVideo, tipe) {
		return fmt.Errorf("%w: format video harus MP4 atau WebM", ErrBerkasTidakSah)
	}
	if ukuran <= 0 || ukuran > MaksVideoByte {
		return fmt.Errorf("%w: ukuran video maksimal 50 MB", ErrBerkasTidakSah)
	}
	return nil
}

func ValidasiDokumen(ukuran int64) error {
	if ukuran <= 0 || ukuran > MaksDokumenByte {
		return fmt.Errorf("%w: ukuran dokumen maksimal 10 MB", ErrBerkasTidakSah)
	}
	return nil
}

func (s *BerkasService) CatatVideo(ctx context.Context, userID, videoID uuid.UUID, nama, tipe string, ukuran int64, folder string) error {
	return s.catatPotongan(ctx, userID, videoID, KodeVideo, nama, tipe, ukuran, folder)
}

func (s *BerkasService) CatatDokumen(ctx context.Context, userID, dokumenID uuid.UUID, nama, tipe string, ukuran int64, folder string) error {
	return s.catatPotongan(ctx, userID, dokumenID, KodeDokumen, nama, tipe, ukuran, folder)
}

// catatPotongan membuat baris berkas untuk unggahan per potongan (status
// "mengunggah") dan menggantikan unggahan sebelumnya dengan kode yang sama.
func (s *BerkasService) catatPotongan(ctx context.Context, userID, id uuid.UUID, kode, nama, tipe string, ukuran int64, folder string) error {
	biodata, err := s.biodataRepo.PastikanBiodata(ctx, userID)
	if err != nil {
		return err
	}
	var lama []entity.Berkas
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("biodata_id = ? AND kode_berkas = ?", biodata.ID, kode).Find(&lama).Error; err != nil {
			return err
		}
		for _, b := range lama {
			if err := hapusBaris(tx, b.ID); err != nil {
				return err
			}
		}
		berkas := entity.Berkas{
			BiodataID: biodata.ID, KodeBerkas: kode, NamaAsli: filepath.Base(nama),
			UkuranByte: ukuran, TipeMIME: tipe, Status: "mengunggah", FilePath: folder,
		}
		berkas.ID = id
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

func (s *BerkasService) berkasDenganKode(ctx context.Context, userID, id uuid.UUID, kode string) (entity.Berkas, error) {
	b, err := s.milikPengguna(ctx, userID, id.String())
	if err == nil && b.KodeBerkas != kode {
		return entity.Berkas{}, ErrBerkasNotFound
	}
	return b, err
}

func (s *BerkasService) videoMilik(ctx context.Context, userID, videoID uuid.UUID) (entity.Berkas, error) {
	return s.berkasDenganKode(ctx, userID, videoID, KodeVideo)
}

var tandaOLE = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// cocokDokumen memeriksa isi potongan pertama sesuai ekstensi: PDF harus
// PDF, format Office baru berupa ZIP, format Office lama berupa OLE.
func cocokDokumen(nama string, awal []byte) bool {
	jenis := http.DetectContentType(awal)
	switch strings.ToLower(filepath.Ext(nama)) {
	case ".pdf":
		return jenis == "application/pdf"
	case ".docx", ".xlsx", ".pptx":
		return jenis == "application/zip"
	case ".doc", ".xls", ".ppt":
		return bytes.HasPrefix(awal, tandaOLE)
	case ".txt", ".csv":
		return strings.HasPrefix(jenis, "text/plain")
	}
	return false
}

func (s *BerkasService) CekAwalDokumen(ctx context.Context, userID, dokumenID uuid.UUID, potongan []byte) error {
	b, err := s.berkasDenganKode(ctx, userID, dokumenID, KodeDokumen)
	if err != nil {
		return err
	}
	if !cocokDokumen(b.NamaAsli, potongan) {
		return fmt.Errorf("%w: isi berkas tidak sesuai dengan format %s", ErrBerkasTidakSah, filepath.Ext(b.NamaAsli))
	}
	return nil
}

// jenisVideo mengenali format dari isi: MP4 diawali box "ftyp" di byte
// 4-8, WebM/Matroska diawali header EBML 1A 45 DF A3. http.DetectContentType
// menolak sebagian MP4 asli (brand tanpa "mp4"), jadi diperiksa sendiri.
func jenisVideo(awal []byte) string {
	switch {
	case len(awal) >= 8 && string(awal[4:8]) == "ftyp":
		return "video/mp4"
	case bytes.HasPrefix(awal, []byte{0x1A, 0x45, 0xDF, 0xA3}):
		return "video/webm"
	}
	return ""
}

// CekAwalVideo memeriksa isi asli potongan pertama, bukan hanya tipe yang
// diakui klien. Tipe di baris berkas disesuaikan dengan isinya, karena
// perekam layar sering menyimpan Matroska dengan nama .mp4.
func (s *BerkasService) CekAwalVideo(ctx context.Context, userID, videoID uuid.UUID, potongan []byte) error {
	b, err := s.videoMilik(ctx, userID, videoID)
	if err != nil {
		return err
	}
	jenis := jenisVideo(potongan)
	if jenis == "" {
		return fmt.Errorf("%w: isi berkas bukan video MP4 atau WebM", ErrBerkasTidakSah)
	}
	if jenis != b.TipeMIME {
		return s.db.WithContext(ctx).Model(&entity.Berkas{}).Where("id = ?", b.ID).
			Update("tipe_mime", jenis).Error
	}
	return nil
}

func (s *BerkasService) TandaiVideoTerunggah(ctx context.Context, userID, videoID uuid.UUID) error {
	return s.tandaiTerunggah(ctx, userID, videoID, KodeVideo)
}

func (s *BerkasService) TandaiDokumenTerunggah(ctx context.Context, userID, dokumenID uuid.UUID) error {
	return s.tandaiTerunggah(ctx, userID, dokumenID, KodeDokumen)
}

func (s *BerkasService) tandaiTerunggah(ctx context.Context, userID, id uuid.UUID, kode string) error {
	if _, err := s.berkasDenganKode(ctx, userID, id, kode); err != nil {
		return err
	}
	sekarang := time.Now()
	return s.db.WithContext(ctx).Model(&entity.Berkas{}).Where("id = ?", id).
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
