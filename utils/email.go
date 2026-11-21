package utils

import (
	"errors"
	"net/smtp"
	"os"
	"strings"
)

// KirimEmail mengirim email teks biasa lewat SMTP Gmail.
// GMAIL_USER adalah alamat Gmail pengirim, GMAIL_APP_PASSWORD adalah
// App Password Google (bukan kata sandi akun biasa).
func KirimEmail(ke, subjek, isi string) error {
	pengirim := os.Getenv("GMAIL_USER")
	sandi := strings.ReplaceAll(os.Getenv("GMAIL_APP_PASSWORD"), " ", "")
	if pengirim == "" || sandi == "" {
		return errors.New("GMAIL_USER atau GMAIL_APP_PASSWORD belum diatur")
	}

	pesan := "From: " + pengirim + "\r\n" +
		"To: " + ke + "\r\n" +
		"Subject: " + subjek + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" + isi

	auth := smtp.PlainAuth("", pengirim, sandi, "smtp.gmail.com")
	return smtp.SendMail("smtp.gmail.com:587", auth, pengirim, []string{ke}, []byte(pesan))
}
