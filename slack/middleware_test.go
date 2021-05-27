package slack

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthCheck(t *testing.T) {
	// Example data from https://api.slack.com/authentication/verifying-requests-from-slack
	slackSigningSecret := "8f742231b10e8888abcd99yyyzzz85a5"
	timestamp := "1531420618"
	payload := []byte("token=xyzz0WbapA4vBCDEFasx0q6G&team_id=T1DC2JH3J&team_domain=testteamnow&channel_id=G8PSS9T3V&channel_name=foobar&user_id=U2CERLKJA&user_name=roadrunner&command=%2Fwebhook-collect&text=&response_url=https%3A%2F%2Fhooks.slack.com%2Fcommands%2FT1DC2JH3J%2F397700885554%2F96rGlfmibIGlgcZRskXaIFfN&trigger_id=398738663015.47445629121.803a0bc887a14d10d2c447fce8b6703c")

	req, err := http.NewRequest(http.MethodPost, "/foobar", bytes.NewReader(payload))
	require.NoError(t, err)
	require.NotNil(t, req)

	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", "v0=a2114d57b48eac39b9ad189dd8316235a7b4a8d21a10bd27519666489c69b503")

	t.Run("requestHmacHash", func(t *testing.T) {
		expectedHash := "a2114d57b48eac39b9ad189dd8316235a7b4a8d21a10bd27519666489c69b503"

		assert.Equal(t, expectedHash, requestHmacHash(payload, timestamp, slackSigningSecret))
	})

	t.Run("middleware_reject", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/foobar", nil)
		w := httptest.NewRecorder()
		route := mux.NewRouter()
		route.Use(AuthCheck(slackSigningSecret))
		route.HandleFunc("/foobar", func(w http.ResponseWriter, r *http.Request) {})

		route.ServeHTTP(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("middleware_allow", func(t *testing.T) {
		w := httptest.NewRecorder()
		route := mux.NewRouter()
		route.Use(AuthCheck(slackSigningSecret))
		route.HandleFunc("/foobar", func(w http.ResponseWriter, r *http.Request) {})

		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
