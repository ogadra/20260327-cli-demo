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
docker build --target test presenter/
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
| presenter | — | スライド同期・アンケート (AWS Lambda + API Gateway WebSocket) |

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

### Presenter WebSocket

API Gateway WebSocket 経由で front と通信する。

| 方向 | メッセージタイプ | 説明 |
|------|----------------|------|
| S→C | `slide_sync` | スライドページ同期 |
| S→C | `hands_on` | ハンズオンモード切替 |
| S→C | `viewer_count` | 接続中の視聴者数 |
| S→C | `poll_state` | アンケート状態（選択肢・投票数・自分の選択） |
| S→C | `poll_error` | アンケート操作エラー |
| C→S | `slide_sync` | スライドページ送信（presenter ロール） |
| C→S | `hands_on` | ハンズオンモード送信（presenter ロール） |
| C→S | `poll_get` | アンケート取得・初期化 |
| C→S | `poll_vote` | 投票 |
| C→S | `poll_unvote` | 投票取消 |
| C→S | `poll_switch` | 投票変更 |

## GitHub Actions シークレット

デプロイワークフローで必要な Repository secrets。OIDC によるロール引き受けを使用するため、AWS クレデンシャルの直接設定は不要。

| シークレット名 | 用途 |
|---------------|------|
| `DEPLOY_BROKER_ROLE_ARN` | broker デプロイ用 IAM ロール ARN |
| `DEPLOY_FRONT_ROLE_ARN` | front デプロイ用 IAM ロール ARN |
| `DEPLOY_NGINX_ROLE_ARN` | nginx デプロイ用 IAM ロール ARN |
| `DEPLOY_PRESENTER_ROLE_ARN` | presenter デプロイ用 IAM ロール ARN |
| `DEPLOY_RUNNER_ROLE_ARN` | runner デプロイ用 IAM ロール ARN |
| `FRONT_S3_BUCKET` | front アセット配信用 S3 バケット名 |
