package handler

import (
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// Fixture for eval #3: INFO log prints full response body (sensitive/noisy).
func Proxy(w http.ResponseWriter, r *http.Request) {
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		log.WithError(err).Error("proxy request failed")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	log.Infof("upstream response status=%d body=%s", resp.StatusCode, string(body))
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}
