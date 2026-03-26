# Media Downloader 仕様書

## 1. 概要

YouTube動画のURLをWeb UIから入力し、yt-dlpを使用して音声ファイルとしてダウンロード・保存するWebアプリケーション。

シークレットリンク（推測困難なURLパス）ごとに保存先ディレクトリが分かれており、各シークレットリンクが独立したページとして機能する。

## 2. システム構成

```
┌─────────────┐       ┌─────────────────────────┐
│  React SPA  │◄─────►│  Go HTTP Server (API)   │
│  (Frontend) │       │                         │
└─────────────┘       │  ┌───────────────────┐  │
                      │  │ Download Worker    │  │
                      │  │ (goroutine pool)   │  │
                      │  └───────┬───────────┘  │
                      │          │              │
                      │  ┌───────▼───────────┐  │
                      │  │ yt-dlp (CLI)       │  │
                      │  └───────┬───────────┘  │
                      │          │              │
                      └──────────┼──────────────┘
                                 │
                      ┌──────────▼──────────┐
                      │  SQLite (履歴DB)     │
                      │  保存先ディレクトリ    │
                      └─────────────────────┘
```

### 技術スタック

| レイヤー | 技術 |
|---------|------|
| Frontend | React (TypeScript) + openapi-fetch |
| API定義 | TypeSpec → OpenAPI 3.1 |
| フロントエンド型生成 | openapi-typescript (OpenAPI → TypeScript型) |
| APIコード生成 | ogen (OpenAPI → Go サーバー/型) |
| Backend | Go (ogen生成サーバー) |
| ダウンロード | yt-dlp (CLIコマンド実行) |
| DB | SQLite |
| DBコード生成 | sqlc (SQL → Go 型安全クエリ) |
| API E2Eテスト | runn (YAMLシナリオテスト) |
| ユニットテスト | Go標準 testing パッケージ |
| ツール管理 | mise (パッケージ・タスク管理) |
| コンテナ | Docker (multi-stage build) |

## 3. 機能要件

### 3.1 シークレットリンクページ

- URLパス `/{secret}` でアクセスすると、そのシークレットリンクに対応するページを表示する
- 存在しないシークレットリンクの場合は404を返す
- ページには以下を含む:
  - YouTube URL入力フォーム
  - ダウンロード履歴・進行状況一覧

### 3.2 ダウンロード

- YouTube URLを入力して送信すると、バックグラウンドでダウンロードを開始する
- yt-dlpを `--audio-quality 0` (bestaudio) で実行し、音声ファイルを取得する
- ダウンロード完了後、設定ファイルで指定されたディレクトリに保存する
- ブラウザセッションが切れてもダウンロードは継続する

### 3.3 進行状況の表示

- ダウンロード中のタスクの進行状況（パーセンテージ）をリアルタイムで表示する
- 進行状況の取得はポーリング（2秒間隔）で実装する
- 表示項目:
  - 動画タイトル
  - ステータス（待機中 / ダウンロード中 / 変換中 / 完了 / エラー）
  - 進行率（%）
  - エラー時のメッセージ

### 3.4 ダウンロード履歴

- 過去のダウンロード履歴をシークレットリンクごとに一覧表示する
- SQLiteに永続化し、コンテナ再起動後も保持する

## 4. 非機能要件

| 項目 | 内容 |
|------|------|
| バックグラウンド実行 | goroutineで非同期処理。ブラウザ切断に依存しない |
| 認証 | シークレットリンク（推測困難なURLパス）のみ。ログイン機能なし |
| 同時ダウンロード | 制限なし |
| 設定 | YAML設定ファイルで管理 |
| デプロイ | Dockerイメージとしてビルド・実行 |

## 5. API設計

### 5.1 TypeSpec → OpenAPI → ogen ワークフロー

```
TypeSpec定義 (.tsp)
       │
       ▼  tsp compile
OpenAPI 3.1 仕様 (.yaml)
       │
       ▼  ogen generate
Go サーバーコード (oas_*_gen.go)
       │
       ▼  Handler interface を実装
ビジネスロジック
```

#### ビルド手順

miseタスクで実行する（詳細はセクション11参照）:

```bash
# 全コード生成を実行
mise run gen

# 個別実行
mise run gen:tsp        # TypeSpec → OpenAPI
mise run gen:go:ogen    # OpenAPI → Go (ogen)
mise run gen:go:sqlc    # SQL → Go (sqlc)
mise run gen:ts         # OpenAPI → TypeScript型 (openapi-typescript)
```

### 5.2 TypeSpec 定義

ファイル: `spec/main.tsp`

```typespec
import "@typespec/http";
import "@typespec/openapi3";

using TypeSpec.Http;

@service({
  title: "Media Downloader API",
})
namespace MediaDownloader;

/** ダウンロードステータス */
enum DownloadStatus {
  pending,
  downloading,
  converting,
  completed,
  error,
}

/** ダウンロード情報 */
model Download {
  id: string;
  url: string;
  title?: string;
  status: DownloadStatus;
  progress: int32;
  filename?: string;
  error?: string;
  createdAt: utcDateTime;
  completedAt?: utcDateTime;
}

/** ダウンロード開始リクエスト */
model CreateDownloadRequest {
  url: string;
}

/** ダウンロード一覧レスポンス */
model DownloadListResponse {
  downloads: Download[];
}

/** チャンネル情報レスポンス */
model ChannelInfoResponse {
  name: string;
}

/** エラーレスポンス */
@error
model ErrorResponse {
  message: string;
}

@route("/api/{secret}")
namespace Channel {
  /** チャンネル情報を取得 */
  @get
  op getChannelInfo(@path secret: string): ChannelInfoResponse | ErrorResponse;

  /** ダウンロード一覧を取得 */
  @route("/downloads")
  @get
  op listDownloads(@path secret: string): DownloadListResponse | ErrorResponse;

  /** ダウンロードを開始 */
  @route("/downloads")
  @post
  op createDownload(
    @path secret: string,
    @body body: CreateDownloadRequest,
  ): {
    @statusCode statusCode: 202;
    @body body: Download;
  } | ErrorResponse;
}
```

ファイル: `spec/tspconfig.yaml`

```yaml
emit:
  - "@typespec/openapi3"
options:
  "@typespec/openapi3":
    output-file: openapi.yaml
    emitter-output-dir: "{output-dir}/tsp-output"
```

### 5.3 ogen コード生成

`mise run gen:go:ogen` で実行（`.mise.toml` で定義）。

ogenが生成するファイル (`backend/api/`):

| ファイル | 内容 |
|---------|------|
| `oas_interfaces_gen.go` | **Handler interface** (実装する契約) |
| `oas_server_gen.go` | HTTPサーバー実装・ルーティング |
| `oas_schemas_gen.go` | リクエスト/レスポンスのGo型 |
| `oas_request_decoders_gen.go` | リクエストのデコード・バリデーション |
| `oas_response_encoders_gen.go` | レスポンスのエンコード |
| `oas_router_gen.go` | 静的ラディックスルーター |
| `oas_json_gen.go` | JSON エンコード/デコード |

### 5.4 Handler interface の実装

ogenが生成するHandler interfaceを実装する:

```go
// ogen が生成する interface (例)
type Handler interface {
    GetChannelInfo(ctx context.Context, params GetChannelInfoParams) (GetChannelInfoRes, error)
    ListDownloads(ctx context.Context, params ListDownloadsParams) (ListDownloadsRes, error)
    CreateDownload(ctx context.Context, req *CreateDownloadRequest, params CreateDownloadParams) (CreateDownloadRes, error)
}
```

### 5.5 ダウンロードステータス

ステータス遷移:

```
pending → downloading → converting → completed
                 │            │
                 └──► error ◄─┘
```

| ステータス | 説明 |
|-----------|------|
| `pending` | キューに追加済み、ダウンロード未開始 |
| `downloading` | yt-dlpでダウンロード中 |
| `converting` | 音声フォーマット変換中 |
| `completed` | 完了 |
| `error` | エラー発生 |

## 6. 設定ファイル

ファイル: `config.yaml`

```yaml
server:
  port: 8080
  # フロントエンドの静的ファイルのパス
  static_dir: "./frontend/dist"

database:
  # SQLiteファイルのパス
  path: "./data/media-downloader.db"

ytdlp:
  # yt-dlpバイナリのパス
  path: "yt-dlp"
  # 音声フォーマット (yt-dlpの--audio-formatオプション)
  # bestaudioのデフォルト出力フォーマットを使用する場合は空にする
  audio_format: ""

# シークレットリンクの定義
channels:
  - secret: "abc123def456"
    name: "Music Collection A"
    output_dir: "/data/music_a"

  - secret: "xyz789ghi012"
    name: "Music Collection B"
    output_dir: "/data/music_b"
```

## 7. データベース

### 7.1 sqlc ワークフロー

```
SQLスキーマ定義 (schema.sql)
SQLクエリ定義 (queries.sql)
       │
       ▼  sqlc generate
Go 型安全クエリコード (db/, models.go, queries.sql.go)
```

### 7.2 sqlc 設定

ファイル: `backend/sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "db/queries.sql"
    schema: "db/schema.sql"
    gen:
      go:
        package: "db"
        out: "db"
```

### 7.3 スキーマ定義

ファイル: `backend/db/schema.sql`

```sql
CREATE TABLE downloads (
    id          TEXT PRIMARY KEY,                -- UUID
    channel     TEXT NOT NULL,                   -- シークレットリンク (channelsのsecret)
    url         TEXT NOT NULL,                   -- YouTube URL
    title       TEXT,                            -- 動画タイトル
    status      TEXT NOT NULL DEFAULT 'pending', -- pending/downloading/converting/completed/error
    progress    INTEGER NOT NULL DEFAULT 0,      -- 進行率 (0-100)
    filename    TEXT,                            -- 保存ファイル名
    error       TEXT,                            -- エラーメッセージ
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE INDEX idx_downloads_channel ON downloads(channel);
CREATE INDEX idx_downloads_created_at ON downloads(created_at DESC);
```

### 7.4 クエリ定義

ファイル: `backend/db/queries.sql`

```sql
-- name: GetDownloadsByChannel :many
SELECT * FROM downloads
WHERE channel = ?
ORDER BY created_at DESC;

-- name: CreateDownload :one
INSERT INTO downloads (id, channel, url, status, progress)
VALUES (?, ?, ?, 'pending', 0)
RETURNING *;

-- name: UpdateDownloadStatus :exec
UPDATE downloads
SET status = ?, progress = ?, title = ?, filename = ?, error = ?,
    completed_at = CASE WHEN ? IN ('completed', 'error') THEN CURRENT_TIMESTAMP ELSE NULL END
WHERE id = ?;
```

sqlcがこれらから以下を自動生成する:
- `db/models.go` — `Download` 構造体
- `db/queries.sql.go` — 型安全なクエリ関数 (`GetDownloadsByChannel`, `CreateDownload`, `UpdateDownloadStatus`)

## 8. フロントエンド

### 8.1 ページ構成

| パス | 内容 |
|------|------|
| `/{secret}` | シークレットリンクのメインページ |

※ トップページ (`/`) は不要。シークレットリンクを知っているユーザーのみがアクセスする。

### 8.2 画面構成

```
┌──────────────────────────────────────┐
│  {channel名}                         │
├──────────────────────────────────────┤
│                                      │
│  YouTube URL: [________________] [DL]│
│                                      │
├──────────────────────────────────────┤
│  ダウンロード一覧                      │
│                                      │
│  ● Example Song A                    │
│    ✓ 完了 - example_song_a.opus      │
│                                      │
│  ● Example Song B                    │
│    ↓ ダウンロード中 72%  ████████░░   │
│                                      │
│  ● Example Song C                    │
│    ✗ エラー: Invalid URL             │
│                                      │
└──────────────────────────────────────┘
```

### 8.3 ポーリング

- ダウンロード中のタスクが存在する間、2秒間隔で `GET /api/{secret}/downloads` をポーリングする
- 全タスクが完了/エラーの場合はポーリングを停止する

## 9. Docker

### 9.1 Dockerfile (multi-stage build)

```
Stage 1: Frontend build
  - Node.js イメージ
  - npm install & npm run build

Stage 2: Backend build
  - Go イメージ
  - go build

Stage 3: Runtime
  - 軽量ベースイメージ (debian-slim等)
  - yt-dlp, ffmpeg をインストール
  - Stage 1, 2 の成果物をコピー
```

### 9.2 docker-compose.yaml (参考)

```yaml
services:
  media-downloader:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./data:/app/data          # SQLiteファイル
      - /path/to/music_a:/data/music_a
      - /path/to/music_b:/data/music_b
```

## 10. テスト

### 10.1 テスト方針

| 種別 | ツール | 対象 |
|------|--------|------|
| API E2Eテスト | runn | APIエンドポイントのシナリオテスト |
| ユニットテスト | Go標準 (`testing`) | yt-dlp出力パース、URL検証など複雑なロジック |

### 10.2 API E2Eテスト (runn)

YAMLでHTTPリクエストのシナリオを定義し、APIの一連の動作を検証する。

ファイル: `e2e/downloads.yaml`

```yaml
desc: ダウンロードの作成と一覧取得
runners:
  req:
    endpoint: http://localhost:8080
steps:
  - desc: チャンネル情報を取得できる
    req:
      /api/{{ vars.secret }}/channel:
        get:
          body: null
    test: |
      current.res.status == 200
      && current.res.body.name != ""
  - desc: ダウンロードを作成できる
    req:
      /api/{{ vars.secret }}/downloads:
        post:
          body:
            application/json:
              url: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
    test: |
      current.res.status == 202
      && current.res.body.status == "pending"
    bind:
      download_id: current.res.body.id
  - desc: ダウンロード一覧に含まれる
    req:
      /api/{{ vars.secret }}/downloads:
        get:
          body: null
    test: |
      current.res.status == 200
      && len(current.res.body.downloads) > 0
```

ファイル: `e2e/secret_link.yaml`

```yaml
desc: 無効なシークレットリンクの検証
runners:
  req:
    endpoint: http://localhost:8080
steps:
  - desc: 存在しないシークレットリンクで404を返す
    req:
      /api/invalid-secret/downloads:
        get:
          body: null
    test: |
      current.res.status == 404
```

### 10.3 ユニットテスト

以下の箇所にユニットテストを書く:

| 対象 | テスト内容 |
|------|-----------|
| `worker/` | yt-dlpの標準出力から進捗率・タイトルを正しくパースできるか |
| `handler/` or `service/` | YouTube URL のバリデーション |

### 10.4 実行方法

```bash
# API E2Eテスト (サーバー起動状態で実行)
mise run test:e2e

# ユニットテスト
mise run test:unit

# 全テスト
mise run test
```

## 11. mise (ツール管理・タスクランナー)

### 10.1 概要

mise を使用してプロジェクトのツールバージョン管理とタスク実行を一元化する。

### 10.2 `.mise.toml`

```toml
[tools]
go = "1.26.1"
node = "25.8.2"
pnpm = "10.33.0"
sqlc = "1.30.0"
"go:github.com/ogen-go/ogen/cmd/ogen" = "1.20.1"
"go:github.com/k1LoW/runn/cmd/runn" = "1.6.1"

# タスク定義
[tasks.gen]
description = "全コード生成を実行"
depends = ["gen:go", "gen:tsp"]

[tasks."gen:tsp"]
description = "TypeSpec → OpenAPI を生成"
run = "cd spec && pnpm exec -- tsp compile ."

[tasks."gen:go"]
description = "Go のコード生成を実行"
depends = ["gen:go:ogen", "gen:go:sqlc"]

[tasks."gen:go:ogen"]
description = "ogen で API コードを生成"
depends = ["gen:tsp"]
run = "ogen --target ./backend/api/ -package api --clean ./spec/tsp-output/openapi.yaml"

[tasks."gen:go:sqlc"]
description = "sqlc で DB クエリコードを生成"
run = "sqlc generate -f ./backend/sqlc.yaml"

[tasks.test]
description = "全テストを実行"
depends = ["test:unit", "test:e2e"]

[tasks."test:unit"]
description = "ユニットテストを実行"
run = "cd backend && go test ./..."

[tasks."test:e2e"]
description = "API E2Eテストを実行 (サーバー起動状態で)"
run = "runn run e2e/*.yaml"

[tasks.clean]
description = "生成コードを削除"
run = "rm -rf ./backend/api/oas_*_gen.go ./backend/db/models.go ./backend/db/queries.sql.go"
```

### 10.3 利用方法

```bash
# ツールのインストール
mise install

# 全コード生成
mise run gen

# 個別実行
mise run gen:tsp
mise run gen:go:ogen
mise run gen:go:sqlc

# 生成コードのクリーンアップ
mise run clean
```

## 11. ディレクトリ構成

```
media-downloader/
├── .mise.toml             # ツール管理・タスク定義
├── package.json           # pnpm workspace ルート
├── pnpm-workspace.yaml    # workspace メンバー定義
├── config.yaml            # アプリ設定
├── Dockerfile
├── docker-compose.yaml
├── docs/
│   └── specification.md
├── spec/                  # TypeSpec定義 (pnpm workspace)
│   ├── package.json
│   ├── tspconfig.yaml
│   ├── main.tsp
│   └── tsp-output/        # 生成: OpenAPI仕様
│       └── @typespec/openapi3/
│           └── openapi.yaml
├── backend/
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   ├── sqlc.yaml
│   ├── api/               # ogen生成コード
│   │   └── oas_*_gen.go
│   ├── db/                # sqlc定義 + 生成コード
│   │   ├── schema.sql
│   │   ├── queries.sql
│   │   ├── models.go      # 生成
│   │   └── queries.sql.go # 生成
│   ├── config/
│   │   └── config.go
│   ├── handler/
│   │   └── handler.go
│   ├── service/
│   │   └── service.go
│   └── worker/
│       ├── worker.go
│       └── worker_test.go
├── frontend/              # React フロントエンド (pnpm workspace)
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── pages/
│       │   └── ChannelPage.tsx
│       └── api/
│           └── client.ts
└── e2e/                   # API E2Eテスト (runn)
    ├── downloads.yaml
    └── secret_link.yaml
```
