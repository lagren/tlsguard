package slack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

func AuthCheck(signingKey string) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			timestamp := r.Header.Get("X-Slack-Request-Timestamp")

			b, err := io.ReadAll(r.Body)
			if err != nil {
				logrus.Errorf("Could not parse request body: %s", err)

				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			requestHash := requestHmacHash(b, timestamp, signingKey)

			requestSignature := r.Header.Get("X-Slack-Signature")

			if requestSignature != fmt.Sprintf("v0=%s", requestHash) {
				// TODO Alert
				logrus.Warn("Invalid request signature")

				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			r.Body = ioutil.NopCloser(bytes.NewReader(b)) // Body already consumed, must re-initialize

			next.ServeHTTP(w, r)
		})
	}
}

func requestHmacHash(payload []byte, timestamp string, signingKey string) string {
	hs := fmt.Sprintf("v0:%s:%s", timestamp, string(payload))

	hash := hmac.New(sha256.New, []byte(signingKey))
	hash.Write([]byte(hs))
	s := hash.Sum(nil)

	return hex.EncodeToString(s)
}
