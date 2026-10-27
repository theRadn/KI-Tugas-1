package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/entity"
	"github.com/theRadn/KI-Tugas-1/repository"

	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidUserID  = errors.New("user_id tidak valid")
	ErrInvalidTab     = errors.New("tab biodata tidak dikenal")
	ErrBiodataNotFound = errors.New("biodata belum tersedia")
)

type BiodataService struct {
	repo *repository.BiodataRepository
}

func NewBiodataService(
	repo *repository.BiodataRepository,
) *BiodataService {
	return &BiodataService{repo: repo}
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ErrInvalidUserID
	}

	return id, nil
}

func (s *BiodataService) GetTab(
	ctx context.Context,
	rawID string,
	tab string,
) (any, error) {
	id, err := parseID(rawID)
	if err != nil {
		return nil, err
	}

	b, err := s.repo.Get(ctx, id)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBiodataNotFound
	}

	if err != nil {
		return nil, err
	}

	switch tab {
	case "data-diri":
		return map[string]any{
			"id":                b.ID,
			"no_pendaftaran":    b.NoPendaftaran,
			"foto":              b.FotoPath,
			"nama_ijazah":       b.NamaIjazah,
			"nama_ktm":          b.NamaKTM,
			"ukuran_almamater":  b.UkuranAlmamater,
			"program_studi":     b.ProgramStudi,
			"jalur_masuk":       b.JalurMasuk,
			"periode_masuk":     b.PeriodeMasuk,
			"kewarganegaraan":   b.Kewarganegaraan,
			"nik":               b.NIK,
			"no_paspor":         b.NoPaspor,
			"status_pernikahan": b.StatusPernikahan,
			"jenis_kelamin":     b.JenisKelamin,
			"agama":              b.Agama,
			"tanggal_lahir":      b.TanggalLahir,
			"negara_lahir":       b.NegaraLahir,
			"provinsi_lahir":     b.ProvinsiLahir,
			"kota_lahir":         b.KotaLahir,
			"tempat_lahir":       b.TempatLahir,
			"alamat_ktp":         b.AlamatKTP,
			"negara_asal":        b.NegaraAsal,
			"provinsi_asal":      b.ProvinsiAsal,
			"kota_asal":          b.KotaAsal,
			"kode_pos":           b.KodePos,
			"alamat_domisili":    b.AlamatDomisili,
			"negara_domisili":    b.NegaraDomisili,
			"provinsi_domisili":  b.ProvinsiDomisili,
			"kota_domisili":      b.KotaDomisili,
			"email":              b.Email,
			"no_hp":              b.NoHP,
			"no_kip_k":           b.NoKIPK,
			"no_bpjs":            b.NoBPJS,
			"sumber_informasi":   b.SumberInformasi,
			"daftar_asrama":      b.DaftarAsrama,
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

	default:
		return nil, ErrInvalidTab
	}
}

func (s *BiodataService) SaveTab(
	ctx context.Context,
	rawID string,
	tab string,
	body []byte,
) error {
	id, err := parseID(rawID)
	if err != nil {
		return err
	}

	switch tab {
	case "data-diri":
		var v dto.DataDiriRequest

		if err = decode(body, &v); err != nil {
			return err
		}

		b := &entity.Biodata{
			UserID:           id,
			FotoPath:         strptr(v.Foto),
			NamaIjazah:       v.NamaIjazah,
			NamaKTM:          v.NamaKTM,
			UkuranAlmamater:  v.UkuranAlmamater,
			ProgramStudi:     v.ProgramStudi,
			JalurMasuk:       v.JalurMasuk,
			PeriodeMasuk:     v.PeriodeMasuk,
			Kewarganegaraan:  v.Kewarganegaraan,
			NIK:              v.NIK,
			NoPaspor:         strptr(v.NoPaspor),
			StatusPernikahan: v.StatusPernikahan,
			JenisKelamin:     v.JenisKelamin,
			Agama:             v.Agama,
			NegaraLahir:      v.NegaraLahir,
			ProvinsiLahir:    v.ProvinsiLahir,
			KotaLahir:        v.KotaLahir,
			TempatLahir:      v.TempatLahir,
			AlamatKTP:        v.AlamatKTP,
			NegaraAsal:       v.NegaraAsal,
			ProvinsiAsal:     v.ProvinsiAsal,
			KotaAsal:         v.KotaAsal,
			KodePos:          v.KodePos,
			AlamatDomisili:   v.AlamatDomisili,
			NegaraDomisili:   v.NegaraDomisili,
			ProvinsiDomisili: v.ProvinsiDomisili,
			KotaDomisili:     v.KotaDomisili,
			Email:            v.Email,
			NoHP:             v.NoHP,
			NoKIPK:           v.NoKIPK,
			NoBPJS:           v.NoBPJS,
			SumberInformasi:  v.SumberInformasi,
			DaftarAsrama:     v.DaftarAsrama,
		}

		if v.TanggalLahir != "" {
			t, e := time.Parse("2006-01-02", v.TanggalLahir)
			if e != nil {
				return e
			}
			b.TanggalLahir = &t
		}

		return s.repo.SaveMain(ctx, b)

	case "pendidikan":
		var v dto.PendidikanRequest

		if err = decode(body, &v); err != nil {
			return err
		}

		p := &entity.Pendidikan{
			ProvinsiSekolah: v.ProvinsiSekolah,
			KotaSekolah:     v.KotaSekolah,
			NamaSekolah:     v.NamaSekolah,
			Jurusan:         v.Jurusan,
			NISN:            v.NISN,
			TahunMasukSMTA:  v.TahunMasukSMTA,
			TahunLulusSMTA:  v.TahunLulusSMTA,
		}

		for _, n := range v.NilaiTKA {
			p.NilaiTKA = append(p.NilaiTKA, entity.NilaiTKA{
				MataPelajaran: n.MataPelajaran,
				Nilai:         n.Nilai,
			})
		}

		return s.repo.SaveRelation(ctx, id, p)

	case "keluarga":
		var v dto.KeluargaRequest

		if err = decode(body, &v); err != nil {
			return err
		}

		k := &entity.BiodataKeluarga{
			AyahNIK:                v.Ayah.NIK,
			AyahNama:               v.Ayah.Nama,
			AyahTempatLahir:        v.Ayah.TempatLahir,
			AyahPekerjaan:          v.Ayah.Pekerjaan,
			AyahInstansi:           v.Ayah.Instansi,
			AyahPendidikanTerakhir: v.Ayah.PendidikanTerakhir,
			AyahNoHP:               v.Ayah.NoHP,
			AyahStatus:             v.Ayah.Status,
			IbuNIK:                 v.Ibu.NIK,
			IbuNama:                v.Ibu.Nama,
			IbuTempatLahir:         v.Ibu.TempatLahir,
			IbuPekerjaan:           v.Ibu.Pekerjaan,
			IbuInstansi:            v.Ibu.Instansi,
			IbuPendidikanTerakhir:  v.Ibu.PendidikanTerakhir,
			IbuNoHP:                v.Ibu.NoHP,
			IbuStatus:              v.Ibu.Status,
			RangePendapatanOrtu:    v.RangePendapatanOrtu,
			NomorKK:                v.NomorKK,
			JumlahKakak:            v.JumlahKakak,
			JumlahAdik:             v.JumlahAdik,
			AlamatOrtuAlamat:       v.AlamatOrtu.Alamat,
			AlamatOrtuRT:           v.AlamatOrtu.RT,
			AlamatOrtuRW:           v.AlamatOrtu.RW,
			AlamatOrtuKelurahan:    v.AlamatOrtu.Kelurahan,
			AlamatOrtuKecamatan:    v.AlamatOrtu.Kecamatan,
			AlamatOrtuProvinsi:     v.AlamatOrtu.Provinsi,
			AlamatOrtuKota:         v.AlamatOrtu.Kota,
			AlamatOrtuKodePos:      v.AlamatOrtu.KodePos,
			AlamatOrtuNoTeleponRumah: v.AlamatOrtu.NoTeleponRumah,
		}

		if v.Ayah.TanggalLahir != "" {
			t, e := time.Parse("2006-01-02", v.Ayah.TanggalLahir)
			if e != nil {
				return e
			}
			k.AyahTanggalLahir = &t
		}

		if v.Ibu.TanggalLahir != "" {
			t, e := time.Parse("2006-01-02", v.Ibu.TanggalLahir)
			if e != nil {
				return e
			}
			k.IbuTanggalLahir = &t
		}

		return s.repo.SaveRelation(ctx, id, k)

	case "rumah":
		var v dto.RumahRequest
		if err = decode(body, &v); err != nil {
			return err
		}

		return s.repo.SaveRelation(ctx, id, &entity.Rumah{
			Kepemilikan:      v.Kepemilikan,
			TahunPerolehan:   v.TahunPerolehan,
			SumberListrik:    v.SumberListrik,
			LuasTanah:        v.LuasTanah,
			LuasBangunan:     v.LuasBangunan,
			NJOP:              v.NJOP,
			MCK:               v.MCK,
			SumberAirMinum:    v.SumberAirMinum,
			JarakPusatKotaKM: v.JarakPusatKotaKM,
			JumlahPenghuni:    v.JumlahPenghuni,
		})

	case "catatan-khusus":
		var v dto.CatatanKhususRequest
		if err = decode(body, &v); err != nil {
			return err
		}

		return s.repo.SaveRelation(ctx, id, &entity.CatatanKhusus{
			RencanaTempatTinggal: v.RencanaTempatTinggal,
			AdaDukunganKeluarga:  v.AdaDukunganKeluarga,
			TransportasiDariAsal: v.TransportasiDariAsal,
			TransportasiHarian:   v.TransportasiHarian,
			GolonganDarah:        v.GolonganDarah,
			Rhesus:               v.Rhesus,
			KondisiKesehatan:     v.KondisiKesehatan,
			Disabilitas:          v.Disabilitas,
		})

	case "pendukung":
		var v dto.PendukungRequest
		if err = decode(body, &v); err != nil {
			return err
		}

		return s.repo.SaveRelation(ctx, id, &entity.PendukungEkonomi{
			PekerjaanAyah:   v.PekerjaanAyah,
			PekerjaanIbu:    v.PekerjaanIbu,
			PenghasilanAyah: v.PenghasilanAyah,
			PenghasilanIbu:  v.PenghasilanIbu,
			TanggunganAnak:  v.TanggunganAnak,
			TagihanListrik:  v.TagihanListrik,
			DayaListrik:     v.DayaListrik,
			PajakRodaDua:    v.PajakRodaDua,
			PajakRodaEmpat:  v.PajakRodaEmpat,
			TotalPBB:        v.TotalPBB,
		})

	default:
		return ErrInvalidTab
	}
}

// decode membaca JSON lalu menjalankan aturan binding:"..." pada DTO,
// karena json.Unmarshal sendiri tidak memeriksanya.
func decode(body []byte, v any) error {
	if err := json.Unmarshal(body, v); err != nil {
		return err
	}
	return binding.Validator.ValidateStruct(v)
}

func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}