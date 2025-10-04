# コードリファクタリング実装計画書

## 📋 概要

本ドキュメントは、apcdeploy プロジェクトのコード品質向上のためのリファクタリング計画を定義します。

**作成日**: 2025-10-04
**対象バージョン**: 現行main branch

---

## 🎯 目標

1. コードの重複を排除し、DRY原則を徹底する
2. 保守性とテスタビリティを向上させる
3. パッケージ構造を最適化する
4. 既存のテストカバレッジを維持する（90%以上）

---

## 📊 タスク一覧

### Phase 1: 共通インターフェースの統合（優先度: 高）

#### Task 1.1: 共通Reporterパッケージの作成
- [x] `internal/reporter/reporter.go` を作成
- [x] `ProgressReporter` インターフェースを定義
- [x] パッケージドキュメントを追加

**成果物**:
```go
// internal/reporter/reporter.go
package reporter

// ProgressReporter defines the interface for reporting progress during operations
type ProgressReporter interface {
    Progress(message string)
    Success(message string)
    Warning(message string)
}
```

**影響範囲**: 新規ファイル作成のみ

---

#### Task 1.2: 既存パッケージのReporterインターフェース削除
- [x] `internal/deploy/reporter.go` の削除
- [x] `internal/diff/reporter.go` の削除
- [x] `internal/status/reporter.go` の削除
- [x] `internal/init/types.go` から `ProgressReporter` インターフェース削除

**注意事項**: インターフェース定義のみ削除、他の型定義は保持

**影響範囲**: 4ファイル

---

#### Task 1.3: 各パッケージでの共通Reporterインポート
- [x] `internal/deploy/executor.go` のimportを更新
- [x] `internal/diff/executor.go` のimportを更新
- [x] `internal/status/executor.go` のimportを更新
- [x] `internal/init/initializer.go` のimportを更新
- [x] `internal/init/types.go` のimportを更新

**変更例**:
```go
import (
    // ... 既存のimport
    "github.com/koh-sh/apcdeploy/internal/reporter"
)

// reporter.ProgressReporter として使用
```

**影響範囲**: 5ファイル

---

#### Task 1.4: cmdパッケージのReporter実装を更新
- [x] `cmd/reporter.go` のimportを更新
- [x] インターフェース参照を `reporter.ProgressReporter` に変更

**影響範囲**: 1ファイル

---

#### Task 1.5: Phase 1 完了チェック（必須）
- [x] `make ci` を実行してパスすることを確認
- [x] `make cov` でカバレッジを確認、低い箇所があれば改善
- [x] このドキュメントのPhase 1チェックリストを全て更新
- [x] 変更をコミット: `git add . && git commit -m "refactor: consolidate ProgressReporter interface into common package"`
- [x] リモートにプッシュ: `git push`

---

### Phase 2: テスト用モックの共通化（優先度: 高）

#### Task 2.1: 共通テストヘルパーパッケージの作成
- [x] `internal/reporter/testing/mock.go` を作成
- [x] `MockReporter` 構造体を実装
- [x] テストヘルパー関数を追加（必要に応じて）

**成果物**:
```go
// internal/reporter/testing/mock.go
package testing

import "github.com/koh-sh/apcdeploy/internal/reporter"

// MockReporter is a test implementation of ProgressReporter
type MockReporter struct {
    Messages []string
}

func (m *MockReporter) Progress(message string) {
    m.Messages = append(m.Messages, "progress: "+message)
}

func (m *MockReporter) Success(message string) {
    m.Messages = append(m.Messages, "success: "+message)
}

func (m *MockReporter) Warning(message string) {
    m.Messages = append(m.Messages, "warning: "+message)
}

// HasMessage checks if the reporter received a message containing the given text
func (m *MockReporter) HasMessage(text string) bool {
    for _, msg := range m.Messages {
        if strings.Contains(msg, text) {
            return true
        }
    }
    return false
}

// Clear clears all messages
func (m *MockReporter) Clear() {
    m.Messages = nil
}
```

**影響範囲**: 新規ファイル作成

---

#### Task 2.2: テストファイルからmockReporter削除とimport更新
- [x] `internal/deploy/executor_test.go` のmockReporter削除、import追加
- [x] `internal/diff/executor_test.go` のmockReporter削除、import追加
- [x] `internal/status/executor_test.go` のmockReporter削除、import追加
- [x] `internal/init/initializer_test.go` のmockReporter削除、import追加

**変更例**:
```go
import (
    // ... 既存のimport
    reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// 使用箇所
reporter := &reportertest.MockReporter{}
```

**影響範囲**: 4ファイル

---

#### Task 2.3: Phase 2 完了チェック（必須）
- [x] `make ci` を実行してパスすることを確認
- [x] `make cov` でカバレッジを確認、低い箇所があれば改善
- [x] このドキュメントのPhase 2チェックリストを全て更新
- [x] 変更をコミット: `git add . && git commit -m "refactor: consolidate test mock reporter into common testing package"`
- [x] リモートにプッシュ: `git push`

---

### Phase 3: AWS Resolverのリファクタリング（優先度: 中）

#### Task 3.1: 共通resolve関数の設計と実装
- [x] `internal/aws/resolver_common.go` を作成
- [x] ジェネリクスを使用した共通resolve関数を実装
- [x] 既存のResolve*メソッドから共通パターンを抽出

**成果物例**:
```go
// internal/aws/resolver_common.go
package aws

import (
    "context"
    "fmt"
)

// resolveByName resolves a resource by name using a generic approach
func resolveByName[T interface{ GetName() *string; GetId() *string }](
    ctx context.Context,
    items []T,
    name string,
    resourceType string,
) (string, error) {
    var matches []string
    for _, item := range items {
        if item.GetName() != nil && *item.GetName() == name {
            if item.GetId() != nil {
                matches = append(matches, *item.GetId())
            }
        }
    }

    if len(matches) == 0 {
        return "", fmt.Errorf("%s not found: %s", resourceType, name)
    }

    if len(matches) > 1 {
        return "", fmt.Errorf("multiple %s found with name: %s", resourceType, name)
    }

    return matches[0], nil
}
```

**影響範囲**: 新規ファイル作成

---

#### Task 3.2: 既存Resolveメソッドのリファクタリング
- [x] `ResolveApplication` を共通関数使用に書き換え
- [x] `ResolveEnvironment` を共通関数使用に書き換え
- [x] `ResolveDeploymentStrategy` を共通関数使用に書き換え

**注意**: `ResolveConfigurationProfile` は戻り値が異なるため個別実装を維持

**影響範囲**: `internal/aws/resolver.go`

---

#### Task 3.3: Phase 3 完了チェック（必須）
- [x] `make ci` を実行してパスすることを確認
- [x] `make cov` でカバレッジを確認、特に`internal/aws/resolver_common.go`の新規コードを確認
- [x] 既存のResolver関連テストが全て通ることを確認
- [x] このドキュメントのPhase 3チェックリストを全て更新
- [ ] 変更をコミット: `git add . && git commit -m "refactor: extract common resolver logic using generics"`
- [ ] リモートにプッシュ: `git push`

**影響範囲**: `internal/aws/resolver_test.go`, `internal/aws/resolver_common.go`

---

### Phase 4: コマンドレイヤーのクリーンアップ（優先度: 中）

#### Task 4.1: init.goのリファクタリング
- [ ] `createInitializer` ヘルパー関数を追加
- [ ] `createDefaultInitializer` 関数を追加
- [ ] `runInit` 関数のロジックを簡素化
- [ ] テスト用分岐を整理

**成果物**:
```go
// cmd/init.go

func runInit(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    initializer, err := createInitializer(ctx)
    if err != nil {
        return fmt.Errorf("failed to create initializer: %w", err)
    }

    opts := &initPkg.Options{
        Application: initApp,
        Profile:     initProfile,
        Environment: initEnv,
        Region:      initRegion,
        ConfigFile:  initConfig,
        OutputData:  initOutputData,
    }

    result, err := initializer.Run(ctx, opts)
    if err != nil {
        return err
    }

    showNextSteps(result)
    return nil
}

func createInitializer(ctx context.Context) (*initPkg.Initializer, error) {
    if initializerFactory != nil {
        return initializerFactory(ctx, initRegion)
    }
    return createDefaultInitializer(ctx)
}

func createDefaultInitializer(ctx context.Context) (*initPkg.Initializer, error) {
    awsClient, err := awsInternal.NewClient(ctx, initRegion)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
    }

    reporter := &cliReporter{}
    return initPkg.New(awsClient, reporter), nil
}

func showNextSteps(result *initPkg.Result) {
    fmt.Println("\n" + display.Success("Initialization complete!"))
    fmt.Println("\nNext steps:")
    fmt.Println("  1. Review the generated configuration files")
    fmt.Println("  2. Modify the data file as needed")
    fmt.Println("  3. Run 'apcdeploy diff' to preview changes")
    fmt.Println("  4. Run 'apcdeploy deploy' to deploy your configuration")
}
```

**影響範囲**: `cmd/init.go`

---

#### Task 4.2: Phase 4 完了チェック（必須）
- [ ] `make ci` を実行してパスすることを確認
- [ ] `make cov` でカバレッジを確認、`cmd/init.go`のカバレッジを確認
- [ ] 既存のinit関連テストが全て通ることを確認
- [ ] このドキュメントのPhase 4チェックリストを全て更新
- [ ] 変更をコミット: `git add . && git commit -m "refactor: clean up init command with helper functions"`
- [ ] リモートにプッシュ: `git push`

**影響範囲**: `cmd/init.go`, `cmd/init_test.go`

---

### Phase 5: 定数の整理（優先度: 低）

#### Task 5.1: 定数の集約
- [ ] `internal/config/constants.go` を作成
- [ ] 定数を集約
  ```go
  package config

  const (
      // MaxConfigSize is the maximum size for configuration data (2MB)
      MaxConfigSize = 2 * 1024 * 1024

      // ContentTypeJSON represents JSON content type
      ContentTypeJSON = "application/json"

      // ContentTypeYAML represents YAML content type
      ContentTypeYAML = "application/x-yaml"

      // ContentTypeText represents plain text content type
      ContentTypeText = "text/plain"
  )
  ```
- [ ] 各ファイルで定数を使用するように更新

**影響範囲**: `internal/config/`, `internal/deploy/`

---

#### Task 5.2: Phase 5 完了チェック（必須）
- [ ] `make ci` を実行してパスすることを確認
- [ ] `make cov` でカバレッジを確認
- [ ] 定数の移行が正しく行われたことを確認
- [ ] このドキュメントのPhase 5チェックリストを全て更新
- [ ] 変更をコミット: `git add . && git commit -m "refactor: consolidate constants"`
- [ ] リモートにプッシュ: `git push`

---

## 📝 完了基準

以下の条件を全て満たした時点で完了とする：

- [ ] 全てのチェックリストが完了
- [ ] `go test ./...` が全て成功
- [ ] カバレッジが90%以上を維持
- [ ] `go vet ./...` でエラーなし
- [ ] ビルドが成功
- [ ] ドキュメントが更新済み

---
