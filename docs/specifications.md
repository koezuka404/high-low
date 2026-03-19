# High & Low 実装仕様書

## 目次

1. システム概要
2. 用語定義（dealer_remaining の定義を含む）
3. アーキテクチャ
4. ドメイン仕様（ゲームルール）
5. 状態管理（PostgreSQL）
6. API フロー & ver 管理ガイド
7. API 仕様
8. 認証
9. エラー仕様
10. テスト要件
11. 補足・制約事項

---

## 1. システム概要

### 1.1 目的

認証済みユーザーが 1～13 のカードを使用し、High & Low ゲームを行う Web API を提供する。

### 1.2 ゲーム仕様

- 1 セットは 2 勝先取で終了
- モード： PLAYER / DEALER（1 セット開始時に決定、セット中は変更不可）
- DEALER モードのみ cheat（イカサマ）が使用可能（1 セット 1 回）
- DRAW が 5 回連続した場合、使用済みカードのみリセットして継続（勝数は維持）
- ゲーム状態は PostgreSQL に保存する（Redis には保存しない）
- Redis は Rate Limit（スロットリング）用途のみに使用する

---

## 2. 用語定義

| 用語 | 定義 |
|------|------|
| セット | 2 勝先取で終了するゲームの 1 単位。start API で開始、どちらかが 2 勝で FINISHED となる。 |
| ラウンド | select API 1回分。プレイヤー・ディーラー双方のカードを remaining からランダム選択し比較する単位。 |
| セッション | PostgreSQL の game_sessions テーブルに保存されるゲーム状態レコード |
| dealer_remaining | 【重要】{1～13} から dealer_used_cards を除いた集合。Redis には保存せず、select・cheat の処理先頭で毎回動的に計算する一時変数。計算式：`dealer_remaining = {1..13} − dealer_used_cards` |
| player_remaining | {1～13} から player_used_cards を除いた集合。dealer_remaining と同様に動的計算。Redis には保存しない。 |
| ver | 楽観的ロック用バージョン番号。新規セッションは ver=1 から開始。更新系 API が成功するたびに ver+1 される。クライアントは直前のレスポンスの ver を次リクエストに使用する。 |
| cheat_reserved | cheat API 実行後に true となる予約フラグ。true の間は次 select で dealer_card が強制的に cheat_card になる。消費後は false に戻る。 |

### 【dealer_remaining は Redis に保存しない】

`dealer_remaining` はセッションの JSON フィールドではない。

`select`・`cheat` の処理先頭で `{1..13} − dealer_used_cards` を計算して使う一時変数。

`dealer_used_cards` が空なら `dealer_remaining = {1,2,...,13}`（13 枚全て残り）。

---

## 3. アーキテクチャ

### 3.1 採用方針

| 層 | 責務 | 主要コンポーネント |
|----|------|------------------|
| Controller | HTTP 入出力・バリデーション・認証チェック | game_controller.go |
| Usecase | 状態更新・ドメイン呼び出し・DB読み書き | game_usecase.go |
| Domain | 勝敗判定・制約チェック（副作用なし・純粋関数） | game_domain.go |
| Infra | PostgreSQL操作（ゲーム状態）Redis操作（RateLimit）DB操作（ユーザー認証） | game_repo.go / user_repo.go / rate_limit_repo.go |

---

## 4. ドメイン仕様（ゲームルール）

### 4.1 列挙型

| 型名 | 値 |
|------|----|
| GameStatus | NOT_STARTED / IN_PROGRESS / FINISHED |
| GameMode | PLAYER / DEALER |
| RoundResult | PLAYER_WIN / DEALER_WIN / DRAW |

### 4.2 基本ルール

- カードは整数 1～13
- セット中は使用済みカードを再利用不可
- `player_used_cards` と `dealer_used_cards` は独立（同じ数字を両者が別々に使用可能）
- 勝敗：カード値が大きい方が勝ち、同値は DRAW
- どちらかが 2 勝到達で `status = FINISHED`

### 4.3 DRAW5 リセット仕様

DRAW が 5 回連続（`consecutive_draws == 5` 到達）した場合のリセット手順：

**【リセット順序の確定仕様】**

- 手順 1：現ラウンドの `player_card` を `player_used_cards` に追加
- 手順 2：現ラウンドの `dealer_card` を `dealer_used_cards` に追加
- 手順 3：`player_used_cards = []`（空配列にリセット）
- 手順 4：`dealer_used_cards = []`（空配列にリセット）
- 手順 5：`consecutive_draws = 0` にリセット

→ DRAW5 到達ラウンドのカードはリセット後の used_cards に含まれない（空配列になる）

**変更しないもの：**

- `player_wins` / `dealer_wins`（勝数は維持）
- `status`（IN_PROGRESS を維持）
- `cheat_reserved` / `cheat_card`（予約済み cheat は維持）

### 4.4 cheat（イカサマ）仕様

#### 実行条件

- `mode == DEALER`
- `status == IN_PROGRESS`
- `cheated == false`（1 セット 1 回のみ）
- `ver` が一致すること（不一致は 409）

#### 処理フロー

- `dealer_remaining` を計算（`{1..13} − dealer_used_cards`）
- `dealer_remaining` が空の場合 → 400 `cheat_not_available`
- `cheat_card = max(dealer_remaining)` を確定
- `cheated = true` にセット
- `cheat_reserved = true` にセット
- `ver = ver + 1`

#### 予約消費（次回 select 時）

- `cheat_reserved == true` の場合、`dealer_card = cheat_card` を強制使用
- `dealer_card` を `dealer_used_cards` に追加（通常通り）
- select 処理成功後に `cheat_reserved = false`、`cheat_card = null` にリセット（消費）

#### DRAW5 リセットとの整合

DRAW が 5 回連続した場合、`used_cards` はリセットされる。このリセットは **カード使用履歴のみを初期化する処理** であり、**cheat の予約状態には影響しない**。

| 項目 | DRAW5 リセット後の状態 |
|------|-----------------------|
| player_used_cards | 空配列にリセット |
| dealer_used_cards | 空配列にリセット |
| cheat_reserved | 変更なし（予約状態を維持） |
| cheat_card | 変更なし（予約されたカードを維持） |

**挙動：**

- `cheat_reserved = true` の状態で DRAW5 が発生しても、cheat 予約は **消えない**
- 次回 `select` 実行時には **予約された `cheat_card` がそのまま使用される**

### 4.5 数字のランダム選択アルゴリズム

本システムにおけるカード数字のランダム表示は、未使用カード集合（remaining）から一様ランダムに 1 件選択する方式を採用する。

プレイヤー側・ディーラー側はそれぞれ独立した使用履歴を持つため、同一ラウンドで同じ数字が選ばれることは許容する。

#### 4.5.1 基本方針

- 使用可能な数字は **1～13 の整数** とする
- 各ラウンド開始時に、以下を動的計算する
  - `player_remaining = {1..13} − player_used_cards`
  - `dealer_remaining = {1..13} − dealer_used_cards`
- `player_card` は `player_remaining` からランダムに 1 件選択する
- `dealer_card` は通常時 `dealer_remaining` からランダムに 1 件選択する
- ただし `cheat_reserved == true` の場合、`dealer_card = cheat_card` を強制使用し、このラウンドでは通常抽選を行わない
- 選択後のカードは、それぞれ対応する `used_cards` に追加する
- 1 セット中は、同一プレイヤー内では同じ数字を再利用しない

#### 4.5.2 ランダム選択手順

**player_card 選択手順**

1. `player_remaining` を昇順配列として生成する（例: `[1,2,4,6,9,10,13]`）
2. 配列長を `n` とする
3. `0` 以上 `n-1` 以下の整数乱数 `idx` を 1 回生成する
4. `player_card = player_remaining[idx]` とする

**dealer_card 選択手順（通常時）**

1. `dealer_remaining` を昇順配列として生成する
2. 配列長を `n` とする
3. `0` 以上 `n-1` 以下の整数乱数 `idx` を 1 回生成する
4. `dealer_card = dealer_remaining[idx]` とする

#### 4.5.3 一様性

- remaining 配列内の各数字は **同一確率** で選択されるものとする
- 重み付け抽選は行わない
- 直前ラウンド結果、勝敗状況、モード、ver、ユーザー情報などによって選択確率を変動させない
- cheat 使用時のみ、ディーラー側は例外的に `cheat_card` を固定使用する

#### 4.5.4 再抽選禁止

- 抽選後に勝敗を見て有利不利を補正する再抽選は行わない
- 「DRAW だったから引き直す」「弱い数字だったから再抽選する」といった処理は禁止する
- 1 ラウンド中に各陣営の抽選は 1 回のみとする
- ただし cheat 予約時のディーラーは抽選ではなく固定値採用とする

#### 4.5.5 同値（DRAW）の扱い

- `player_card == dealer_card` の場合は DRAW とする
- プレイヤー側とディーラー側の使用履歴は独立しているため、両者が同じ数字を引くこと自体は正常動作とする
- DRAW であっても、そのラウンドで使用した `player_card` と `dealer_card` は通常通り `used_cards` に追加する
- `consecutive_draws` を 1 加算する

#### 4.5.6 DRAW5 発生時の整合

DRAW が 5 回連続した場合は、既存仕様に従い以下を実行する。

1. 当該ラウンドの `player_card` を `player_used_cards` に追加
2. 当該ラウンドの `dealer_card` を `dealer_used_cards` に追加
3. `player_used_cards = []`
4. `dealer_used_cards = []`
5. `consecutive_draws = 0`

これにより、次ラウンド以降の remaining は再び 1..13 全体になる。

ただし `player_wins` / `dealer_wins` は維持し、`cheat_reserved` / `cheat_card` も既存仕様通り維持する。

#### 4.5.7 cheat 予約時の優先順位

- `cheat_reserved == true` の場合、ディーラー側の数字決定は通常ランダム選択より優先される
- このとき `dealer_card = cheat_card` とし、`dealer_remaining` からのランダム抽選は行わない
- 使用後は既存仕様通り `cheat_reserved = false`、`cheat_card = null` に戻す

#### 4.5.8 remaining 生成ルール

remaining は DB に保存せず、各 API 処理時にその場で計算する一時値とする。

生成例：
```
player_used_cards = [2,5,9]
player_remaining  = [1,3,4,6,7,8,10,11,12,13]

dealer_used_cards = [1,4,7,13]
dealer_remaining  = [2,3,5,6,8,9,10,11,12]
```

remaining の並び順は **昇順固定** とする。これにより、乱数インデックスと選択値の対応が実装間でぶれないようにする。

#### 4.5.9 実装制約

- 乱数は **疑似乱数生成器** を使用してよい
- 実装言語標準ライブラリの乱数関数を利用してよい
- 乱数は「remaining 配列の添字」を選ぶ用途にのみ使用し、1～13 全体から乱数を出して未使用判定に失敗したら再抽選、という方式は採用しない
- 必ず **remaining を先に構築してから、その配列から 1 回で選択** する
- この方式により、再抽選ループや偏りのある実装を防止する

#### 4.5.10 擬似コード

```python
all_cards = [1,2,3,4,5,6,7,8,9,10,11,12,13]

player_remaining = all_cards - player_used_cards
dealer_remaining = all_cards - dealer_used_cards

player_card = random_one(player_remaining)

if cheat_reserved == true:
    dealer_card = cheat_card
else:
    dealer_card = random_one(dealer_remaining)

player_used_cards に player_card を追加
dealer_used_cards に dealer_card を追加

if player_card > dealer_card:
    result = PLAYER_WIN
elif player_card < dealer_card:
    result = DEALER_WIN
else:
    result = DRAW
```

---

## 5. 状態管理（PostgreSQL）

### 5.1 採用方針

- ゲーム状態・history（勝敗ログ）・ver（楽観ロック）は PostgreSQL で永続管理する
- Redis は Rate Limit のみを管理する（ゲーム状態は保持しない）

### 5.2 PostgreSQL 管理対象

- `game_sessions`（現在のセット状態）
- `game_round_logs`（ラウンドログ / history）
- `ver`（楽観ロック用バージョン番号）

### 5.3 楽観ロック（DB）

更新系 API（start / select / cheat / mode）はリクエストに `ver` を含める。

DB 更新は「`WHERE id=? AND ver=?`」で実行し、更新成功時に `ver` を +1 する。

更新件数が 0 の場合：
- ver 不一致として `409 version_conflict` を返却する（状態は変更しない）

### 5.4 Redis（Rate Limit のみ）

Redis は userID 単位で Rate Limit を実施する（詳細は 11.5 参照）。

ゲーム状態・ログ・ver は Redis に保存しない。

---

## 6. API フロー & ver 管理ガイド

### 6.1 ver 管理フロー（クライアント実装ガイド）

ver はクライアントがローカル変数として保持し、全ての更新系リクエストに含める。

**【ver の取得・管理手順】**

1. start レスポンスの ver をローカル変数に保存する（初回は ver=1）
2. select / cheat / mode の各リクエストに保存した ver を含める
3. 各レスポンスで返る ver でローカル変数を上書きする
4. 409 conflict を受信した場合 → `GET /api/game/status` で最新 ver を再取得してリトライ
5. 画面リロード等でローカル変数が失われた場合も `GET /api/game/status` で ver を再取得する

### 6.2 通常ゲームフロー

| # | API | 説明 |
|---|-----|------|
| 1 | POST /api/game/start | 新規セッション作成。ver = 1 で開始。レスポンスで session_id と ver を受け取り、クライアントはローカル状態として保存する。以降の select API では保存した session_id を Request Body に含めて送信する。 |
| 2 | POST /api/game/select | カード選択。保存した ver を含めて送信する。レスポンスで 新しい ver（ver + 1） を受け取り更新する。 |
| 3 | 繰り返し | どちらかが 2勝 するまで select を繰り返す。毎回 ver を更新する。 |
| 4 | status = FINISHED | select レスポンスで status = FINISHED を受信したらセット終了。 |
| 5 | POST /api/game/mode | 次セットのモード変更。FINISHED 状態でのみ使用可能。 |
| 6 | POST /api/game/start | 新セット開始。現在の ver を含めて送信。レスポンスで ver = old_ver + 1 を受け取る。 |

### 6.3 cheat フロー（DEALER モード時）

| # | API | 説明 |
|---|-----|------|
| 1 | POST /api/game/start | DEALER モードでセット開始。ver を保存。 |
| 2 | POST /api/game/cheat | 任意タイミングで実行。dealer_remaining の最大値を予約。ver が+1 されるので更新する。 |
| 3 | POST /api/game/select | 次の select で dealer_card = cheat_card が強制使用される。その後 cheat_reserved は false に戻る。 |
| 4 | （以降通常） | 以降は dealer_remaining からのランダム選択に戻る。 |

**【POST /api/game/mode の使用目的】**

FINISHED 状態で次のセットのモードを事前に切り替えるための API。

セット中（IN_PROGRESS）には変更できない。

変更後に `POST /api/game/start` を呼ぶことで新モードで次セットが開始される。

変更しない場合、前セットのモードが引き継がれる（初回デフォルトは PLAYER）。

---

## 7. API 仕様

### 7.0 共通形式

**成功レスポンス**

```json
{
  "success": true,
  "data": { ... }
}
```

| フィールド | 型 | 説明 |
|------------|-----|------|
| success | boolean | 処理成功時 true |
| data | object | APIごとのレスポンス |

**失敗レスポンス**

```json
{
  "success": false,
  "error": {
    "code": "error_code",
    "message": "error message"
  }
}
```

| フィールド | 型 | 説明 |
|------------|-----|------|
| success | boolean | 処理失敗時 false |
| error.code | string | エラーコード |
| error.message | string | エラー内容 |

### 7.1 エラーコード一覧

| code | HTTP | 説明 |
|------|------|------|
| invalid_input | 400 | 型・範囲不正（card が 1〜13 以外、小数、文字列など） |
| invalid_game_state | 400 | 状態不正（IN_PROGRESS で start、FINISHED 以外で mode 変更 等） |
| game_not_started | 400 | セッションは存在するがセット未開始（NOT_STARTED / FINISHED で select 等） |
| game_not_finished | 400 | セット終了していない（FINISHED 以外で mode 変更 等） |
| game_already_started | 400 | セット進行中（IN_PROGRESS 中に start 等） |
| invalid_mode | 400 | mode が PLAYER / DEALER 以外 |
| cheat_not_available | 400 | dealer_remaining が空 |
| cheat_already_used | 400 | cheated == true |
| cheat_not_allowed | 400 | DEALER モード以外で cheat 実行 |
| unauthorized | 401 | session_id Cookie なし / セッション無効 / 期限切れ |
| forbidden | 403 | 他ユーザーのセッション操作 |
| session_not_found | 404 | 対象ゲームセッション不存在 |
| version_conflict | 409 | ver 不一致 |
| too_many_requests | 429 | レート制限超過 |
| internal_error | 500 | Redis接続エラー / 予期しない例外 |
| invalid_json | 400 | JSON 形式不正（Request Body のパース失敗、必須キー欠落など） |

### 7.2 POST /api/game/start

セット開始

**Request Body**

| フィールド | 型 | 必須 | 説明 |
|------------|-----|------|------|
| ver | integer | 任意 | 再start時 |

**シチュエーション**

| 状況 | 処理 | HTTP |
|------|------|------|
| セッション未存在 | 新規作成 | 200 |
| セット終了 | 初期化 | 200 |
| セット進行中 | エラー | 400 |

**エラー**

| code | HTTP | 説明 |
|------|------|------|
| game_already_started | 400 | セット進行中 |
| version_conflict | 409 | ver不一致 |

**成功レスポンス**

```json
{
  "success": true,
  "data": {
    "session_id": 1,
    "mode": "PLAYER",
    "player_wins": 0,
    "dealer_wins": 0,
    "ver": 1
  }
}
```

### 7.3 POST /api/game/select

プレイヤーおよびディーラーのカードを決定し、1 ラウンド分の勝敗判定と状態更新を行う API。

#### 7.3.0 エンドポイント

`POST /api/game/select`

#### 7.3.1 Request Body

```json
{
  "session_id": 1,
  "ver": 3
}
```

- `session_id`: integer, 必須
  - クライアントが「このセッションだ」と認識しているゲームセッション ID
  - サーバー側で取得した `game.id` と一致しているかの検証に使う
- `ver`: integer, 必須
  - 楽観ロック用バージョン番号
  - 直前のレスポンス（start / select / cheat / mode / status）で受け取った `ver` をそのまま送る

#### 7.3.2 認証・前提条件

- Cookie セッション認証済みであること（`session_id` Cookie）
- CSRF トークンが正しいこと（Double Submit Cookie）
- 認証に失敗した場合
  - HTTP 401
  - `code: unauthorized`

#### 7.3.3 セッション取得

1. 認証情報（Cookie の `session_id`）から `user_id` を特定する。
2. `user_id` をキーとしてゲームセッションを 1 件取得する（1 ユーザー 1 セッション前提）。

取得結果に応じて以下のように扱う。

- **セッションが存在しない場合**
  - HTTP 404
  - `code: session_not_found`
- **セッションが存在するが、`game.id != session_id`（Request Body）**
  - 別ユーザーや別タブなどの不整合とみなす
  - HTTP 403
  - `code: forbidden`
- **status が NOT_STARTED または FINISHED の場合**
  - まだセットが開始されていない / すでに終了しているため select 不可
  - HTTP 400
  - `code: game_not_started`
- **それ以外（IN_PROGRESS かつ user_id / session_id が一致）**
  - select 続行

#### 7.3.4 ver チェック（楽観ロック）

Request Body の `ver` と、取得したセッションの `game.ver` を比較する。

- 一致しない場合
  - DB 更新は行わずエラーを返す。
  - HTTP 409
  - `code: version_conflict`
  - クライアントは `GET /api/game/status` で最新 ver を再取得してからリトライする。

#### 7.3.5 remaining 計算

プレイヤーとディーラーはそれぞれ独立したカード履歴を持つ。

```
all_cards         = [1..13]
player_remaining  = all_cards − player_used_cards
dealer_remaining  = all_cards − dealer_used_cards
```

ルール：
- remaining は DB に保存しない一時値。
- 各 API 呼び出し時に、その時点の used_cards から動的に計算する。

#### 7.3.6 プレイヤーカード決定（ランダム）

1. `player_remaining` を昇順配列に変換する（例: `[1,2,4,6,8,10,11,13]`）
2. 配列長を `n` とする。
3. `0〜n-1` の整数乱数 `idx` を 1 回生成する。
4. `player_card = player_remaining[idx]` とする。

性質：
- remaining 内の各数字は **完全に一様確率で選ばれる**。
- **再抽選は禁止**（弱い数字だったから引き直す、などは行わない）。

#### 7.3.7 ディーラーカード決定

**cheat 未予約時（通常）**

1. `dealer_remaining` を昇順配列に変換する。
2. 配列長を `n` とする。
3. `0〜n-1` の整数乱数 `idx` を 1 回生成する。
4. `dealer_card = dealer_remaining[idx]` とする。

**cheat 予約あり（`cheat_reserved == true`）**

- このラウンドではランダム抽選を行わず、固定値を使用する。
- `dealer_card = cheat_card`
- 抽選はスキップし、`dealer_card` を `used_cards` に追加するタイミング以降は通常通り。

#### 7.3.8 使用カード更新

- `player_used_cards` に `player_card` を追加
- `dealer_used_cards` に `dealer_card` を追加

制約：
- 1 セット中、同一プレイヤー内で同じ数字は再利用しない。（プレイヤーとディーラーは別履歴なので、同じ値を引いてもよい）

#### 7.3.9 勝敗判定

```python
if player_card > dealer_card:
    result = PLAYER_WIN
elif player_card < dealer_card:
    result = DEALER_WIN
else:
    result = DRAW
```

#### 7.3.10 DRAW 処理と DRAW5 リセット

- `result == DRAW` の場合
  - `consecutive_draws += 1`
- `PLAYER_WIN` / `DEALER_WIN` の場合
  - `consecutive_draws = 0`

**DRAW が 5 回連続した場合（`consecutive_draws == 5` 到達）：**

1. 当該ラウンドの `player_card` を `player_used_cards` に追加（すでに追加済み前提）
2. 当該ラウンドの `dealer_card` を `dealer_used_cards` に追加（同上）
3. `player_used_cards = []`
4. `dealer_used_cards = []`
5. `consecutive_draws = 0`

リセットされないもの：
- `player_wins` / `dealer_wins`（勝数）
- `status`（IN_PROGRESS を維持）
- `cheat_reserved` / `cheat_card`（予約済み cheat は維持）

#### 7.3.11 cheat 消費

ラウンド開始時点で `cheat_reserved == true` だった場合：

- 上記のとおり `dealer_card = cheat_card` で固定値採用。
- ラウンド終了時に以下を実行する：
  - `cheat_reserved = false`
  - `cheat_card = null`

#### 7.3.12 勝利数・ゲーム状態更新

- `result == PLAYER_WIN` の場合：`player_wins += 1`
- `result == DEALER_WIN` の場合：`dealer_wins += 1`

セット終了条件：

```python
if player_wins == 2 or dealer_wins == 2:
    game_status = FINISHED
else:
    game_status = IN_PROGRESS のまま
```

#### 7.3.13 ver 更新

ラウンドの更新が正しく反映された場合、`ver = ver + 1` とする。

この新しい `ver` をレスポンスで返し、クライアントはローカルで保持して次の更新系 API に送る。

#### 7.3.14 レスポンス

```json
{
  "success": true,
  "data": {
    "player_card": 7,
    "dealer_card": 10,
    "result": "DEALER_WIN",
    "player_wins": 0,
    "dealer_wins": 1,
    "game_status": "IN_PROGRESS",
    "ver": 4
  }
}
```

フィールド：

| フィールド | 型 | 説明 |
|------------|-----|------|
| player_card | int | |
| dealer_card | int | |
| result | string | "PLAYER_WIN" \| "DEALER_WIN" \| "DRAW" |
| player_wins | int | セット内でのプレイヤー勝利数 |
| dealer_wins | int | セット内でのディーラー勝利数 |
| game_status | string | "NOT_STARTED" \| "IN_PROGRESS" \| "FINISHED" |
| ver | int | 現在のバージョン番号（次リクエストで送る値） |

#### 7.3.15 エラー

- **400 不正パラメータ**：JSON 形式不正、型不正、ver が数値でない 等
- **400 game_not_started**：セッションはあるが status が NOT_STARTED / FINISHED で select できない
- **401 未認証**：Cookie セッション不正・期限切れ
- **403 forbidden**：認証済みだが、user_id と session_id の組み合わせが不整合（他人のセッションを指定など）
- **404 session_not_found**：user_id でゲームセッションを取得できなかった
- **409 version_conflict**：Request Body の ver と DB の ver が一致しない（楽観ロック衝突）

### 7.4 POST /api/game/cheat

イカサマ予約

**Request Body**

```json
{
  "ver": 3
}
```

**前提条件**

| 条件 | エラー |
|------|--------|
| セッション不存在 | session_not_found |
| セット進行中 | game_not_started |
| DEALERモード | cheat_not_allowed |
| cheat未使用 | cheat_already_used |
| dealer_remaining存在 | cheat_not_available |
| ver一致 | version_conflict |

**セッション取得**

1. 認証情報（Cookie の `session_id`）から `user_id` を取得する。
2. `user_id` をキーとしてゲームセッションを取得する。

取得結果に応じて以下のように扱う。

- セッション不存在：HTTP 404 / `code: session_not_found`
- `status != IN_PROGRESS`：HTTP 400 / `code: game_not_started`

**エラー**

| code | HTTP | 説明 |
|------|------|------|
| session_not_found | 404 | ゲームセッション不存在 |
| cheat_not_allowed | 400 | PLAYERモード |
| cheat_already_used | 400 | 2回目 |
| cheat_not_available | 400 | dealer_remaining が空 |
| version_conflict | 409 | ver不一致 |

**成功レスポンス**

```json
{
  "success": true,
  "data": {
    "cheat_reserved": true,
    "cheat_card": 13,
    "ver": 4
  }
}
```

### 7.5 POST /api/game/mode

**Request Body**

```json
{
  "mode": "DEALER",
  "ver": 7
}
```

**前提条件**

| 条件 | エラー |
|------|--------|
| セッション不存在 | game_not_started |
| セット終了 | game_not_finished |
| mode正しい | invalid_mode |
| ver一致 | version_conflict |

**エラー**

| code | HTTP | 説明 |
|------|------|------|
| game_not_started | 400 | ゲームセッションが存在しない |
| game_not_finished | 400 | セットが進行中 / まだ FINISHED になっていない |
| invalid_mode | 400 | mode が PLAYER / DEALER 以外 |
| version_conflict | 409 | ver不一致 |

### 7.6 GET /api/game/status

ゲーム状態取得

**シチュエーション**

| 状況 | 処理 | HTTP |
|------|------|------|
| セッション存在 | 状態返却 | 200 |
| セッションなし | NOT_STARTED | 200 |

**セッション不存在時レスポンス**

```json
{
  "success": true,
  "data": {
    "session_id": 0,
    "status": "NOT_STARTED",
    "mode": "PLAYER",
    "player_wins": 0,
    "dealer_wins": 0,
    "cheated": false,
    "cheat_reserved": false,
    "ver": 0,
    "history": []
  }
}
```

補足：
- `session_id` は **0 または省略** として扱う
- クライアントは start API を呼び出して新セッションを作成する

**成功レスポンス**

```json
{
  "success": true,
  "data": {
    "session_id": 1,
    "status": "IN_PROGRESS",
    "mode": "PLAYER",
    "player_wins": 1,
    "dealer_wins": 0,
    "cheated": false,
    "cheat_reserved": false,
    "ver": 4,
    "history": []
  }
}
```

#### 7.6.1 レスポンスフィールド定義

| フィールド | 型 | 説明 |
|------------|-----|------|
| session_id | integer | セッションID（未存在時は 0） |
| status | string | ゲーム状態（NOT_STARTED / IN_PROGRESS / FINISHED） |
| mode | string | モード（PLAYER / DEALER） |
| player_wins | integer | プレイヤーの勝利数 |
| dealer_wins | integer | ディーラーの勝利数 |
| cheated | boolean | そのセットで cheat を使用済みか |
| cheat_reserved | boolean | 次回 select で cheat_card が適用される予約状態 |
| ver | integer | 楽観ロック用バージョン番号 |
| history | array | ラウンド履歴（詳細は別途定義） |

### 7.7 POST /api/game/reset

ゲーム状態の強制リセット（開発・デバッグ用途を想定）

**概要**

現在のゲームセッションを初期状態にリセットする。セッション自体は維持し、状態のみ初期化する。

**Request Body**

```json
{
  "ver": 5
}
```

| フィールド | 型 | 必須 | 説明 |
|------------|-----|------|------|
| ver | integer | 必須 | 楽観ロック用バージョン |

**前提条件**

| 条件 | エラー |
|------|--------|
| セッション不存在 | session_not_found |
| ver不一致 | version_conflict |

※ status に関係なく実行可能（実装準拠）

**処理内容**

以下を初期化する：

```
player_used_cards  = []
dealer_used_cards  = []
player_wins        = 0
dealer_wins        = 0
consecutive_draws  = 0
status             = IN_PROGRESS
cheated            = false
cheat_reserved     = false
cheat_card         = null
```

mode は維持する（実装準拠）

**レスポンス**

```json
{
  "success": true,
  "data": {
    "status": "IN_PROGRESS",
    "mode": "PLAYER",
    "ver": 6
  }
}
```

---

## 8. 認証

### 8.1 方針

本システムでは **Cookie セッション方式** を採用する。

認証済みユーザーはログイン後、サーバーが発行した **セッションIDを Cookie に保存** し、以降の API リクエストでは Cookie を利用して認証を行う。

CSRF 対策として **ダブルサブミットトークン方式** を採用する。

### 8.2 セッション管理

| 項目 | 内容 |
|------|------|
| セッション保存先 | PostgreSQL |
| セッションキー | `session_id` |
| TTL | 24時間 |
| 更新 | API成功時に TTL 更新 |

### 8.3 Cookie設定

| 属性 | 値 |
|------|----|
| HttpOnly | true |
| Secure | true |
| SameSite | Lax |
| Path | / |

Cookie例：

```
Set-Cookie: session_id=abc123; HttpOnly; Secure; SameSite=Lax
```

### 8.4 CSRF対策（ダブルサブミット）

本システムでは **Double Submit Cookie** を採用する。

**トークン発行**

ログイン時にサーバーは CSRF トークンを発行する。

```
Set-Cookie: csrf_token=xyz123
```

**クライアント送信**

更新系 API 呼び出し時：

```
X-CSRF-Token: xyz123
```

**検証**

サーバーは

- Cookie `csrf_token`
- Header `X-CSRF-Token`

が一致することを確認する。

一致しない場合：`403 forbidden` を返す。

### 8.5 認証フロー

1. ユーザーがログイン
2. サーバーが `session_id` Cookie を発行
3. クライアントは Cookie を自動送信
4. サーバーが PostgreSQL セッションを取得
5. 認証成功後 API 処理

### 8.6 API認証ヘッダー

更新系 API は以下を送信する。

```
Cookie: session_id=abc123
X-CSRF-Token: xyz123
```

### 8.7 セッション失効

| 条件 | 動作 |
|------|------|
| TTL期限切れ | PostgreSQL削除 |
| ログアウト | PostgreSQL削除 |
| セッション不存在 | 401 unauthorized |

---

## 9. エラー仕様

| HTTP | code | 説明・発生条件 |
|------|------|---------------|
| 400 | invalid_input | 型・範囲不正（card が 1〜13 以外、小数、文字列など） |
| 400 | invalid_game_state | 状態不正（IN_PROGRESS で start、FINISHED 以外で mode 変更 など） |
| 400 | game_not_started | セッション未開始（NOT_STARTED / FINISHED で select を呼んだ 等） |
| 400 | game_not_finished | セット終了していない（FINISHED 以外で mode 変更 等） |
| 400 | game_already_started | セット進行中（IN_PROGRESS 中に start 等） |
| 400 | invalid_mode | mode が PLAYER / DEALER 以外 |
| 400 | cheat_not_available | dealer_remaining が空（使えるカードがない） |
| 400 | cheat_already_used | cheated == true（1 セット内で 2 回 cheat しようとした） |
| 400 | cheat_not_allowed | DEALER モード以外で cheat を呼んだ |
| 401 | unauthorized | session_id Cookie なし / セッション無効 / 期限切れ |
| 403 | forbidden | 権限不足（他ユーザーのセッションを操作しようとした 等） |
| 404 | session_not_found | セッション不存在（select 時に対象セッションが存在しない） |
| 409 | version_conflict | ver 不一致。クライアントは GET /status で ver を再取得してリトライ |
| 429 | too_many_requests | レート制限超過 |
| 500 | internal_error | Redis 接続エラー・予期しない例外 |

---

## 10. テスト要件

### 10.1 単体テスト（Domain 層）

| # | テストケース | 期待値 |
|---|------------|--------|
| 1 | player_card > dealer_card | PLAYER_WIN、player_wins++、consecutive_draws=0 |
| 2 | player_card < dealer_card | DEALER_WIN、dealer_wins++、consecutive_draws=0 |
| 3 | player_card == dealer_card | DRAW、consecutive_draws++ |
| 4 | DRAW が 5 回連続 | used_cards リセット（空配列）、wins 維持、status=IN_PROGRESS 維持 |
| 5 | dealer_remaining=[3,7,11] で cheat | cheat_card = 11（最大値） |
| 6 | cheat_reserved=true で select | dealer_card = cheat_card、消費後 cheat_reserved=false、cheat_card=null |
| 7 | DRAW5 発生 + cheat 予約あり | used_cards=[] になるが cheat_card の値は変わらない |
| 8 | dealer_remaining が空で cheat | cheat_not_available エラー（400） |

### 10.2 結合テスト（API）

| # | シナリオ | 確認ポイント |
|---|---------|------------|
| 1 | start → select × N → FINISHED | 2勝到達で status=FINISHED になること |
| 2 | ver 不一致で select | 409 version_conflict が返り、セッション状態が変化しないこと |
| 3 | cheat → 次 select | dealer_card が cheat_card と一致すること。その後 cheat_reserved=false になること |
| 4 | FINISHED 以外で mode 変更 | 400 game_not_finished が返ること |
| 5 | IN_PROGRESS 中に start | 400 game_already_started が返ること |
| 6 | FINISHED → start → select | ver が old_ver+1 で再開されること。mode が引き継がれること |
| 7 | 未認証で任意 API 呼び出し | 401 unauthorized が返ること |
| 8 | 409 受信後に GET /status → リトライ | 最新 ver で select が成功すること |
| 9 | セッション不存在で select | 404 session_not_found が返ること |

---

## 11. 補足・制約事項

### 11.1 dealer_remaining の取り扱い

- `dealer_remaining` はセッションに保存するフィールドではない
- `select`・`cheat` の各処理の先頭で `{1..13} − dealer_used_cards` として動的に計算する
- `dealer_used_cards` が 13 枚埋まった場合は `dealer_remaining` が空になるが、DRAW5 リセットが先に発生しているはずなので通常は発生しない

### 11.2 モードの引き継ぎ

- FINISHED から再 start した場合、mode は前セットの値を引き継ぐ
- 明示的に変更したい場合は start 前に `POST /api/game/mode` を呼ぶ
- 初回 start 時のデフォルト mode = PLAYER

### 11.3 実装上の注意

- レート制限用 Redis キーの TTL は 60 秒とする（11.5.9 参照）。SETEX で毎回 60 秒にリセットすること。
- JSON デシリアライズ時に整数が float にキャストされないよう注意（言語によっては明示的なキャストが必要）
- card バリデーションは整数型チェックも含める（`"card": 5.5` は 400 `invalid_input`）
- history のラウンド番号はセット内で 1 から始まるカウンター。DRAW5 リセット後もカウントは継続する

### 11.4 スコープ外

- 複数ユーザー間の対戦機能
- Prometheus メトリクス・監査ログ
- Refresh Token・トークンローテーション

### 11.5 レート制限（Rate Limit）

#### 11.5.1 目的

API の過剰なリクエストによる **サーバー負荷・不正操作・DoS 的挙動** を防ぐため、ユーザー単位でレート制限を実施する。

本システムでは **Redis を使用した Token Bucket アルゴリズム** を採用する。

#### 11.5.2 制限単位

| 項目 | 内容 |
|------|------|
| 単位 | userID |
| 識別子 | userID（session_id から取得） |
| Redisキー | `ratelimit:user:{userID}` |

例：
```
ratelimit:12
ratelimit:45
ratelimit:102
```

#### 11.5.3 対象API

レート制限は **ゲーム状態を変更するAPIのみ** 対象とする。

| HTTP | Endpoint | 対象 |
|------|----------|------|
| POST | /api/game/start | 対象 |
| POST | /api/game/select | 対象 |
| POST | /api/game/cheat | 対象 |
| POST | /api/game/mode | 対象 |
| GET | /api/game/status | 対象外 |

理由：`GET /status` は状態取得のみで ver 更新や状態変更が発生しないため

#### 11.5.4 アルゴリズム

Token Bucket を採用する。

| パラメータ | 値 | 説明 |
|-----------|-----|------|
| capacity | 20 tokens | 最大保持トークン |
| refill_rate | 5 tokens/sec | 1秒あたり補充数 |
| token_cost | 1 token | 1リクエスト消費量 |

#### 11.5.5 バケットの動作

バケットの状態は Redis に保存する。

| フィールド | 型 | 説明 |
|------------|-----|------|
| tokens | float | 現在のトークン数 |
| last_refill | timestamp | 最後に補充した時刻 |

Redis保存例：

```
ratelimit:user:12
{
  tokens: 17
  last_refill: 1712345678
}
```

#### 11.5.6 リクエスト処理フロー

更新系 API 呼び出し時の処理：

1. `userID` を取得（`session_id` Cookie からセッションを取得し、その `userID` を使用）
   - ※ userID の解決に失敗した場合（セッション不正・期限切れ・取得失敗など）は、`userID = 0` として扱う。この場合の Redis キーは `ratelimit:user:0`。`userID = 0` は未認証または不正セッション用の共通バケットとする。
2. Redisキーを生成：`ratelimit:user:{userID}`
3. Redis に対して Lua Script を実行する
   - 現在時刻（now）を取得
   - 前回更新時刻（last_refill）を取得
   - トークン数（tokens）を取得
4. トークン補充処理（Token Bucket アルゴリズム）
   ```
   tokens += (now - last_refill) * refill_rate
   tokens = min(tokens, capacity)
   ```
5. リクエスト判定
   - `tokens >= 1` の場合：tokens を 1 消費し、リクエストを許可
   - `tokens < 1` の場合：リクエストを拒否（HTTP 429 を返却）
6. Redis に更新値を書き戻す（tokens、last_refill）

#### 11.5.7 超過時レスポンス

HTTP: `429 Too Many Requests`

```json
{
  "success": false,
  "error": {
    "code": "too_many_requests",
    "message": "rate limit exceeded"
  }
}
```

#### 11.5.8 Retry-After ヘッダー

次のトークンが補充されるまでの秒数を返す。

計算式：

```
wait = ceil((1 - tokens) / refill_rate)
```

例：

```
tokens = 0.2
refill_rate = 5
wait = ceil((1 - 0.2)/5) = ceil(0.16) = 1秒
```

レスポンスヘッダー：

```
Retry-After: 1
```

#### 11.5.9 Redis TTL

| 項目 | 値 |
|------|----|
| TTL | 60秒 |

理由：長時間アクセスがないユーザーのバケット状態を自動削除するため

#### 11.5.10 初回アクセス

Redis キーが存在しない場合：

```
tokens = capacity
last_refill = now
```

で初期化する。

#### 11.5.11 同時リクエスト

複数のリクエストが同時に到着した場合でも、**トークン数（tokens）の計算・消費・保存を原子的に処理し、整合性を維持する必要がある**。

そのため、本システムでは **Redis Lua Script を使用してレート制限判定を実装する**。

Lua Script により以下の処理を **1回の原子的操作（Atomic Operation）** として実行する。

1. Redis から現在の `tokens` と `last_refill` を取得
2. 現在時刻 `now` を取得
3. トークン補充量を計算
   ```
   tokens += (now - last_refill) * refill_rate
   tokens = min(tokens, capacity)
   ```
4. `tokens >= token_cost` を判定
5. 許可する場合はトークンを消費：`tokens = tokens - token_cost`
6. `last_refill = now` に更新
7. 更新後の `tokens` と `last_refill` を Redis に保存
8. Redis Key の TTL を再設定
9. リクエスト許可 / 拒否を返却

**Lua Script 入出力仕様**

Redis Key：`ratelimit:user:{userID}`

入力パラメータ：

| 項目 | 内容 |
|------|------|
| KEYS[1] | `ratelimit:user:{userID}` |
| ARGV[1] | 現在時刻 now |
| ARGV[2] | capacity |
| ARGV[3] | refill_rate |
| ARGV[4] | token_cost |
| ARGV[5] | ttl_seconds |

Redis保存データ：

| フィールド | 型 | 説明 |
|------------|-----|------|
| tokens | float | 現在のトークン数 |
| last_refill | timestamp | 最後に補充した時刻 |

**許可時の処理**（`tokens >= token_cost`）：

```
tokens = tokens - token_cost
last_refill = now
```

更新後の状態を Redis に保存し、Lua Script は `allowed = 1` を返す。

**拒否時の処理**（`tokens < token_cost`）：

トークンは消費しない。

```
retry_after = ceil((token_cost - tokens) / refill_rate)
```

レスポンスヘッダー：`Retry-After: retry_after`

Lua Script は `allowed = 0` を返す。

**実装制約：**

- レート制限判定は Controller 層のミドルウェアで実行する。
- `GET → 計算 → SET` のような分割実装は禁止し、必ず Lua Script 1回の実行で完結させる。
- `tokens` は小数値を扱うため、Lua Script 内でも float 計算を行う。
- 時刻単位（秒 / ミリ秒）はアプリケーションと Lua Script で統一する。

#### 11.5.12 Redisキー例

```
ratelimit:user:1
ratelimit:user:42
ratelimit:user:123
```

#### 11.5.13 実装制約

レート制限は **Controller 層のミドルウェア** で実行する。

処理順序：

```
RateLimit Middleware
        ↓
Auth Middleware
        ↓
Controller
        ↓
Usecase
```
