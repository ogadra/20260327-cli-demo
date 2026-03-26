// Package main は /login エンドポイントの Lambda ハンドラーを提供する。
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"golang.org/x/crypto/bcrypt"

	"github.com/ogadra/20260327-cli-demo/presenter/internal/session"
)

// fatalf はエラー時の終了処理。テスト時に差し替える。
var fatalf = log.Fatalf

// startLambda は lambda.Start のラッパー。テスト時に差し替える。
var startLambda = lambda.Start

// runFn は run のラッパー。テスト時に差し替える。
var runFn = run

// loadConfig は AWS 設定読み込みのラッパー。テスト時に差し替える。
var loadConfig = config.LoadDefaultConfig

// jsonMarshal は JSON エンコードのラッパー。テスト時に差し替える。
var jsonMarshal = json.Marshal

// secretGetterAPI は Secrets Manager からシークレットを取得するインターフェース。
type secretGetterAPI interface {
	// GetSecretValue はシークレットの値を取得する。
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// sessionCreator はセッションを作成するインターフェース。
type sessionCreator interface {
	// Create はセッションを作成する。
	Create(ctx context.Context, token string) error
}

// secretGetter は Secrets Manager クライアント。テスト時に差し替える。
var secretGetter secretGetterAPI

// sessCreator はセッション作成。テスト時に差し替える。
var sessCreator sessionCreator

// secretARN はシークレットの ARN。テスト時に差し替える。
var secretARN string

// tokenFn はランダムトークンを生成する関数。テスト時に差し替える。
var tokenFn = generateToken

// randReader はランダムバイト読み取り元。テスト時に差し替える。
var randReader io.Reader = rand.Reader

// compareHashFn は bcrypt ハッシュとパスワードを比較する関数。テスト時に差し替える。
var compareHashFn = bcrypt.CompareHashAndPassword

// loginRequest はログインリクエストボディ。
type loginRequest struct {
	Password string `json:"password"`
}

// generateToken は 32 バイトのランダムトークンを生成し、16 進文字列として返す。
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(randReader, b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// handler は /login リクエストを処理する。
func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	switch req.RequestContext.HTTP.Method {
	case "GET":
		return handleGet()
	case "POST":
		return handlePost(ctx, req)
	default:
		return events.APIGatewayV2HTTPResponse{StatusCode: 405, Body: `{"error":"method not allowed"}`}, nil
	}
}

// handleGet は GET リクエストを処理する。
func handleGet() (events.APIGatewayV2HTTPResponse, error) {
	body, err := jsonMarshal(map[string]string{"message": "presenter login"})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: 500, Body: `{"error":"internal server error"}`}, nil
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: 200, Body: string(body)}, nil
}

// handlePost は POST リクエストを処理する。パスワード検証とセッション作成を行う。
func handlePost(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var loginReq loginRequest
	if err := json.Unmarshal([]byte(req.Body), &loginReq); err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: 400, Body: `{"error":"invalid request body"}`}, nil
	}

	secretOut, err := secretGetter.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretARN,
	})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: 500, Body: `{"error":"internal server error"}`}, nil
	}

	hash := *secretOut.SecretString
	if err := compareHashFn([]byte(hash), []byte(loginReq.Password)); err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: 401, Body: `{"error":"unauthorized"}`}, nil
	}

	token, err := tokenFn()
	if err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: 500, Body: `{"error":"internal server error"}`}, nil
	}

	if err := sessCreator.Create(ctx, token); err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: 500, Body: `{"error":"internal server error"}`}, nil
	}

	cookie := fmt.Sprintf("slide_auth=%s; HttpOnly; Secure; SameSite=Strict; Path=/", token)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Set-Cookie": cookie,
			"Location":   "/",
		},
	}, nil
}

// run は依存を初期化し Lambda ハンドラーを起動する。
func run() error {
	ctx := context.Background()
	cfg, err := loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	sessionsTable := os.Getenv("SESSIONS_TABLE")
	if sessionsTable == "" {
		return fmt.Errorf("SESSIONS_TABLE environment variable is required")
	}

	arn := os.Getenv("SECRET_ARN")
	if arn == "" {
		return fmt.Errorf("SECRET_ARN environment variable is required")
	}
	secretARN = arn

	ddbClient := dynamodb.NewFromConfig(cfg)
	sessCreator = session.NewStore(ddbClient, sessionsTable)
	secretGetter = secretsmanager.NewFromConfig(cfg)

	startLambda(handler)
	return nil
}

// main は login Lambda のエントリポイント。
func main() {
	if err := runFn(); err != nil {
		fatalf("login: %v", err)
	}
}
