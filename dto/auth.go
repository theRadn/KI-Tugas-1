package dto

type DaftarRequest struct {
	NamaPengguna        string `json:"nama_pengguna" binding:"required,min=3,max=50"`
	Email               string `json:"email" binding:"required,email"`
	KataSandi           string `json:"kata_sandi" binding:"required,min=8"`
	KonfirmasiKataSandi string `json:"konfirmasi_kata_sandi" binding:"required,eqfield=KataSandi"`
}

type MasukRequest struct {
	NamaPengguna string `json:"nama_pengguna" binding:"required"`
	KataSandi    string `json:"kata_sandi" binding:"required"`
}

type LupaKataSandiRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type AuthResponse struct {
	ID           string `json:"id"`
	NamaPengguna string `json:"nama_pengguna"`
	Email        string `json:"email"`
}

type GantiKataSandiRequest struct {
	KataSandiLama           string `json:"kata_sandi_lama" binding:"required"`
	KataSandiBaru           string `json:"kata_sandi_baru" binding:"required,min=8"`
	KonfirmasiKataSandiBaru string `json:"konfirmasi_kata_sandi_baru" binding:"required,eqfield=KataSandiBaru"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}
