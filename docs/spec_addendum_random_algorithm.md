# High & Low 実装仕様書 補足：ランダム選択アルゴリズム

本ドキュメントは、メイン仕様書（PDF）の **「remaining からランダム選択」** の詳細を定める補足仕様です。  
select 処理における `player_card` / `dealer_card` の決定方法を実装レベルで一意にします。

---

## 1. 対象箇所

- **POST /api/game/select** の処理フローにおける  
  - `player_card = random(player_remaining)`  
  - `dealer_card = random(dealer_remaining)`（cheat 予約時を除く）

---

## 2. 前提

- **remaining** は 1〜13 の整数のうち、使用済みを除いた集合。  
  空でない限り、いずれの要素も同様に選ばれる必要がある（一様ランダム）。
- **空集合** の場合は select 前にバリデーションで弾く（DRAW5 リセット等で通常は空にならない想定）。

---

## 3. アルゴリズム仕様

### 3.1 入出力

| 項目 | 内容 |
|------|------|
| 入力 | 整数の集合（スライスまたは配列）`remaining`。要素は 1〜13 の重複なし。`len(remaining) >= 1` を前提とする。 |
| 出力 | `remaining` に含まれる整数 1 つ。**一様分布**で 1 つ選ぶ。 |

### 3.2 選び方

- **方法**  
  `remaining` から **一様乱数で 1 要素を 1 つ選ぶ**。
- **実装例（擬似コード）**  
  - `index = random_int(0, len(remaining) - 1)` を一様で生成し、  
  - `card = remaining[index]` を返す。  
  - ここで `random_int(min, max)` は min 以上 max 以下の整数を一様に返すものとする。

### 3.3 乱数ソース

| 項目 | 仕様 |
|------|------|
| 乱数 | 言語標準の **擬似乱数**（例: Go の `math/rand`）でよい。暗号論的乱数は必須としない。 |
| シード | 本番では **プロセス起動時に 1 回だけ** シードする（時刻など）。同一プロセス内で select ごとにシードし直さない。 |
| スレッド安全性 | 複数リクエストから同時に呼ばれる場合、使用する RNG のスレッド安全性に従う（必要なら呼び出し側で排他する）。 |

### 3.4 禁止事項

- remaining の要素の順序に依存した「先頭だけ選ぶ」「最後だけ選ぶ」などの固定選出は禁止。
- 同一 remaining に対して、実質的に偏りが生じる実装（例: ハッシュ値の剰余でそのまま選ぶなど）は禁止。**一様選択**であること。

---

## 4. 実装例（Go）

```go
import "math/rand"

// PickRandomFromRemaining は remaining から一様に 1 つ選んで返す。
// remaining は空でないこと。
func PickRandomFromRemaining(remaining []int) int {
	if len(remaining) == 0 {
		panic("remaining must not be empty")
	}
	return remaining[rand.Intn(len(remaining))]
}
```

- `rand` は `init()` または main で `rand.Seed()` を 1 回だけ行う（Go 1.20 以降は自動シードでも可）。
- 同時実行する場合は `rand.New(source)` でインスタンスを分けるか、呼び出し側でロックする。

---

## 5. テスト

- **同一 remaining** を多数回渡し、各要素の出現回数が統計的に一様に近いことを確認する。
- **remaining が 1 要素** のときは、常にその 1 つが返ることを確認する。

---

## 6. 仕様書との対応

- 用語定義「ラウンド」の「remaining からランダム選択」→ 本補足の「一様に 1 つ選ぶ」で実装する。
- 7.3 select の処理フロー「random(player_remaining)」「random(dealer_remaining)」→ 上記アルゴリズムで実装する。
