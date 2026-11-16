package entity

type User struct {
	BaseModel

	NamaPengguna  string `gorm:"type:varchar(100);not null;uniqueIndex" json:"nama_pengguna"`
	Email         string `gorm:"type:varchar(255);not null;index" json:"email"`
	KataSandiHash string `gorm:"type:text;not null" json:"-"`
	KEKWrapped    []byte `json:"-" gorm:"type:blob;not null"`
}

func (User) TableName() string {
	return "users"
}
