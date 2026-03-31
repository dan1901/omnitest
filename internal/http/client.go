package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	gohttp "net/http"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

// Client는 부하 테스트용 HTTP 클라이언트다.
type Client struct {
	client *gohttp.Client
}

// NewClient는 최적화된 Transport 설정으로 HTTP 클라이언트를 생성한다.
func NewClient() *Client {
	transport := &gohttp.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	return &Client{
		client: &gohttp.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// Do는 HTTP 요청을 실행하고 RequestResult를 반환한다.
func (c *Client) Do(ctx context.Context, baseURL string, req model.Request, headers map[string]string) model.RequestResult {
	start := time.Now()

	url := baseURL + req.Path

	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return model.RequestResult{
				Error:     fmt.Errorf("marshal body: %w", err),
				Latency:   time.Since(start),
				Timestamp: start,
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := gohttp.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		return model.RequestResult{
			Error:     fmt.Errorf("create request: %w", err),
			Latency:   time.Since(start),
			Timestamp: start,
		}
	}

	// target-level 헤더 적용
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	// request-level 헤더 적용 (오버라이드)
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return model.RequestResult{
			Error:     err,
			Latency:   time.Since(start),
			Timestamp: start,
		}
	}

	// body를 읽어야 커넥션이 재사용됨. 내용은 불필요하므로 discard.
	n, _ := io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	latency := time.Since(start)

	result := model.RequestResult{
		StatusCode: resp.StatusCode,
		Latency:    latency,
		BytesIn:    n,
		Timestamp:  start,
	}

	if resp.StatusCode >= 400 {
		result.Error = fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return result
}
