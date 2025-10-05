# apcdeploy リファクタリング計画 - Google Go Best Practices 準拠

**作成日**: 2025-10-05
**対象**: Google Go Style Guide Best Practices 準拠

---

## 実装の全体方針

### 目的

このリファクタリング計画は、Google の Go ベストプラクティスレビューで指摘された改善点を、TDDの原則に則って実装することを目的とします。

### 開発手法

**TDD (Test-Driven Development) による実装**

**基本原則**:

1. **Red**: まず失敗するテストを書く（または既存テストを修正）
2. **Green**: テストが通る最小限のコードを書く
3. **Refactor**: コードをリファクタリングする

### MUSTルール

**1. 各Epicの完了時に `make ci` が必ずパスすること**

`make ci` は以下を実行します：

- `make test` - すべてのテストが通過
- `make lint` - リンターチェックが通過
- `make modernize` - コードの最新化チェックが通過
- `make fmt` - フォーマットチェックが通過

**2. 適切な粒度でGitコミット・プッシュを行うこと**

コミットメッセージの形式（英語）：

- Epic完了時: `refactor: complete Refactoring Epic N - [Epic description]`
- リファクタリング時: `refactor: [change description]`
- テスト修正時: `test: update tests for [target]`

---

## Epic R1: コードの重複排除（優先度：高）

**目的**: テストヘルパー関数の重複を排除し、標準ライブラリを使用する

### TDD実装順序

既存のテストが壊れないことを確認しながら、段階的にリファクタリングします。

### タスクチェックリスト

- [x] **R1.1 重複箇所の特定と分析**
  - [x] 重複している`contains`関数の使用箇所を確認
    - [x] `/Users/koh/github/apcdeploy/internal/run/deploy_test.go:829`
    - [x] `/Users/koh/github/apcdeploy/internal/init/initializer_test.go`
    - [x] `/Users/koh/github/apcdeploy/internal/aws/testutil_test.go:4`
  - [x] 各ファイルでの使用パターンを確認
  - [x] 影響範囲の調査

- [x] **R1.2 既存テストの確認（Before Refactoring）**
  - [x] `go test ./internal/run/...` を実行して現在のテスト状態を確認
  - [x] `go test ./internal/init/...` を実行
  - [x] `go test ./internal/aws/...` を実行
  - [x] すべてのテストが通過していることを確認

- [x] **R1.3 `internal/run/deploy_test.go` のリファクタリング**
  - [x] テスト修正（`contains`関数削除前の準備）
    - [x] `contains`関数の使用箇所を特定
    - [x] `strings.Contains`に置き換えるテストケースを確認
  - [x] 実装修正
    - [x] `import "strings"` を追加
    - [x] `contains(s, substr)` を `strings.Contains(s, substr)` に置き換え
    - [x] カスタム`contains`関数を削除
  - [x] テスト実行
    - [x] `go test ./internal/run/... -v`
    - [x] すべてのテストがパスすることを確認
  - [x] リファクタリング
    - [x] コードの可読性を確認

- [x] **R1.4 `internal/init/initializer_test.go` のリファクタリング**
  - [x] テスト修正
    - [x] `contains`関数の使用箇所を特定
    - [x] `strings.Contains`に置き換える
  - [x] 実装修正
    - [x] `import "strings"` を追加
    - [x] `contains(s, substr)` を `strings.Contains(s, substr)` に置き換え
    - [x] カスタム`contains`関数を削除
  - [x] テスト実行
    - [x] `go test ./internal/init/... -v`
    - [x] すべてのテストがパスすることを確認
  - [x] リファクタリング

- [x] **R1.5 `internal/aws/testutil_test.go` のリファクタリング**
  - [x] ファイル全体の削除を検討
    - [x] このファイルは`contains`関数のみを含む
    - [x] 削除可能かどうか確認
  - [x] `internal/aws/resolver_test.go` の修正
    - [x] `contains`関数の使用箇所を特定
    - [x] `strings.Contains`に置き換える
    - [x] `import "strings"` を追加
  - [x] `internal/aws/errors_test.go` の修正（必要な場合）
    - [x] 同様に`strings.Contains`に置き換え
  - [x] `internal/aws/testutil_test.go` を削除
  - [x] テスト実行
    - [x] `go test ./internal/aws/... -v`
    - [x] すべてのテストがパスすることを確認
  - [x] リファクタリング

- [x] **R1.6 Epic R1 完了確認（MUST）**
  - [x] 全テスト実行
    - [x] `make test` - すべてのテストがパス
  - [x] テストカバレッジ確認
    - [x] `go test -cover ./...`
    - [x] カバレッジが低下していないことを確認
  - [x] `make ci` 実行
    - [x] `make test` - すべてのテストがパス
    - [x] `make lint` - リンターエラーを修正
    - [x] `make modernize` - 最新化の問題を修正
    - [x] `make fmt` - フォーマット適用
  - [x] すべてのチェックがパスするまで修正を繰り返す
  - [x] 実装計画のチェックリストを更新
  - [x] Gitコミット・プッシュ
    - [x] `git add .`
    - [x] `git commit -m "refactor: replace custom contains with strings.Contains"`
    - [x] `git push origin main`
    - [x] チェックリスト更新をコミット・プッシュ

---

## Epic R2: 未使用コードの削除（優先度：高）

**目的**: 未使用の関数を削除する

### TDD実装順序

未使用コードを安全に削除し、テストの整合性を保ちます。

### タスクチェックリスト

- [x] **R2.1 未使用コードの調査**
  - [x] `formatUserFriendlyError`関数の使用箇所を検索
    - [x] `grep -r "formatUserFriendlyError" .`
  - [x] 関数が本当に未使用であることを確認
  - [x] 他の未使用関数がないか確認

- [x] **R2.2 `internal/aws/errors_test.go` の確認**
  - [x] `formatUserFriendlyError`に関連するテストの有無を確認
  - [x] 関連テストがある場合、削除対象として記録
  - [x] テストが存在しない場合、そのまま進む

- [x] **R2.3 未使用関数の削除**
  - [x] `internal/aws/errors.go` から`formatUserFriendlyError`を削除
  - [x] 関連するテストコードを削除（存在する場合）
  - [x] テスト実行
    - [x] `go test ./internal/aws/... -v`
    - [x] すべてのテストがパスすることを確認
  - [x] ビルド確認
    - [x] `go build ./...`
    - [x] ビルドエラーがないことを確認

- [x] **R2.4 Epic R2 完了確認（MUST）**
  - [x] 全テスト実行
    - [x] `make test` - すべてのテストがパス
  - [x] テストカバレッジ確認
    - [x] `go test -cover ./internal/aws/...`
    - [x] カバレッジが適切であることを確認
  - [x] `make ci` 実行
    - [x] `make test` - すべてのテストがパス
    - [x] `make lint` - リンターエラーを修正
    - [x] `make modernize` - 最新化の問題を修正
    - [x] `make fmt` - フォーマット適用
  - [x] すべてのチェックがパスするまで修正を繰り返す
  - [x] 実装計画のチェックリストを更新
  - [x] Gitコミット・プッシュ
    - [x] `git add .`
    - [x] `git commit -m "refactor: remove unused formatUserFriendlyError function"`
    - [x] `git push origin main`
    - [x] チェックリスト更新をコミット・プッシュ

---

## Epic R3: 関数コメントの追加（優先度：中）

**目的**: 主要な非公開ヘルパー関数にgodocコメントを追加する

### TDD実装順序

ドキュメント追加は実装の変更を伴わないため、既存テストの動作確認が中心です。

### タスクチェックリスト

- [x] **R3.1 コメント不足関数のリスト作成**
  - [x] コメントが不足している関数を特定
    - [x] `internal/run/deploy.go:43` - `loadConfiguration`
    - [x] `internal/diff/calculator.go:27` - `calculate`
    - [x] `internal/diff/calculator.go:157` - `formatDiffs`
  - [x] 各関数の責務と入出力を分析

- [x] **R3.2 `internal/run/deploy.go` のドキュメント追加**
  - [x] 既存テストの確認
    - [x] `go test ./internal/run/... -v`
    - [x] テストが通過することを確認
  - [x] `loadConfiguration`関数にコメント追加

```go
// loadConfiguration loads the configuration file and data file.
// It returns the parsed Config, the raw data file content, and any error encountered.
// The data file path in the returned Config is resolved to an absolute path.
//
// Parameters:
//   - configPath: Path to the apcdeploy.yml configuration file
//
// Returns:
//   - *config.Config: Parsed configuration with resolved paths
//   - []byte: Raw content of the data file
//   - error: Any error during loading or parsing
func loadConfiguration(configPath string) (*config.Config, []byte, error) {
```

  - [x] テスト実行（変更なしの確認）
    - [x] `go test ./internal/run/... -v`
  - [x] godocフォーマット確認
    - [x] `go doc -all internal/run`

- [x] **R3.3 `internal/diff/calculator.go` の `calculate` 関数ドキュメント追加**
  - [x] 既存テストの確認
    - [x] `go test ./internal/diff/... -v`
  - [x] `calculate`関数にコメント追加

```go
// calculate computes the diff between remote and local configuration.
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields
// before comparing to avoid false positives from auto-generated timestamps.
//
// The function normalizes both contents based on file type (JSON/YAML/text)
// to ensure consistent formatting before comparison.
//
// Parameters:
//   - remoteContent: The deployed configuration content
//   - localContent: The local configuration content
//   - fileName: Name of the local file (used to determine file type)
//   - profileType: AWS AppConfig profile type (e.g., "AWS.AppConfig.FeatureFlags")
//
// Returns:
//   - *Result: Diff result containing normalized contents and unified diff
//   - error: Any error during normalization or diff calculation
func calculate(remoteContent, localContent, fileName, profileType string) (*Result, error) {
```

  - [x] テスト実行
    - [x] `go test ./internal/diff/... -v`
  - [x] godocフォーマット確認

- [x] **R3.4 `internal/diff/calculator.go` の `formatDiffs` 関数ドキュメント追加**
  - [x] `formatDiffs`関数にコメント追加

```go
// formatDiffs converts line-based diffs to a simple diff format.
// It processes each diff chunk and formats lines with prefixes:
//   - "+" for added lines
//   - "-" for deleted lines
//   - " " for context lines (unchanged)
//
// Empty lines are skipped to produce a cleaner output.
//
// Parameters:
//   - diffs: Slice of diff chunks from go-diff library
//
// Returns:
//   - string: Formatted diff output
func formatDiffs(diffs []diffmatchpatch.Diff) string {
```

  - [x] テスト実行
    - [x] `go test ./internal/diff/... -v`
  - [x] godocフォーマット確認

- [x] **R3.5 その他の関数のドキュメント確認と追加**
  - [x] 複雑な非公開関数を追加で探す
    - [x] `grep -n "^func [a-z]" internal/**/*.go`
  - [x] 必要に応じてコメントを追加
  - [x] 全テスト実行

- [x] **R3.6 Epic R3 完了確認（MUST）**
  - [x] godoc生成確認
    - [x] `go doc -all internal/run`
    - [x] `go doc -all internal/diff`
    - [x] コメントが正しく表示されることを確認
  - [x] 全テスト実行
    - [x] `make test` - すべてのテストがパス
  - [x] テストカバレッジ確認
    - [x] `go test -cover ./...`
  - [x] `make ci` 実行
    - [x] `make test` - すべてのテストがパス
    - [x] `make lint` - リンターエラーを修正
    - [x] `make modernize` - 最新化の問題を修正
    - [x] `make fmt` - フォーマット適用
  - [x] すべてのチェックがパスするまで修正を繰り返す
  - [x] 実装計画のチェックリストを更新
  - [x] Gitコミット・プッシュ
    - [x] `git add .`
    - [x] `git commit -m "docs: add godoc comments to helper functions"`
    - [x] `git push origin main`
    - [x] チェックリスト更新をコミット・プッシュ

---

## Epic R4: マジックナンバー/文字列の定数化（優先度：中）

**目的**: 繰り返し使用される文字列リテラルを定数として定義する

### TDD実装順序

定数化により既存のテストが壊れないことを確認しながら進めます。

### タスクチェックリスト

- [x] **R4.1 マジックストリングの特定**
  - [x] プロファイルタイプ関連
    - [x] `"AWS.AppConfig.FeatureFlags"` の使用箇所を検索
    - [x] `"AWS.Freeform"` の使用箇所を検索
  - [x] デプロイ戦略プレフィックス
    - [x] `"AppConfig."` の使用箇所を検索（`internal/aws/resolver.go:134`）
  - [x] その他のマジックストリング
    - [x] コンテンツタイプ関連の文字列

- [x] **R4.2 定数定義の設計**
  - [x] `internal/config/constants.go` に定義する定数を設計

```go
package config

// Profile types
const (
    ProfileTypeFeatureFlags = "AWS.AppConfig.FeatureFlags"
    ProfileTypeFreeform     = "AWS.Freeform"
)

// Deployment strategy prefixes
const (
    StrategyPrefixPredefined = "AppConfig."
)

// Content types
const (
    ContentTypeJSON = "application/json"
    ContentTypeYAML = "application/x-yaml"
    ContentTypeText = "text/plain"
)
```

  - [x] パッケージ構成の確認

- [x] **R4.3 `internal/config/constants.go` の作成（TDD）**
  - [x] `internal/config/constants_test.go` 作成
    - [x] 定数の値が正しいことを検証するテスト

```go
func TestConstants(t *testing.T) {
    tests := []struct {
        name     string
        constant string
        expected string
    }{
        {"ProfileTypeFeatureFlags", ProfileTypeFeatureFlags, "AWS.AppConfig.FeatureFlags"},
        {"ProfileTypeFreeform", ProfileTypeFreeform, "AWS.Freeform"},
        {"StrategyPrefixPredefined", StrategyPrefixPredefined, "AppConfig."},
        {"ContentTypeJSON", ContentTypeJSON, "application/json"},
        {"ContentTypeYAML", ContentTypeYAML, "application/x-yaml"},
        {"ContentTypeText", ContentTypeText, "text/plain"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.constant != tt.expected {
                t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
            }
        })
    }
}
```

  - [x] テスト実行（失敗することを確認）
    - [x] `go test ./internal/config/... -v`
  - [x] `internal/config/constants.go` 作成
    - [x] 定数を定義
  - [x] テスト実行（成功することを確認）
    - [x] `go test ./internal/config/... -v`

- [x] **R4.4 既存コードでの定数使用（段階的リファクタリング）**
  - [x] **Phase 1: `internal/run/deploy.go` の修正**
    - [x] 既存テストの確認
      - [x] `go test ./internal/run/... -v`
    - [x] マジックストリングを定数に置き換え
      - [x] `"AWS.AppConfig.FeatureFlags"` → `config.ProfileTypeFeatureFlags`
      - [x] `"application/json"` → `config.ContentTypeJSON`
      - [x] `"application/x-yaml"` → `config.ContentTypeYAML`
      - [x] `"text/plain"` → `config.ContentTypeText`
    - [x] テスト実行
      - [x] `go test ./internal/run/... -v`
    - [x] リファクタリング
  - [x] **Phase 2: `internal/diff/calculator.go` の修正**
    - [x] 既存テストの確認
      - [x] `go test ./internal/diff/... -v`
    - [x] マジックストリングを定数に置き換え
      - [x] `"AWS.AppConfig.FeatureFlags"` → `config.ProfileTypeFeatureFlags`
    - [x] テスト実行
      - [x] `go test ./internal/diff/... -v`
    - [x] リファクタリング
  - [x] **Phase 3: `internal/aws/resolver.go` の修正**
    - [x] 既存テストの確認
      - [x] `go test ./internal/aws/... -v`
    - [x] マジックストリングを定数に置き換え
      - [x] `"AppConfig."` → `config.StrategyPrefixPredefined`
      - [x] `strings.HasPrefix` を使用してコードを改善
    - [x] テスト実行
      - [x] `go test ./internal/aws/... -v`
    - [x] リファクタリング
  - [x] **Phase 4: テストファイルの修正**
    - [x] `internal/run/deploy_test.go` の修正
      - [x] マジックストリングを定数に置き換え
      - [x] テスト実行
    - [x] `internal/diff/calculator_test.go` の修正
      - [x] 同様に定数に置き換え
      - [x] テスト実行
    - [x] その他のテストファイルを修正
      - [x] `internal/config/generator_test.go`
      - [x] `internal/aws/resolver_test.go`
      - [x] `internal/init/initializer_test.go`

- [x] **R4.5 全ファイルの検索と置き換え確認**
  - [x] 残っているマジックストリングを検索
    - [x] `grep -r "AWS.AppConfig.FeatureFlags" --include="*.go" .`
    - [x] `grep -r "AWS.Freeform" --include="*.go" .`
    - [x] `grep -r '"AppConfig\."' --include="*.go" .`
  - [x] 見つかった箇所をすべて定数に置き換え
    - [x] AWS API mockレスポンスは意図的に保持（データ形式の明示性のため）

- [x] **R4.6 Epic R4 完了確認（MUST）**
  - [x] 全テスト実行
    - [x] `make test` - すべてのテストがパス
  - [x] テストカバレッジ確認
    - [x] `go test -cover ./...`
  - [x] `make ci` 実行
    - [x] `make test` - すべてのテストがパス
    - [x] `make lint` - リンターエラーを修正
    - [x] `make modernize` - 最新化の問題を修正
    - [x] `make fmt` - フォーマット適用
  - [x] すべてのチェックがパスするまで修正を繰り返す
  - [x] 実装計画のチェックリストを更新
  - [x] Gitコミット・プッシュ
    - [x] `git add .`
    - [x] `git commit -m "refactor: replace magic strings with constants"`
    - [x] `git push origin main`
    - [x] チェックリスト更新をコミット・プッシュ

---

## Epic R5: エラーメッセージの小文字化（優先度：低）

**目的**: エラーメッセージを小文字で始めるGoの慣習に従う

### TDD実装順序

エラーメッセージの変更に伴い、関連するテストも更新します。

### タスクチェックリスト

- [ ] **R5.1 大文字で始まるエラーメッセージの特定**
  - [ ] エラーメッセージを検索
    - [ ] `grep -r 'errors.New\|fmt.Errorf' --include="*.go" . | grep -E '["](A-Z)'`
  - [ ] 各エラーメッセージを分類
    - [ ] 固有名詞やアクロニムで始まる（修正不要）
    - [ ] 一般的な単語で始まる（修正対象）
  - [ ] 修正対象のリストを作成

- [ ] **R5.2 主要なエラーメッセージの修正（TDD）**
  - [ ] **`internal/aws/errors.go` の修正**
    - [ ] 既存テストの確認
      - [ ] `go test ./internal/aws/... -v`
    - [ ] テスト修正（エラーメッセージの期待値を更新）
      - [ ] `internal/aws/errors_test.go` のテストケースを更新
      - [ ] 大文字で始まるメッセージを小文字に変更
    - [ ] テスト実行（失敗することを確認）
      - [ ] `go test ./internal/aws/... -v`
    - [ ] 実装修正
      - [ ] `errors.go:154` の修正例:

```go
// Before
"Resource not found during %s operation. Please verify..."

// After
"resource not found during %s operation, please verify..."
```

      - [ ] その他のエラーメッセージを修正
      - [ ] 固有名詞（AWS, AppConfig等）で始まるものは除外
    - [ ] テスト実行（成功することを確認）
      - [ ] `go test ./internal/aws/... -v`
    - [ ] リファクタリング
  - [ ] **`internal/config/loader.go` の修正**
    - [ ] テスト修正
      - [ ] `internal/config/loader_test.go` の期待値を更新
    - [ ] テスト実行（失敗を確認）
    - [ ] 実装修正
      - [ ] エラーメッセージを小文字に変更
    - [ ] テスト実行（成功を確認）
  - [ ] **`internal/run/deploy.go` の修正**
    - [ ] テスト修正
      - [ ] `internal/run/deploy_test.go` の期待値を更新
    - [ ] テスト実行（失敗を確認）
    - [ ] 実装修正
    - [ ] テスト実行（成功を確認）
  - [ ] **その他のファイルの修正**
    - [ ] `cmd/` 配下のコマンドファイル
    - [ ] `internal/diff/` 配下のファイル
    - [ ] `internal/status/` 配下のファイル
    - [ ] 各ファイルごとに Test → Implementation のサイクルを実行

- [ ] **R5.3 固有名詞とアクロニムの確認**
  - [ ] 以下で始まるエラーメッセージは大文字のまま保持
    - [ ] "AWS"
    - [ ] "AppConfig"
    - [ ] "IAM"
    - [ ] "HTTP"
    - [ ] "JSON"
    - [ ] "YAML"
  - [ ] 該当するエラーメッセージをリストアップ
  - [ ] 正しく大文字が保持されていることを確認

- [ ] **R5.4 エラーメッセージのチェーン確認**
  - [ ] `fmt.Errorf` with `%w` の使用箇所を確認
  - [ ] ラップされたエラーメッセージの可読性を確認
  - [ ] 必要に応じて調整

- [ ] **R5.5 Epic R5 完了確認（MUST）**
  - [ ] 全テスト実行
    - [ ] `make test` - すべてのテストがパス
  - [ ] テストカバレッジ確認
    - [ ] `go test -cover ./...`
    - [ ] カバレッジが維持されていることを確認
  - [ ] エラーメッセージの一貫性チェック
    - [ ] `grep -r 'errors.New\|fmt.Errorf' --include="*.go" . | grep -v "test"`
    - [ ] 小文字で始まることを確認（固有名詞除く）
  - [ ] `make ci` 実行
    - [ ] `make test` - すべてのテストがパス
    - [ ] `make lint` - リンターエラーを修正
    - [ ] `make modernize` - 最新化の問題を修正
    - [ ] `make fmt` - フォーマット適用
  - [ ] すべてのチェックがパスするまで修正を繰り返す
  - [ ] 実装計画のチェックリストを更新
  - [ ] Gitコミット・プッシュ
    - [ ] `git add .`
    - [ ] `git commit -m "refactor: lowercase error messages per Go conventions"`
    - [ ] `git push origin main`
    - [ ] チェックリスト更新をコミット・プッシュ

---

## 完成の定義（Definition of Done）

各Epicが完了とみなされる基準:

- [ ] すべてのタスクが完了している
- [ ] すべてのユニットテストが通過している
- [ ] テストカバレッジが維持または向上している
- [ ] **`make ci` がパスする**（MUST）
  - [ ] `make test` - すべてのテストがパス
  - [ ] `make lint` - リンターチェックがパス
  - [ ] `make modernize` - 最新化チェックがパス
  - [ ] `make fmt` - フォーマットチェックがパス
- [ ] コードレビューが完了している（チーム開発の場合）
- [ ] リファクタリングのチェックリストが更新されている
- [ ] 手動での動作確認が完了している（必要に応じて）

### TDDサイクルの確認

各リファクタリングにおいて以下のサイクルが守られていること:

1. ✅ **Red**: 既存テストまたは新規テストが失敗する状態を作る
2. ✅ **Green**: テストを通す最小限の実装
3. ✅ **Refactor**: コードをリファクタリング（テストは維持）

### MUSTルールの再確認

**`make ci` が各Epic完了時に必ずパスすること**

これは絶対に守られる必要があります。各Epicの最後のタスクとして `make ci` の実行と修正が含まれています。

---

## 優先度の説明

- **高**: コードの重複やバグにつながる可能性がある（Epic R1, R2）
- **中**: 保守性と可読性の向上（Epic R3, R4）
- **低**: スタイルガイドへの準拠（Epic R5）

---

## 参考資料

- [Google Go Style Guide - Best Practices](https://google.github.io/styleguide/go/best-practices.html)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
