// Package healthcheck は runner の死活監視を提供する。
package healthcheck

import (
	"context"
	"fmt"
	"net/http"
)

// Checker は runner が到達可能かどうかをテストするインターフェース。
type Checker interface {
	// Check は指定された privateURL の runner にヘルスチェックを実行する。
	// runner が正常であれば nil を、到達不能または異常であればエラーを返す。
	Check(ctx context.Context, privateURL string) error
}

// HTTPChecker は HTTP ベースの Checker 実装。
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker は指定された http.Client を使用する HTTPChecker を生成する。
// client が nil の場合は http.DefaultClient を使用する。
func NewHTTPChecker(client *http.Client) *HTTPChecker {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPChecker{client: client}
}

// Check は GET {privateURL}/health を実行し、200 であれば nil を返す。
// 200 以外のステータスコードまたはリクエストエラーの場合はエラーを返す。
func (c *HTTPChecker) Check(ctx context.Context, privateURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, privateURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("healthcheck: create request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: unexpected status %d", resp.StatusCode)
	}
	return nil
}
