package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/dustin/go-humanize"
)

func PostMessage(ctx context.Context, channelID string, hostname string, expires time.Time) error {
	url := "https://slack.com/api/chat.postMessage"

	clientSecret := os.Getenv("SLACK_MESSAGE_KEY")

	payload := map[string]interface{}{
		"channel": channelID,
		"text":    fmt.Sprintf("%s's certificate expires in %s (%s)", hostname, humanize.Time(expires), expires.Format(time.RFC1123)),
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(b))

	return nil
}
