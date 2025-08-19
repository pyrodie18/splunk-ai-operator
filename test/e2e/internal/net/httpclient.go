package net

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

func PostJSONLocal(localPort, path string, body []byte, headers map[string]string) (int, []byte, error) {
	url := fmt.Sprintf("http://127.0.0.1:%s%s", localPort, path)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}
