# media-downloader

yt-dlp を使ったメディアダウンロード管理 Web アプリケーションです。  
チャンネルごとに保存先を分けて管理でき、ブラウザからダウンロードの登録・進捗確認ができます。

## 機能

- URL を指定してメディアをダウンロード（yt-dlp 経由）
- チャンネルを複数定義し、保存先ディレクトリを分けて管理
- ダウンロードのステータス管理（pending / downloading / converting / completed / error）
- 進捗表示
- Web UI からの操作

## アーキテクチャ

```
frontend/          React + TypeScript (Vite)
backend/           Go
  ├── api/         ogen 生成コード（OpenAPI サーバー実装）
  ├── handler/     ハンドラー実装
  ├── service/     ビジネスロジック
  ├── worker/      yt-dlp ダウンロードワーカー
  ├── db/          sqlc 生成コード + スキーマ
  └── config/      設定読み込み
spec/              TypeSpec による API 定義
```

バックエンドは Go + [ogen](https://github.com/ogen-go/ogen) で型安全な OpenAPI サーバーを実装し、SQLite でダウンロード情報を管理します。  
フロントエンドは TypeSpec → openapi-typescript で生成した型定義をもとに、型安全な API クライアントを利用します。

## 構築

### 設定

`config.yaml` を編集して起動します。

```yaml
server:
  port: 8080
  static_dir: "./frontend/dist"

database:
  path: "./data/media-downloader.db"

ytdlp:
  path: "yt-dlp"       # yt-dlp の実行パス
  audio_format: ""     # 音声変換フォーマット（例: "mp3"）、空文字で変換なし

channels:
  - secret: "abc123def456"       # URL の認証トークン（/api/{secret}/...）
    name: "Music Collection A"
    output_dir: "/data/music_a"

  - secret: "xyz789ghi012"
    name: "Music Collection B"
    output_dir: "/data/music_b"
```

チャンネルごとに `secret` を設定し、そのシークレットが API のパスおよび Web UI のアクセスキーになります。

## 開発環境

### 準備

```bash
mise install
pnpm install
```

### バックエンドの起動

```bash
docker compose up
```

コンテナに python3、ffmpeg、yt-dlp、air が含まれています。  
バックエンドは air によるホットリロードで起動します。

### フロントエンドの起動

```bash
pnpm dev
```

### 開発環境へアクセスする

http://localhost:5173/list_secret_a
http://localhost:5173/list_secret_b

## 開発

### コード生成

```bash
mise run gen
```

### テスト実行

```bash
mise run test
```
