package entity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/theRadn/KI-Tugas-1/utils"
	"gorm.io/gorm/schema"
)

// Kolom bertanda `gorm:"serializer:enkripsi"` disimpan sebagai
// "enc:v3:" + base64(JSON {"AES": ..., "DES": ..., "RC4": ...}), yaitu
// tiga versi ciphertext (AES-256-CTR, DES-CTR, RC4) dari nilai yang sama.
// Setiap versi memakai DEK dan MAC key-nya sendiri. Format v2 dan v1 tetap
// dapat dibaca untuk mendukung data lama.
//
// Format lama tetap terbaca: "enc:v1:" (hanya AES), "enc:v2:" dan teks biasa.
// Saat baris biodata disimpan ulang, nilainya otomatis menjadi format v3.
const (
	awalanV1 = "enc:v1:"
	awalanV2 = "enc:v2:"
	awalanV3 = "enc:v3:"
)

var algoritmaBiodata = []struct {
	nama    string
	alg     utils.Algorithm
	namaEnv string
	panjang int // 0 berarti bebas, asal tidak kosong
}{
	{"AES", utils.AlgorithmAES256CTR, "DATABASE_SECRET_AES256", 32},
	{"DES", utils.AlgorithmDESCTR, "DATABASE_SECRET_DES", 8},
	{"RC4", utils.AlgorithmRC4, "DATABASE_SECRET_RC4", 0},
}

type kunciKonteks struct{}

// DenganAlgoritma memilih versi ciphertext yang didekripsi saat membaca
// biodata (AES, DES, atau RC4). Tanpa ini, dipakai AES.
func DenganAlgoritma(ctx context.Context, algoritma string) context.Context {
	return context.WithValue(ctx, kunciKonteks{}, algoritma)
}

func algoritmaDari(ctx context.Context) string {
	if a, ok := ctx.Value(kunciKonteks{}).(string); ok && a != "" {
		return a
	}
	return "AES"
}

func init() {
	schema.RegisterSerializer("enkripsi", serializerEnkripsi{})
}

type serializerEnkripsi struct{}

// kunciBiodata mengembalikan kunci environment untuk data legacy.
func kunciBiodata() (map[string][]byte, []byte, error) {
	kunci := make(map[string][]byte, len(algoritmaBiodata))
	for _, a := range algoritmaBiodata {
		k := []byte(os.Getenv(a.namaEnv))
		if len(k) == 0 || (a.panjang > 0 && len(k) != a.panjang) {
			return nil, nil, fmt.Errorf("%s belum diatur dengan panjang yang benar", a.namaEnv)
		}
		kunci[a.nama] = k
	}
	return kunci, kunci["AES"], nil
}

func (serializerEnkripsi) Value(
	ctx context.Context, field *schema.Field, _ reflect.Value, nilai interface{},
) (interface{}, error) {
	if material, ok := materialUntukField(ctx, field); ok {
		return valueHierarki(material, nilai)
	}

	kunci, kunciMAC, err := kunciBiodata()
	if err != nil {
		return nil, err
	}
	polos, err := json.Marshal(nilai)
	if err != nil {
		return nil, err
	}
	versi := make(map[string]utils.EncryptedData, len(algoritmaBiodata))
	for _, a := range algoritmaBiodata {
		versi[a.nama], err = utils.Encrypt(a.alg, kunci[a.nama], kunciMAC, polos)
		if err != nil {
			return nil, err
		}
	}
	amplop, err := json.Marshal(versi)
	if err != nil {
		return nil, err
	}
	return awalanV2 + base64.StdEncoding.EncodeToString(amplop), nil
}

func (serializerEnkripsi) Scan(
	ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{},
) error {
	hasil := reflect.New(field.FieldType)

	var teks string
	switch v := dbValue.(type) {
	case nil:
	case string:
		teks = v
	case []byte:
		teks = string(v)
	default:
		// Data lama bertipe angka, bool, atau waktu dari driver.
		if b, err := json.Marshal(v); err == nil {
			teks = string(b)
		}
	}

	switch {
	case strings.HasPrefix(teks, awalanV3):
		material, ok := materialUntukField(ctx, field)
		if !ok {
			return errors.New("key hierarchy context is not configured")
		}
		polos, err := bukaV3(strings.TrimPrefix(teks, awalanV3), algoritmaDari(ctx), material)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(polos, hasil.Interface()); err != nil {
			return err
		}
	case strings.HasPrefix(teks, awalanV2):
		polos, err := bukaV2(strings.TrimPrefix(teks, awalanV2), algoritmaDari(ctx))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(polos, hasil.Interface()); err != nil {
			return err
		}
	case strings.HasPrefix(teks, awalanV1):
		polos, err := bukaV1(strings.TrimPrefix(teks, awalanV1))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(polos, hasil.Interface()); err != nil {
			return err
		}
	case teks != "":
		if field.FieldType.Kind() == reflect.String {
			hasil.Elem().SetString(teks)
		} else {
			_ = json.Unmarshal([]byte(teks), hasil.Interface())
		}
	}

	field.ReflectValueOf(ctx, dst).Set(hasil.Elem())
	return nil
}

func materialUntukField(ctx context.Context, field *schema.Field) (KeyMaterial, bool) {
	if field == nil || field.Schema == nil {
		return KeyMaterial{}, false
	}
	switch field.Schema.Table {
	case "biodata",
		"biodata_keluarga",
		"pendidikan",
		"nilai_tka",
		"rumah",
		"catatan_khusus",
		"pendukung_ekonomi":
		return KeyMaterialFor(ctx, field.Schema.Table)
	default:
		return KeyMaterial{}, false
	}
}

func valueHierarki(material KeyMaterial, nilai interface{}) (interface{}, error) {
	polos, err := json.Marshal(nilai)
	if err != nil {
		return nil, err
	}
	versi := make(map[string]utils.EncryptedData, len(algoritmaBiodata))
	for _, a := range algoritmaBiodata {
		key := material.Keys[a.nama]
		if len(key) == 0 {
			return nil, errors.New("DEK " + a.nama + " tidak tersedia")
		}
		macKey, err := utils.DeriveMACKey(key, []byte("biodata:"+material.Scope+":"+a.nama))
		if err != nil {
			return nil, err
		}
		versi[a.nama], err = utils.Encrypt(a.alg, key, macKey, polos)
		if err != nil {
			return nil, err
		}
	}
	amplop, err := json.Marshal(versi)
	if err != nil {
		return nil, err
	}
	return awalanV3 + base64.StdEncoding.EncodeToString(amplop), nil
}

func bukaV2(isi, algoritma string) ([]byte, error) {
	kunci, kunciMAC, err := kunciBiodata()
	if err != nil {
		return nil, err
	}
	amplop, err := base64.StdEncoding.DecodeString(isi)
	if err != nil {
		return nil, err
	}
	var versi map[string]utils.EncryptedData
	if err := json.Unmarshal(amplop, &versi); err != nil {
		return nil, err
	}
	data, ok := versi[algoritma]
	if !ok || kunci[algoritma] == nil {
		return nil, errors.New("versi ciphertext " + algoritma + " tidak ada")
	}
	return utils.Decrypt(data, kunci[algoritma], kunciMAC)
}

func bukaV3(isi, algoritma string, material KeyMaterial) ([]byte, error) {
	amplop, err := base64.StdEncoding.DecodeString(isi)
	if err != nil {
		return nil, err
	}
	var versi map[string]utils.EncryptedData
	if err := json.Unmarshal(amplop, &versi); err != nil {
		return nil, err
	}
	data, ok := versi[algoritma]
	key := material.Keys[algoritma]
	if !ok || len(key) == 0 {
		return nil, errors.New("versi ciphertext " + algoritma + " tidak ada")
	}
	macKey, err := utils.DeriveMACKey(key, []byte("biodata:"+material.Scope+":"+algoritma))
	if err != nil {
		return nil, err
	}
	return utils.Decrypt(data, key, macKey)
}

// bukaV1 membaca format lama yang hanya berisi AES (kunci enkripsi dan
// HMAC sama-sama kunci AES).
func bukaV1(isi string) ([]byte, error) {
	kunci, _, err := kunciBiodata()
	if err != nil {
		return nil, err
	}
	amplop, err := base64.StdEncoding.DecodeString(isi)
	if err != nil {
		return nil, err
	}
	var data utils.EncryptedData
	if err := json.Unmarshal(amplop, &data); err != nil {
		return nil, err
	}
	return utils.Decrypt(data, kunci["AES"], kunci["AES"])
}
