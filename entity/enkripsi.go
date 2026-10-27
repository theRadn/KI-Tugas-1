package entity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"

	"github.com/theRadn/KI-Tugas-1/utils"
	"gorm.io/gorm/schema"
)

// Kolom bertanda `gorm:"serializer:enkripsi"` disimpan sebagai
// "enc:v1:" + base64(JSON utils.EncryptedData), memakai AES-256-CTR dan
// HMAC-SHA256 dengan kunci DATABASE_SECRET_AES256. Nilai yang belum
// berawalan itu dianggap data lama (teks biasa) dan tetap terbaca.
const awalanEnkripsi = "enc:v1:"

var errKunciBiodata = errors.New("DATABASE_SECRET_AES256 harus 32 karakter")

func init() {
	schema.RegisterSerializer("enkripsi", serializerEnkripsi{})
}

type serializerEnkripsi struct{}

func kunciBiodata() ([]byte, error) {
	kunci := []byte(os.Getenv("DATABASE_SECRET_AES256"))
	if len(kunci) != 32 {
		return nil, errKunciBiodata
	}
	return kunci, nil
}

func (serializerEnkripsi) Value(
	_ context.Context, _ *schema.Field, _ reflect.Value, nilai interface{},
) (interface{}, error) {
	kunci, err := kunciBiodata()
	if err != nil {
		return nil, err
	}
	polos, err := json.Marshal(nilai)
	if err != nil {
		return nil, err
	}
	data, err := utils.Encrypt(utils.AlgorithmAES256CTR, kunci, kunci, polos)
	if err != nil {
		return nil, err
	}
	amplop, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return awalanEnkripsi + base64.StdEncoding.EncodeToString(amplop), nil
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

	if isi, ok := strings.CutPrefix(teks, awalanEnkripsi); ok {
		kunci, err := kunciBiodata()
		if err != nil {
			return err
		}
		amplop, err := base64.StdEncoding.DecodeString(isi)
		if err != nil {
			return err
		}
		var data utils.EncryptedData
		if err := json.Unmarshal(amplop, &data); err != nil {
			return err
		}
		polos, err := utils.Decrypt(data, kunci, kunci)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(polos, hasil.Interface()); err != nil {
			return err
		}
	} else if teks != "" {
		if field.FieldType.Kind() == reflect.String {
			hasil.Elem().SetString(teks)
		} else {
			_ = json.Unmarshal([]byte(teks), hasil.Interface())
		}
	}

	field.ReflectValueOf(ctx, dst).Set(hasil.Elem())
	return nil
}
