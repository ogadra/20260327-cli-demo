// Package main は broker サービスのエントリポイントを提供する。
package main

import (
	"fmt"
	"net/http"
	"os"
)

// newMux は broker の HTTP ルーティングを構成した ServeMux を返す。
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	return mux
}

// main は broker の HTTP サーバーを起動する。
func main() {
	mux := newMux()
	addr := ":8080"
	fmt.Fprintf(os.Stdout, "broker listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
