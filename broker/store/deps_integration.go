//go:build integration

// Package store はインテグレーションテスト用の依存を保持する。
package store

import (
	_ "github.com/aws/aws-sdk-go-v2/config"
	_ "github.com/aws/aws-sdk-go-v2/credentials"
)
