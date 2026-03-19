# High & Low

認証付きで `1〜13` のカードを使って戦う「High & Low」ゲームの Web API（Go/Echo + PostgreSQL + Redis）と、React フロントエンドです。

## 構成

- `backend/`: Go + Echo（API / ゲームロジック / DB永続 / Redis Rate Limit）
- `frontend/01/`: React + Vite（UI）
- `docker-compose.yml`: PostgreSQL / Redis / backend / frontend をまとめて起動

## 起動（Docker）

`docker compose up --build`

起動後のアクセス先:

- フロント: `http://localhost:5175`
- API 健康確認: `http://localhost:8080/healthz`

初回は PostgreSQL / Redis の DB が初期化され、以降は `docker-compose.yml` の volume に保存されます。

## ローカル開発（環境変数）

バックエンドは `backend/.env`（または環境変数）を読みます。

最低限必要な環境変数（`backend/.env`）:

- `DB_DSN`
- `REDIS_ADDR`

`docker-compose.yml` では、さらに `COOKIE_SECURE` を `false` に設定してローカル開発で `Secure` Cookie 問題を回避しています。

レート制限（任意。デフォルト値は `backend/main.go` 内）:

- `RATE_LIMIT_CAPACITY`
- `RATE_LIMIT_REFILL_RATE`
- `RATE_LIMIT_TOKEN_COST`
- `RATE_LIMIT_TTL_SEC`

## 認証・CSRF

### 認証

- ログイン後、`session_id` Cookie が発行されます。
- API のうちゲーム系は `session_id` が必要です。

### CSRF（Double Submit Cookie）

POST は CSRF 対策として以下を要求します:

- `csrf_token` Cookie
- リクエストヘッダ `X-CSRF-Token`（Cookie と一致させる）

フロント側は `frontend/01/src/api/http.ts` で、Cookie から `csrf_token` を読み出して `X-CSRF-Token` を付与しています。

## API（エンドポイント概要）

### 共通

レスポンスは成功・失敗ともに以下の形です:

- 成功: `{ "success": true, "data": ... }`
- 失敗: `{ "success": false, "error": { "code": "...", "message": "..." } }`

### 健康確認

- `GET /healthz`

### 認証（ユーザー）

- `POST /signup`
- `POST /login`
- `POST /logout`（CSRF + 要認証）

### ゲーム

- `GET /api/game/status`（要認証）
- `POST /api/game/start`（要認証 + CSRF）
- `POST /api/game/select`（要認証 + CSRF）
- `POST /api/game/cheat`（要認証 + CSRF）
- `POST /api/game/reset`（要認証 + CSRF、開発・デバッグ用途）
- `POST /api/game/mode`（要認証 + CSRF）

ゲーム状態の取得は `status` で:

- `NOT_STARTED / IN_PROGRESS / FINISHED`
- `ver` をクライアントで保持し、更新系リクエスト（start/select/cheat/mode/reset）に送信します

### Rate Limit

- 対象: ゲームの状態を変更する `POST`（start/select/cheat/mode/reset）
- 超過時: `429 Too Many Requests`
- 追加ヘッダ: `Retry-After`（次に許可されるまでの秒数）

## フロント開発

`cd frontend/01` の上で:

- `npm install`
- `npm run dev`

`docker-compose.yml` を使う場合、frontend はすでに同梱ビルド・起動されます。

## 補足（ゲーム仕様の要点）

- 1 セットは「先に 2 勝で終了」
- モードは `PLAYER` / `DEALER`（セット開始時に決まる。IN_PROGRESS では変更できない）
- `DEALER` のみ cheat が使用可能（1 セット 1 回）
- `DRAW` が 5 回連続した場合は used cards の履歴のみリセット（勝数は維持）

