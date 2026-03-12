# 仕様書（cf10f5e4…）矛盾点の修正

メイン仕様書 PDF で指摘した矛盾を解消するための**差し替え用テキスト**です。Notion や PDF の該当箇所に反映してください。

---

## 1. 7.4 POST /api/game/cheat の前提条件・エラー（修正版）

### 1.1 前提条件表の修正

**誤り**: 「セッション存在」→ game_not_started（意味が逆）

**修正後**:

```
前提条件（修正）

条件                    エラー
--------------------------------------------------
セッション不存在         session_not_found
セット進行中             game_not_started
DEALERモード             cheat_not_allowed
cheat未使用              cheat_already_used
dealer_remaining存在     cheat_not_available
ver一致                  version_conflict
```

### 1.2 エラー表の修正（select と統一）

**誤り**: セッションなし時に 400 game_not_started のみ記載 → select は 404 session_not_found で不統一

**修正後**:

```
エラー（修正）

code                 HTTP  説明
--------------------------------------------------------
session_not_found    404   ゲームセッションが存在しない
game_not_started     400   セッションは存在するが、status が NOT_STARTED / FINISHED のため cheat 不可
cheat_not_allowed    400   PLAYERモード
cheat_already_used   400   2回目
cheat_not_available  400   残カードなし
version_conflict     409   ver不一致
```

※ セッションが存在しない場合は **404 session_not_found** とし、7.3 select と統一する。

---

## 2. 7.1 エラーコード一覧（欠落している場合の追記用）

7.1 の表が PDF で見当たらない場合、以下を 7.2 の前に挿入してください。

```
7.1 エラーコード一覧

code                  HTTP  説明
------------------------------------------------------------------
invalid_json          400   JSON形式不正
invalid_input         400   型・範囲不正（card が 1〜13 以外、小数、文字列など）
invalid_game_state    400   状態不正（IN_PROGRESS で start、FINISHED 以外で mode 変更 など）
game_not_started      400   セッション未開始（NOT_STARTED / FINISHED で select を呼んだ 等）
game_not_finished     400   セット終了していない（FINISHED 以外で mode 変更 等）
game_already_started  400   セット進行中（IN_PROGRESS 中に start 等）
cheat_not_available   400   残カードなし（dealer_remaining が空）
cheat_already_used    400   1 セット内で 2 回 cheat しようとした
cheat_not_allowed     400   PLAYERモードで cheat を呼んだ
invalid_mode          400   mode が PLAYER / DEALER 以外
unauthorized          401   session_id Cookie なし / セッション無効 / 期限切れ
forbidden             403   権限不足（他ユーザーのセッション参照など）
session_not_found     404   セッション不存在（select / cheat 時に対象セッションが存在しない）
version_conflict      409   ver不一致（楽観ロック衝突）
too_many_requests     429   レート制限超過
internal_error        500   Redis 接続エラー・予期しない例外
```

---

## 3. 6.2 通常ゲームフローの追記（session_id の取得・保存）

**現状**: 「レスポンスで ver を受け取り保存」のみで、session_id に触れていない。

**追記する 1 文**（6.2 の #1 の説明の直後などに追加）:

```
1  POST /api/game/start  新規セッション作成（ver=1）。body なし。レスポンスの session_id と ver を
                          ローカルに保存する。以降の POST /api/game/select の Request Body に session_id と ver を含める。
```

または、6.2 の冒頭に次の注意を追加:

```
【クライアントの保持する値】
・session_id: POST /api/game/start または GET /api/game/status の data.session_id から取得し、POST /api/game/select の Request Body に使用する。
・ver: 各更新系 API のレスポンスで返る ver で都度更新する。失った場合は GET /api/game/status で再取得する。
```

---

## 4. 7.6 GET /api/game/status の補足（セッションなし時）

**補足**: セッションが存在しない場合のレスポンスで、`session_id` をどうするか明示する場合の例。

```
セッションなし（NOT_STARTED）のとき:
  data.session_id は 0 とする（または省略可能とする）。
  status: "NOT_STARTED", mode: "PLAYER", player_wins: 0, dealer_wins: 0, ver: 0, history: [] を返す。
```

---

以上を仕様書に反映すると、指摘した矛盾は解消されます。
