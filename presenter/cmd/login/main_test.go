package main

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"golang.org/x/crypto/bcrypt"
)

// errorReader は常にエラーを返す io.Reader。
type errorReader struct{}

// Read は常にエラーを返す。
func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, fmt.Errorf("read error")
}

// mockSecretGetter は secretGetterAPI のモック実装。
type mockSecretGetter struct {
	getSecretValueFn func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// GetSecretValue はモックの GetSecretValue を呼び出す。
func (m *mockSecretGetter) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return m.getSecretValueFn(ctx, params, optFns...)
}

// mockSessionCreator は sessionCreator のモック実装。
type mockSessionCreator struct {
	createFn func(ctx context.Context, token string) error
}

// Create はモックの Create を呼び出す。
func (m *mockSessionCreator) Create(ctx context.Context, token string) error {
	return m.createFn(ctx, token)
}

// mockSessionValidator は sessionValidatorAPI のモック実装。
type mockSessionValidator struct {
	isValidFn func(ctx context.Context, token string) (bool, error)
}

// IsValid はモックの IsValid を呼び出す。
func (m *mockSessionValidator) IsValid(ctx context.Context, token string) (bool, error) {
	return m.isValidFn(ctx, token)
}

// newHTTPRequest はテスト用の APIGatewayV2HTTPRequest を生成する。
func newHTTPRequest(method, body string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method,
			},
		},
		Body: body,
	}
}

// newHTTPRequestWithCookie はテスト用の Cookie 付き APIGatewayV2HTTPRequest を生成する。
func newHTTPRequestWithCookie(method, cookie string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method,
			},
		},
		Headers: map[string]string{
			"cookie": cookie,
		},
	}
}

// setupGetDeps は GET テスト用の依存を設定する。クリーンアップ関数を返す。
func setupGetDeps(t *testing.T) func() {
	t.Helper()
	origSessValidator := sessValidator
	origJSONMarshal := jsonMarshal
	return func() {
		sessValidator = origSessValidator
		jsonMarshal = origJSONMarshal
	}
}

// validBcryptHash はテスト用の bcrypt ハッシュ。パスワード "correct" に対応する。
// $2a$10$ のコストで生成されたもの。
const validBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// setupPostDeps は POST テスト用の依存を設定する。クリーンアップ関数を返す。
func setupPostDeps(t *testing.T) func() {
	t.Helper()
	origSecretGetter := secretGetter
	origSessCreator := sessCreator
	origSecretARN := secretARN
	origTokenFn := tokenFn
	origCompareHashFn := compareHashFn
	origJSONMarshal := jsonMarshal
	return func() {
		secretGetter = origSecretGetter
		sessCreator = origSessCreator
		secretARN = origSecretARN
		tokenFn = origTokenFn
		compareHashFn = origCompareHashFn
		jsonMarshal = origJSONMarshal
	}
}

// TestHandler_GET_Authenticated は有効な cookie で GET リクエストが 200 を返すことを検証する。
func TestHandler_GET_Authenticated(t *testing.T) {
	cleanup := setupGetDeps(t)
	defer cleanup()

	sessValidator = &mockSessionValidator{
		isValidFn: func(_ context.Context, token string) (bool, error) {
			if token != "validtoken" {
				t.Errorf("unexpected token: %s", token)
			}
			return true, nil
		},
	}

	req := newHTTPRequestWithCookie("GET", "slide_auth=validtoken")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"status":"authenticated"}` {
		t.Errorf("unexpected body: %s", resp.Body)
	}
}

// TestHandler_GET_AuthenticatedViaCookiesField は API Gateway v2 の Cookies フィールド経由で認証が通ることを検証する。
func TestHandler_GET_AuthenticatedViaCookiesField(t *testing.T) {
	cleanup := setupGetDeps(t)
	defer cleanup()

	sessValidator = &mockSessionValidator{
		isValidFn: func(_ context.Context, token string) (bool, error) {
			if token != "v2token" {
				t.Errorf("unexpected token: %s", token)
			}
			return true, nil
		},
	}

	req := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
			},
		},
		Cookies: []string{"slide_auth=v2token"},
	}
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestHandler_GET_NoCookie は cookie なしで GET リクエストが 401 を返すことを検証する。
func TestHandler_GET_NoCookie(t *testing.T) {
	cleanup := setupGetDeps(t)
	defer cleanup()

	req := newHTTPRequest("GET", "")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// TestHandler_GET_InvalidSession は無効なセッションで GET リクエストが 401 を返すことを検証する。
func TestHandler_GET_InvalidSession(t *testing.T) {
	cleanup := setupGetDeps(t)
	defer cleanup()

	sessValidator = &mockSessionValidator{
		isValidFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}

	req := newHTTPRequestWithCookie("GET", "slide_auth=expired")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// TestHandler_GET_ValidatorError はセッション検証エラーで GET リクエストが 500 を返すことを検証する。
func TestHandler_GET_ValidatorError(t *testing.T) {
	cleanup := setupGetDeps(t)
	defer cleanup()

	sessValidator = &mockSessionValidator{
		isValidFn: func(_ context.Context, _ string) (bool, error) {
			return false, fmt.Errorf("dynamo error")
		},
	}

	req := newHTTPRequestWithCookie("GET", "slide_auth=sometoken")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandler_GET_MarshalError は GET リクエストで jsonMarshal がエラーを返す場合を検証する。
func TestHandler_GET_MarshalError(t *testing.T) {
	cleanup := setupGetDeps(t)
	defer cleanup()

	sessValidator = &mockSessionValidator{
		isValidFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, fmt.Errorf("marshal error")
	}

	req := newHTTPRequestWithCookie("GET", "slide_auth=validtoken")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandler_InvalidMethod は許可されていない HTTP メソッドで 405 を返すことを検証する。
func TestHandler_InvalidMethod(t *testing.T) {
	t.Parallel()
	req := newHTTPRequest("DELETE", "")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 405 {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

// TestHandler_POST_Success は POST リクエストで正しいパスワードを送信した場合のセッション作成と 302 リダイレクトを検証する。
func TestHandler_POST_Success(t *testing.T) {
	cleanup := setupPostDeps(t)
	defer cleanup()

	secretGetter = &mockSecretGetter{
		getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(validBcryptHash),
			}, nil
		},
	}
	var createdToken string
	sessCreator = &mockSessionCreator{
		createFn: func(_ context.Context, token string) error {
			createdToken = token
			return nil
		},
	}
	secretARN = "arn:aws:secretsmanager:us-east-1:123456789:secret:test"
	tokenFn = func() (string, error) {
		return "aabbccdd", nil
	}
	compareHashFn = func(_, _ []byte) error {
		return nil
	}

	req := newHTTPRequest("POST", `{"password":"correct"}`)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}
	if createdToken != "aabbccdd" {
		t.Errorf("token = %q, want %q", createdToken, "aabbccdd")
	}
	if len(resp.Cookies) != 1 || resp.Cookies[0] != "slide_auth=aabbccdd; HttpOnly; Secure; SameSite=Strict; Path=/" {
		t.Errorf("unexpected cookies: %v", resp.Cookies)
	}
	location := resp.Headers["Location"]
	if location != "/" {
		t.Errorf("expected Location /, got %s", location)
	}
}

// TestHandler_POST_BadBody は不正な JSON ボディで 400 を返すことを検証する。
func TestHandler_POST_BadBody(t *testing.T) {
	cleanup := setupPostDeps(t)
	defer cleanup()

	req := newHTTPRequest("POST", `invalid json`)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// TestHandler_POST_SecretError は Secrets Manager のエラーで 500 を返すことを検証する。
func TestHandler_POST_SecretError(t *testing.T) {
	cleanup := setupPostDeps(t)
	defer cleanup()

	secretGetter = &mockSecretGetter{
		getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return nil, fmt.Errorf("secret error")
		},
	}

	req := newHTTPRequest("POST", `{"password":"test"}`)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandler_POST_WrongPassword は間違ったパスワードで 401 を返すことを検証する。
func TestHandler_POST_WrongPassword(t *testing.T) {
	cleanup := setupPostDeps(t)
	defer cleanup()

	secretGetter = &mockSecretGetter{
		getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(validBcryptHash),
			}, nil
		},
	}
	compareHashFn = func(_, _ []byte) error {
		return bcrypt.ErrMismatchedHashAndPassword
	}

	req := newHTTPRequest("POST", `{"password":"wrong"}`)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// TestHandler_POST_BcryptInternalError は bcrypt のフォーマットエラーなど ErrMismatchedHashAndPassword 以外のエラーで 500 を返すことを検証する。
func TestHandler_POST_BcryptInternalError(t *testing.T) {
	cleanup := setupPostDeps(t)
	defer cleanup()

	secretGetter = &mockSecretGetter{
		getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(validBcryptHash),
			}, nil
		},
	}
	compareHashFn = func(_, _ []byte) error {
		return fmt.Errorf("hashedSecret too short to be a bcrypted password")
	}

	req := newHTTPRequest("POST", `{"password":"test"}`)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandler_POST_NilSecretString は SecretString が nil の場合に 500 を返すことを検証する。
func TestHandler_POST_NilSecretString(t *testing.T) {
	cleanup := setupPostDeps(t)
	defer cleanup()

	secretGetter = &mockSecretGetter{
		getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{
				SecretString: nil,
			}, nil
		},
	}

	req := newHTTPRequest("POST", `{"password":"test"}`)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandler_POST_TokenError はトークン生成エラーで 500 を返すことを検証する。
func TestHandler_POST_TokenError(t *testing.T) {
	cleanup := setupPostDeps(t)
	defer cleanup()

	secretGetter = &mockSecretGetter{
		getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(validBcryptHash),
			}, nil
		},
	}
	compareHashFn = func(_, _ []byte) error {
		return nil
	}
	tokenFn = func() (string, error) {
		return "", fmt.Errorf("token error")
	}

	req := newHTTPRequest("POST", `{"password":"correct"}`)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandler_POST_SessionCreateError はセッション作成エラーで 500 を返すことを検証する。
func TestHandler_POST_SessionCreateError(t *testing.T) {
	cleanup := setupPostDeps(t)
	defer cleanup()

	secretGetter = &mockSecretGetter{
		getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(validBcryptHash),
			}, nil
		},
	}
	compareHashFn = func(_, _ []byte) error {
		return nil
	}
	tokenFn = func() (string, error) {
		return "token123", nil
	}
	sessCreator = &mockSessionCreator{
		createFn: func(_ context.Context, _ string) error {
			return fmt.Errorf("session error")
		},
	}

	req := newHTTPRequest("POST", `{"password":"correct"}`)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestGenerateToken は generateToken が 64 文字の 16 進文字列を返すことを検証する。
func TestGenerateToken(t *testing.T) {
	t.Parallel()
	token, err := generateToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token) != 64 {
		t.Errorf("token length = %d, want 64", len(token))
	}
}

// TestGenerateToken_ReadError は randReader がエラーを返す場合を検証する。
func TestGenerateToken_ReadError(t *testing.T) {
	origReader := randReader
	defer func() { randReader = origReader }()
	randReader = io.Reader(&errorReader{})

	_, err := generateToken()
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRun_Success は run 関数の正常処理を検証する。
func TestRun_Success(t *testing.T) {
	origLoadConfig := loadConfig
	origStartLambda := startLambda
	defer func() {
		loadConfig = origLoadConfig
		startLambda = origStartLambda
	}()

	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, nil
	}
	startLambda = func(_ interface{}) {}

	t.Setenv("SESSIONS_TABLE", "sessions-table")
	t.Setenv("SECRET_ARN", "arn:aws:secretsmanager:us-east-1:123456789:secret:test")

	if err := run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRun_ConfigError は AWS 設定読み込みエラーを検証する。
func TestRun_ConfigError(t *testing.T) {
	origLoadConfig := loadConfig
	defer func() { loadConfig = origLoadConfig }()

	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, fmt.Errorf("config error")
	}

	if err := run(); err == nil {
		t.Fatal("expected error")
	}
}

// TestRun_MissingSessionsTable は SESSIONS_TABLE 未設定時のエラーを検証する。
func TestRun_MissingSessionsTable(t *testing.T) {
	origLoadConfig := loadConfig
	origStartLambda := startLambda
	defer func() {
		loadConfig = origLoadConfig
		startLambda = origStartLambda
	}()

	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, nil
	}
	startLambda = func(_ interface{}) {}

	t.Setenv("SESSIONS_TABLE", "")

	if err := run(); err == nil {
		t.Fatal("expected error")
	}
}

// TestRun_MissingSecretARN は SECRET_ARN 未設定時のエラーを検証する。
func TestRun_MissingSecretARN(t *testing.T) {
	origLoadConfig := loadConfig
	origStartLambda := startLambda
	defer func() {
		loadConfig = origLoadConfig
		startLambda = origStartLambda
	}()

	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, nil
	}
	startLambda = func(_ interface{}) {}

	t.Setenv("SESSIONS_TABLE", "sessions-table")
	t.Setenv("SECRET_ARN", "")

	if err := run(); err == nil {
		t.Fatal("expected error")
	}
}

// TestMain_Success は main 関数の正常処理を検証する。
func TestMain_Success(t *testing.T) {
	origRunFn := runFn
	defer func() { runFn = origRunFn }()

	runFn = func() error { return nil }
	main()
}

// TestMain_Error は main 関数のエラー処理を検証する。
func TestMain_Error(t *testing.T) {
	origRunFn := runFn
	origFatalf := fatalf
	defer func() {
		runFn = origRunFn
		fatalf = origFatalf
	}()

	runFn = func() error { return fmt.Errorf("run error") }
	var called bool
	fatalf = func(_ string, _ ...interface{}) { called = true }
	main()
	if !called {
		t.Error("expected fatalf to be called")
	}
}
