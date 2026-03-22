# 2026年3月27日 発表デモ

[Terminal Night #2](https://kichijojipm.connpass.com/event/382650/)の発表で用いるデモリポジトリ

## 開発

```bash
direnv allow        # Nix devShell 有効化
lefthook install    # pre-commit フック設定
```

## ビルド・テスト

```bash
docker build --target test broker/
docker build --target test runner/
docker build --target test front/
docker build --target config-test nginx/
```

## ローカル起動

```bash
# 全サービス起動
docker compose --profile broker --profile runner --profile nginx --profile front up --build
```

ブラウザで http://localhost:5173 を開く。

## サービス一覧

| サービス | ポート | 説明 |
|---------|--------|------|
| front | 5173 | Vite dev server |
| nginx | 80 | リバースプロキシ |
| broker | 8080 | 制御プレーン (DynamoDB) |
| runner | 3000 | コマンド実行サーバー |

## API

### NGINX 経由

| メソッド | パス | 説明 |
|---------|------|------|
| POST | `/api/session` | bash セッション作成。`session_id` cookie を返す |
| DELETE | `/api/session` | bash セッション削除 |
| POST | `/api/execute` | コマンド実行 (SSE ストリーム)。`{"command": "..."}` |

初回アクセス時に Broker が自動で Runner を割り当て、`runner_id` cookie を発行する。

SSE イベント種別: `stdout` (リアルタイム), `stderr` (完了時), `complete` (`exitCode` 付き)

### Broker 内部 API

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/resolve` | セッション解決 or 新規作成 |
| DELETE | `/sessions/{sessionId}` | セッション終了 |
| POST | `/internal/runners/register` | Runner 登録 |
| DELETE | `/internal/runners/{runnerId}` | Runner 登録解除 |
