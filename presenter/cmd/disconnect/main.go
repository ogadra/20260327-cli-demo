// Package main は WebSocket $disconnect ルートの Lambda ハンドラーを提供する。
package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

// fatalf はエラー時の終了処理。テスト時に差し替える。
var fatalf = log.Fatalf

// startLambda は lambda.Start のラッパー。テスト時に差し替える。
var startLambda = lambda.Start

// runFn は run のラッパー。テスト時に差し替える。
var runFn = run

// run は依存を初期化し Lambda ハンドラーを起動する。
func run() error {
	startLambda(handler)
	return nil
}

// handler は $disconnect イベントを処理するプレースホルダー。
func handler() error {
	return nil
}

// main は disconnect Lambda のエントリポイント。
func main() {
	if err := runFn(); err != nil {
		fatalf("disconnect: %v", err)
	}
}
