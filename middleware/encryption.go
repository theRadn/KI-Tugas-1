package middleware

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/theRadn/KI-Tugas-1/utils"
)

const (
	sessionEnvelopeVersion = "v3"
	maxEncryptedBodySize   = 1 << 20 // 1 MiB

	// Browser tidak bisa mengirim body pada GET, jadi amplop untuk GET
	// dikirim lewat header ini sebagai JSON yang di-base64.
	envelopeHeader = "X-Encrypted-Envelope"
)

type EncryptedEnvelope = utils.EncryptedEnvelope

type bufferedResponseWriter struct {
	gin.ResponseWriter
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newBufferedResponseWriter(
	writer gin.ResponseWriter,
) *bufferedResponseWriter {
	return &bufferedResponseWriter{
		ResponseWriter: writer,
		status:         http.StatusOK,
	}
}

func (w *bufferedResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}

	w.status = code
	w.wroteHeader = true
}

func (w *bufferedResponseWriter) WriteHeaderNow() {
	if !w.wroteHeader {
		w.wroteHeader = true
	}
}

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.body.Write(data)
}

func (w *bufferedResponseWriter) WriteString(data string) (int, error) {
	return w.Write([]byte(data))
}

func (w *bufferedResponseWriter) Status() int {
	return w.status
}

func (w *bufferedResponseWriter) Size() int {
	return w.body.Len()
}

func (w *bufferedResponseWriter) Written() bool {
	return w.wroteHeader || w.body.Len() > 0
}

func Encryption(sessionManager *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		if sessionManager == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "encryption session manager unavailable",
			})
			return
		}

		var body []byte
		if c.Request.Method == http.MethodGet {
			decoded, err := base64.StdEncoding.DecodeString(
				c.GetHeader(envelopeHeader),
			)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "invalid encrypted envelope header",
				})
				return
			}
			body = decoded
		} else {
			if c.Request.Body == nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "missing encrypted request body",
				})
				return
			}

			read, err := io.ReadAll(
				io.LimitReader(c.Request.Body, maxEncryptedBodySize+1),
			)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "failed to read request body",
				})
				return
			}
			body = read
		}

		if len(body) == 0 || len(body) > maxEncryptedBodySize {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid encrypted request body size",
			})
			return
		}

		var envelope EncryptedEnvelope
		if err := json.Unmarshal(body, &envelope); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid encrypted envelope",
			})
			return
		}

		if envelope.Version != sessionEnvelopeVersion ||
			envelope.Direction != "request" ||
			envelope.SessionID == "" ||
			envelope.RequestID == "" ||
			len(envelope.RequestID) > 128 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid encrypted envelope fields",
			})
			return
		}

		session, ok := sessionManager.GetSession(envelope.SessionID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired session",
			})
			return
		}

		keys := utils.SessionKeys{
			AESKey: session.AESKey,
			MACKey: session.MACKey,
		}

		plaintext, err := utils.DecryptSessionPayload(
			keys,
			envelope.SessionID,
			"request",
			envelope,
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "encrypted request authentication failed",
			})
			return
		}

		// Only consume the request ID after HMAC authentication succeeds.
		if !sessionManager.ConsumeRequestID(
			envelope.SessionID,
			envelope.RequestID,
		) {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": "duplicate or invalid request ID",
			})
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(plaintext))
		c.Request.ContentLength = int64(len(plaintext))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Header.Del("Content-Length")

		originalWriter := c.Writer
		bufferedWriter := newBufferedResponseWriter(originalWriter)
		c.Writer = bufferedWriter

		c.Next()

		// Restore the original writer before writing the final response.
		c.Writer = originalWriter

		encryptedResponse, err := utils.EncryptSessionPayload(
			keys,
			envelope.SessionID,
			envelope.RequestID,
			"response",
			bufferedWriter.body.Bytes(),
		)
		if err != nil {
			originalWriter.Header().Set("Content-Type", "application/json")
			originalWriter.WriteHeader(http.StatusInternalServerError)
			_, _ = originalWriter.Write([]byte(
				`{"error":"failed to encrypt response"}`,
			))
			return
		}

		originalWriter.Header().Set("Content-Type", "application/json")
		originalWriter.Header().Del("Content-Length")
		originalWriter.Header().Del("Content-Encoding")
		originalWriter.Header().Del("Content-Length")

		// Prevent headers intended for plaintext JSON from leaking into the encrypted response.
		originalWriter.Header().Del("ETag")
		originalWriter.Header().Del("Content-Range")

		originalWriter.WriteHeader(bufferedWriter.status)

		if err := json.NewEncoder(originalWriter).Encode(encryptedResponse); err != nil {
			// The response has already started, so there is no safe
			// alternate response to send here.
			return
		}
	}
}
