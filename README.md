# 2026年3月27日 発表デモ

[Terminal Night #2](https://kichijojipm.connpass.com/event/382650/)の発表で用いるデモリポジトリ

## 開発

```bash
direnv allow        # Nix devShell 有効化
lefthook install    # pre-commit フック設定
```

## Runnerコンテナのビルド・テスト・実行

```bash
docker build --target test runner/   # テスト
docker build -t runner runner/       # ビルド
docker run --rm -p 3000:3000 runner  # 起動
```
