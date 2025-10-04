# apcdeploy 実装計画

**作成日**: 2025-10-04
**対象バージョン**: 1.0

---

## 実装の全体方針

### 開発順序

1. **Epic 1**: プロジェクト基盤とコア構造
2. **Epic 2**: AWS連携とリソース解決
3. **Epic 3**: initコマンド実装
4. **Epic 4**: deployコマンド実装
5. **Epic 5**: diffコマンド実装
6. **Epic 6**: statusコマンド実装
7. **Epic 7**: テストとドキュメント

### 技術スタック

- **言語**: Go 1.25+
- **CLI Framework**: [cobra](https://github.com/spf13/cobra)
- **AWS SDK**: [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)
- **YAML処理**: [go-yaml](https://github.com/goccy/go-yaml)
- **差分表示**: [go-diff](https://github.com/sergi/go-diff)
- **テスティング**: Go標準テストパッケージ + テーブル駆動テスト

### 開発手法

**TDD (Test-Driven Development) による実装**

このプロジェクトはTDDに則って開発します。

**基本原則**:
1. **Red**: まず失敗するテストを書く
2. **Green**: テストが通る最小限のコードを書く
3. **Refactor**: コードをリファクタリングする

**重要**: テストは実装の**後**ではなく**前**に書きます。すべてのEpicにおいて、各機能の実装前に対応するテストを作成してください。

### MUSTルール

**1. 各Epicの完了時に `make ci` が必ずパスすること**

`make ci` は以下を実行します：
- `make test` - すべてのテストが通過
- `make lint` - リンターチェックが通過
- `make modernize` - コードの最新化チェックが通過
- `make fmt` - フォーマットチェックが通過

このルールは**絶対に守られる必要があります**。各Epicのタスクに `make ci` の実行と修正が含まれています。

**2. 適切な粒度でGitコミット・プッシュを行うこと**

以下のタイミングで必ずコミット・プッシュを行います：
- 各Epic完了時（必須）
- 大きなタスク（機能単位）完了時（推奨）
- `make ci` がパスした後（必須）

コミットメッセージの形式（英語）：
- Epic完了時: `feat: complete Epic N - [Epic description in English]`
- 機能追加時: `feat: implement [feature name]`
- テスト追加時: `test: add tests for [target]`
- リファクタリング時: `refactor: [change description]`

例：
- `feat: complete Epic 1 - project foundation and core structure`
- `feat: implement config file loader`
- `test: add tests for AWS resource resolver`

---

## Epic 1: プロジェクト基盤とコア構造

**目的**: プロジェクトの基礎となるディレクトリ構造、CLI基盤、設定ファイル処理を実装する

### TDD実装順序

各機能について **Test → Implementation → Refactor** の順で実装します。

### タスクチェックリスト

- [ ] **1.1 プロジェクト初期化**
  - [ ] Go モジュール初期化 (`go.mod`, `go.sum`) - Go 1.25使用
  - [ ] `.gitignore` 作成
  - [ ] ディレクトリ構造作成

```text
apcdeploy/
├── cmd/              # CLIコマンド定義
├── internal/         # 内部パッケージ
│   ├── config/       # 設定ファイル処理
│   ├── aws/          # AWS API操作
│   ├── diff/         # 差分計算
│   └── display/      # 出力フォーマット
├── testdata/         # テストデータ
└── main.go
```

- [ ] **1.2 依存関係インストール**
  - [ ] Cobra のインストールと初期化
  - [ ] AWS SDK for Go v2 のインストール
  - [ ] go-yaml のインストール
  - [ ] go-diff のインストール
  - [ ] テスト用ライブラリ確認（標準 testing パッケージ使用）

- [ ] **1.3 CLI基本構造実装（TDD）**
  - [ ] `cmd/root_test.go` 作成 - ルートコマンドのテスト
  - [ ] `cmd/root.go` 実装 - ルートコマンド定義
  - [ ] `main.go` 作成
  - [ ] バージョン情報コマンドテスト作成
  - [ ] バージョン情報コマンド実装 (`--version`)
  - [ ] グローバルフラグのテスト作成
  - [ ] グローバルフラグ実装
    - [ ] `--config, -c` フラグ
    - [ ] `--region` フラグ
  - [ ] ヘルプメッセージテンプレート

- [ ] **1.4 設定ファイル構造体定義（TDD）**
  - [ ] `internal/config/types_test.go` 作成
    - [ ] 構造体のバリデーションテスト
    - [ ] デフォルト値適用テスト
  - [ ] `internal/config/types.go` 実装
    - [ ] `Config` 構造体（apcdeploy.yml）
    - [ ] `DeploymentConfig` 構造体
    - [ ] 構造体のバリデーションタグ追加
    - [ ] デフォルト値の定義

- [ ] **1.5 設定ファイル読み込み機能（TDD）**
  - [ ] `internal/config/loader_test.go` 作成
    - [ ] 正常系テスト（有効なYAML読み込み）
    - [ ] ファイル不存在エラーテスト
    - [ ] YAML構文エラーテスト
    - [ ] 必須フィールド欠如テスト
    - [ ] デフォルト値適用テスト
    - [ ] パス解決テスト（相対パス→絶対パス）
  - [ ] テストデータ作成 (`testdata/config/`)
  - [ ] `internal/config/loader.go` 実装
    - [ ] YAML読み込み関数
    - [ ] 設定ファイルバリデーション
    - [ ] デフォルト値の適用
    - [ ] パス解決
  - [ ] エラーハンドリング実装
  - [ ] リファクタリング

- [ ] **1.6 設定データファイル処理（TDD）**
  - [ ] `internal/config/data_test.go` 作成
    - [ ] JSON読み込みテスト（正常系・異常系）
    - [ ] YAML読み込みテスト（正常系・異常系）
    - [ ] Text読み込みテスト
    - [ ] ContentType判定テスト
    - [ ] サイズチェックテスト（2MB境界値）
    - [ ] 構文バリデーションテスト
  - [ ] テストデータ作成
    - [ ] 有効なJSON/YAML/Text
    - [ ] 不正なJSON/YAML
    - [ ] サイズ超過データ
  - [ ] `internal/config/data.go` 実装
    - [ ] JSON読み込み
    - [ ] YAML読み込み
    - [ ] Text読み込み
    - [ ] ContentType判定機能
    - [ ] サイズチェック
    - [ ] 構文バリデーション
  - [ ] リファクタリング

- [ ] **1.7 共通ユーティリティ（TDD）**
  - [ ] `internal/display/output_test.go` 作成
    - [ ] 成功メッセージフォーマットテスト
    - [ ] エラーメッセージフォーマットテスト
    - [ ] 警告メッセージフォーマットテスト
    - [ ] 進捗表示テスト
  - [ ] `internal/display/output.go` 実装
    - [ ] 成功メッセージフォーマット（✓）
    - [ ] エラーメッセージフォーマット（✗）
    - [ ] 警告メッセージフォーマット（⚠）
    - [ ] 進捗表示（⏳）
  - [ ] ロギング設定
  - [ ] リファクタリング

- [ ] **1.8 Epic 1 完了確認（MUST）**
  - [ ] `make ci` 実行
    - [ ] `make test` - すべてのテストがパス
    - [ ] `make lint` - リンターエラーを修正
    - [ ] `make modernize` - 最新化の問題を修正
    - [ ] `make fmt` - フォーマット適用
  - [ ] すべてのチェックがパスするまで修正を繰り返す
  - [ ] Gitコミット・プッシュ
    - [ ] `git add .`
    - [ ] `git commit -m "feat: complete Epic 1 - project foundation and core structure"`
    - [ ] `git push origin main`

---

## Epic 2: AWS連携とリソース解決

**目的**: AWS AppConfig APIとの連携機能と、リソース名からIDへの変換機能を実装する

### TDD実装順序

AWS APIのモックを使用してテストファーストで実装します。

### タスクチェックリスト

- [ ] **2.1 AWS SDK初期化（TDD）**
  - [ ] `internal/aws/client_test.go` 作成
    - [ ] AWS Config読み込みテスト
    - [ ] 環境変数からの設定読み込みテスト
    - [ ] リージョン設定テスト
    - [ ] 認証失敗エラーテスト
  - [ ] `internal/aws/client.go` 実装
    - [ ] AWS Config読み込み（認証情報、リージョン）
    - [ ] AppConfig クライアント初期化
    - [ ] リトライポリシー設定
  - [ ] 環境変数サポート実装
    - [ ] `AWS_REGION`
    - [ ] `AWS_PROFILE`
  - [ ] エラーハンドリング実装
  - [ ] リファクタリング

- [ ] **2.2 AWS APIモック基盤**
  - [ ] `internal/aws/mock/` ディレクトリ作成
  - [ ] `internal/aws/mock/appconfig.go` - モックインターフェース定義
  - [ ] テスト用モック実装
    - [ ] ListApplications モック
    - [ ] ListConfigurationProfiles モック
    - [ ] ListEnvironments モック
    - [ ] ListDeploymentStrategies モック
    - [ ] GetConfigurationProfile モック

- [ ] **2.3 Application解決（TDD）**
  - [ ] `internal/aws/resolver_test.go` 作成
    - [ ] Application名前検索テスト（成功ケース）
    - [ ] Application不存在エラーテスト
    - [ ] 複数マッチエラーテスト
    - [ ] API権限エラーテスト
  - [ ] テストフィクスチャ作成
  - [ ] `internal/aws/resolver.go` 実装
    - [ ] `ListApplications` API呼び出し
    - [ ] 名前による検索
    - [ ] Application ID取得
  - [ ] エラーハンドリング実装
    - [ ] Application不存在エラー
    - [ ] 複数マッチエラー
    - [ ] 利用可能なApplication一覧表示
  - [ ] リファクタリング

- [ ] **2.4 Configuration Profile解決（TDD）**
  - [ ] `resolver_test.go` にProfile解決テスト追加
    - [ ] Profile名前検索テスト
    - [ ] Profile情報取得テスト
    - [ ] Type判定テスト（Feature Flags / Freeform）
    - [ ] Profile不存在エラーテスト
    - [ ] 複数マッチエラーテスト
  - [ ] `resolver.go` に実装
    - [ ] `ListConfigurationProfiles` API呼び出し
    - [ ] 名前による検索
    - [ ] Profile ID取得
    - [ ] `GetConfigurationProfile` で詳細取得
      - [ ] Type（Feature Flags / Freeform）
      - [ ] LocationUri
      - [ ] Validators
  - [ ] エラーハンドリング実装
  - [ ] リファクタリング

- [ ] **2.5 Environment解決（TDD）**
  - [ ] `resolver_test.go` にEnvironment解決テスト追加
    - [ ] Environment名前検索テスト
    - [ ] Environment不存在エラーテスト
    - [ ] 複数マッチエラーテスト
  - [ ] `resolver.go` に実装
    - [ ] `ListEnvironments` API呼び出し
    - [ ] 名前による検索
    - [ ] Environment ID取得
  - [ ] エラーハンドリング実装
  - [ ] リファクタリング

- [ ] **2.6 Deployment Strategy解決（TDD）**
  - [ ] `resolver_test.go` にStrategy解決テスト追加
    - [ ] 名前検索テスト（完全一致、大文字小文字区別）
    - [ ] デフォルト戦略テスト（`AppConfig.AllAtOnce`）
    - [ ] Strategy不存在エラーテスト
  - [ ] `resolver.go` に実装
    - [ ] `ListDeploymentStrategies` API呼び出し
    - [ ] 名前による検索
    - [ ] Strategy ID取得
    - [ ] デフォルト戦略サポート
  - [ ] エラーハンドリング実装
    - [ ] 利用可能な戦略一覧表示
  - [ ] リファクタリング

- [ ] **2.7 エラーハンドリング基盤（TDD）**
  - [ ] `internal/aws/errors_test.go` 作成
    - [ ] AWS APIエラーラップテスト
    - [ ] IAM権限エラー判定テスト
    - [ ] エラーメッセージ変換テスト
    - [ ] リトライロジックテスト
  - [ ] `internal/aws/errors.go` 実装
    - [ ] AWS APIエラーのラップ
    - [ ] ユーザーフレンドリーなメッセージ変換
    - [ ] IAM権限エラーの特別処理
    - [ ] 必要な権限の表示
  - [ ] リトライロジック実装
    - [ ] スロットリングエラー対応
    - [ ] 一時的なネットワークエラー対応
  - [ ] リファクタリング

- [ ] **2.8 リソース解決の統合（TDD）**
  - [ ] `resolver_test.go` に統合テスト追加
    - [ ] 全リソース一括解決テスト
    - [ ] 部分的失敗ケーステスト
  - [ ] `resolver.go` に統合関数実装
    - [ ] 全リソースを一括解決
    - [ ] 並行処理による高速化（goroutineとerrgroup使用）
  - [ ] 解決結果の構造体定義
    - [ ] `ResolvedResources` 構造体
    - [ ] 各リソースのIDと詳細情報
  - [ ] リファクタリング

- [ ] **2.9 Epic 2 完了確認（MUST）**
  - [ ] `make ci` 実行
    - [ ] `make test` - すべてのテストがパス
    - [ ] `make lint` - リンターエラーを修正
    - [ ] `make modernize` - 最新化の問題を修正
    - [ ] `make fmt` - フォーマット適用
  - [ ] すべてのチェックがパスするまで修正を繰り返す
  - [ ] Gitコミット・プッシュ
    - [ ] `git add .`
    - [ ] `git commit -m "feat: complete Epic 2 - AWS integration and resource resolution"`
    - [ ] `git push origin main`

---

## Epic 3: initコマンド実装

**目的**: 既存のAppConfigリソースから設定ファイルを生成する機能を実装する

**依存**: Epic 1, Epic 2

### TDD実装順序

コマンドのロジックをテスト可能な関数として分離し、テストファーストで実装します。

### タスクチェックリスト

- [ ] **3.1 コマンド定義（TDD）**
  - [ ] `cmd/init_test.go` 作成
    - [ ] フラグ解析テスト
    - [ ] 必須フラグ検証テスト
  - [ ] `cmd/init.go` 作成
  - [ ] コマンド登録
  - [ ] フラグ定義
    - [ ] `--app` (必須)
    - [ ] `--profile` (必須)
    - [ ] `--env` (必須)
    - [ ] `--region` (オプション)
    - [ ] `--config, -c` (オプション、デフォルト: apcdeploy.yml)
    - [ ] `--output-data` (オプション、デフォルト: 自動判定)
  - [ ] フラグバリデーション実装

- [ ] **3.2 設定ファイル生成ロジック（TDD）**
  - [ ] `internal/config/generator_test.go` 作成
    - [ ] apcdeploy.yml生成テスト
    - [ ] 各フィールド値の検証テスト
    - [ ] ファイル上書き確認テスト
  - [ ] `internal/config/generator.go` 実装
    - [ ] テンプレートから設定ファイル生成
    - [ ] YAML書き込み
    - [ ] ファイル上書き確認
  - [ ] リファクタリング

- [ ] **3.3 設定データ取得・生成（TDD）**
  - [ ] `generator_test.go` にデータ生成テスト追加
    - [ ] ContentType判定テスト
    - [ ] ファイル名決定テスト
    - [ ] フォーマット整形テスト（JSON/YAML）
    - [ ] バージョンが存在しない場合のテスト
  - [ ] AWS APIモック拡張
    - [ ] `ListHostedConfigurationVersions` モック
    - [ ] `GetHostedConfigurationVersion` モック
  - [ ] `generator.go` に実装追加
    - [ ] 最新バージョン取得
    - [ ] 設定データ取得
    - [ ] ContentTypeに基づくファイル名決定
      - [ ] `application/json` → data.json
      - [ ] `application/x-yaml` → data.yaml
      - [ ] `text/plain` → data.txt
    - [ ] ユーザー指定ファイル名のサポート
    - [ ] データ書き込みとフォーマット整形
  - [ ] リファクタリング

- [ ] **3.4 initコマンド統合（TDD）**
  - [ ] `cmd/init_test.go` に統合テスト追加
    - [ ] エンドツーエンド実行テスト
    - [ ] 出力メッセージ検証テスト
  - [ ] `cmd/init.go` に統合実装
    - [ ] リソース解決の呼び出し
    - [ ] ジェネレーターの呼び出し
    - [ ] 出力メッセージ実装
      - [ ] 初期化開始メッセージ
      - [ ] リソース情報表示
      - [ ] 生成ファイル一覧表示
      - [ ] Next steps ガイド
  - [ ] エラーハンドリング実装
    - [ ] リソース不存在エラー
    - [ ] バージョンが存在しない場合の警告
    - [ ] ファイル書き込みエラー
    - [ ] 権限エラー
  - [ ] リファクタリング

- [ ] **3.5 Epic 3 完了確認（MUST）**
  - [ ] `make ci` 実行
    - [ ] `make test` - すべてのテストがパス
    - [ ] `make lint` - リンターエラーを修正
    - [ ] `make modernize` - 最新化の問題を修正
    - [ ] `make fmt` - フォーマット適用
  - [ ] すべてのチェックがパスするまで修正を繰り返す
  - [ ] Gitコミット・プッシュ
    - [ ] `git add .`
    - [ ] `git commit -m "feat: complete Epic 3 - init command implementation"`
    - [ ] `git push origin main`

---

## Epic 4: deployコマンド実装

**目的**: 設定データをAWS AppConfigにデプロイする機能を実装する

**依存**: Epic 1, Epic 2

### TDD実装順序

デプロイロジックを小さな関数に分割し、それぞれをテストファーストで実装します。

### タスクチェックリスト

- [ ] **4.1 コマンド定義**
  - [ ] `cmd/deploy.go` 作成
  - [ ] コマンド登録
  - [ ] フラグ定義
    - [ ] `--config, -c` (オプション)
    - [ ] `--no-wait` (オプション)
    - [ ] `--timeout` (オプション)
  - [ ] フラグバリデーション

- [ ] **4.2 設定読み込み**
  - [ ] apcdeploy.yml 読み込み
  - [ ] 設定データファイル読み込み
  - [ ] パス解決（相対パス対応）

- [ ] **4.3 リソースID解決**
  - [ ] Application/Profile/Environment/Strategy の解決
  - [ ] リソース情報表示
  - [ ] Profile Typeの取得

- [ ] **4.4 デプロイ中チェック**
  - [ ] `internal/aws/deployment.go` 作成
  - [ ] `ListDeployments` API呼び出し
  - [ ] 進行中のデプロイ検出
    - [ ] DEPLOYING ステータスのチェック
  - [ ] デプロイ中の場合、エラー表示して終了
    - [ ] 現在のデプロイ情報表示
    - [ ] デプロイ番号、開始時刻、戦略

- [ ] **4.5 ContentType決定**
  - [ ] Profile Typeに基づく判定
    - [ ] Feature Flags → `application/json`
    - [ ] Freeform → ファイル拡張子から判定
      - [ ] `.json` → `application/json`
      - [ ] `.yaml`, `.yml` → `application/x-yaml`
      - [ ] その他 → `text/plain`

- [ ] **4.6 ローカルバリデーション**
  - [ ] ファイル存在確認
  - [ ] サイズチェック（2MB以下）
  - [ ] 構文チェック
    - [ ] JSON: `json.Unmarshal`
    - [ ] YAML: `yaml.Unmarshal`
  - [ ] エラー表示（行番号含む）

- [ ] **4.7 バージョン作成**
  - [ ] `CreateHostedConfigurationVersion` API呼び出し
  - [ ] パラメータ設定
    - [ ] ApplicationId
    - [ ] ConfigurationProfileId
    - [ ] Content（設定データ）
    - [ ] ContentType
    - [ ] Description（オプション）
  - [ ] バージョン番号取得
  - [ ] AWS側Validatorエラーハンドリング
    - [ ] JSON Schema エラー
    - [ ] Lambda Function エラー
    - [ ] エラーメッセージの整形表示

- [ ] **4.8 デプロイ開始**
  - [ ] `StartDeployment` API呼び出し
  - [ ] パラメータ設定
    - [ ] ApplicationId
    - [ ] EnvironmentId
    - [ ] DeploymentStrategyId
    - [ ] ConfigurationProfileId
    - [ ] ConfigurationVersion
    - [ ] Description
  - [ ] デプロイ番号取得
  - [ ] デプロイ開始メッセージ表示

- [ ] **4.9 デプロイ待機**
  - [ ] `--wait` フラグによる制御
  - [ ] ポーリングループ実装
    - [ ] `GetDeployment` API定期呼び出し
    - [ ] ステータス確認（DEPLOYING / COMPLETE / ROLLED_BACK）
    - [ ] 進捗表示（パーセンテージ、経過時間）
  - [ ] タイムアウト処理
  - [ ] 完了/失敗判定

- [ ] **4.10 結果表示**
  - [ ] 成功時のサマリー
    - [ ] バージョン番号
    - [ ] デプロイ番号
    - [ ] 所要時間
  - [ ] 失敗時のエラー詳細
  - [ ] ロールバック発生時の警告

- [ ] **4.11 エラーハンドリング**
  - [ ] 設定ファイル読み込みエラー
  - [ ] リソース解決エラー
  - [ ] デプロイ中エラー
  - [ ] バリデーションエラー
  - [ ] API権限エラー
  - [ ] タイムアウトエラー

- [ ] **4.12 Epic 4 完了確認（MUST）**
  - [ ] `make ci` 実行
    - [ ] `make test` - すべてのテストがパス
    - [ ] `make lint` - リンターエラーを修正
    - [ ] `make modernize` - 最新化の問題を修正
    - [ ] `make fmt` - フォーマット適用
  - [ ] すべてのチェックがパスするまで修正を繰り返す
  - [ ] Gitコミット・プッシュ
    - [ ] `git add .`
    - [ ] `git commit -m "feat: complete Epic 4 - deploy command implementation"`
    - [ ] `git push origin main`

---

## Epic 5: diffコマンド実装

**目的**: ローカル設定とデプロイ済み設定の差分を表示する機能を実装する

**依存**: Epic 1, Epic 2

### TDD実装順序

差分計算ロジックと表示ロジックを分離し、テストファーストで実装します。

### タスクチェックリスト

- [ ] **5.1 コマンド定義**
  - [ ] `cmd/diff.go` 作成
  - [ ] コマンド登録
  - [ ] フラグ定義
    - [ ] `--config, -c` (オプション)

- [ ] **5.2 設定読み込み**
  - [ ] apcdeploy.yml 読み込み
  - [ ] ローカル設定データファイル読み込み
  - [ ] リソースID解決

- [ ] **5.3 最新デプロイ取得**
  - [ ] `ListDeployments` API呼び出し
  - [ ] 最新のデプロイ特定
    - [ ] 完了済み（COMPLETE）
    - [ ] 進行中（DEPLOYING）
  - [ ] デプロイが存在しない場合の処理
  - [ ] デプロイ中の場合の警告表示

- [ ] **5.4 リモート設定取得**
  - [ ] デプロイからバージョン番号取得
  - [ ] `GetHostedConfigurationVersion` API呼び出し
  - [ ] 設定データ取得

- [ ] **5.5 差分計算**
  - [ ] `internal/diff/calculator.go` 作成
  - [ ] go-diff ライブラリ使用
  - [ ] Unified diff 形式で差分生成
  - [ ] 正規化処理
    - [ ] JSON: インデント統一、キーソート
    - [ ] YAML: フォーマット統一
    - [ ] Text: 改行コード統一

- [ ] **5.6 差分表示**
  - [ ] `internal/diff/display.go` 作成
  - [ ] ヘッダー表示
    - [ ] Configuration情報
    - [ ] ローカルファイル名
    - [ ] リモートバージョン番号
  - [ ] Unified diff 表示
    - [ ] 削除行（-）を赤色
    - [ ] 追加行（+）を緑色
    - [ ] コンテキスト行
  - [ ] サマリー表示
    - [ ] 変更行数

- [ ] **5.7 特殊ケース処理**
  - [ ] デプロイが存在しない場合
    - [ ] "初回デプロイ" メッセージ
    - [ ] ローカルファイルの内容表示
  - [ ] デプロイ中の場合
    - [ ] 警告メッセージ
    - [ ] デプロイ番号と状況表示
  - [ ] 差分なしの場合
    - [ ] "No changes" メッセージ

- [ ] **5.8 エラーハンドリング**
  - [ ] 設定ファイル読み込みエラー
  - [ ] リソース解決エラー
  - [ ] API呼び出しエラー
  - [ ] 差分計算エラー

- [ ] **5.9 Epic 5 完了確認（MUST）**
  - [ ] `make ci` 実行
    - [ ] `make test` - すべてのテストがパス
    - [ ] `make lint` - リンターエラーを修正
    - [ ] `make modernize` - 最新化の問題を修正
    - [ ] `make fmt` - フォーマット適用
  - [ ] すべてのチェックがパスするまで修正を繰り返す
  - [ ] Gitコミット・プッシュ
    - [ ] `git add .`
    - [ ] `git commit -m "feat: complete Epic 5 - diff command implementation"`
    - [ ] `git push origin main`

---

## Epic 6: statusコマンド実装

**目的**: デプロイ状況を確認する機能を実装する

**依存**: Epic 1, Epic 2

### TDD実装順序

ステータス表示フォーマッターをテストファーストで実装します。

### タスクチェックリスト

- [ ] **6.1 コマンド定義**
  - [ ] `cmd/status.go` 作成
  - [ ] コマンド登録
  - [ ] フラグ定義
    - [ ] `--config, -c` (オプション)
    - [ ] `--deployment` (オプション)

- [ ] **6.2 設定読み込み**
  - [ ] apcdeploy.yml 読み込み
  - [ ] リソースID解決

- [ ] **6.3 デプロイ情報取得**
  - [ ] デプロイ番号指定時
    - [ ] `GetDeployment` API呼び出し
  - [ ] デプロイ番号未指定時
    - [ ] `ListDeployments` で最新取得
    - [ ] `GetDeployment` で詳細取得
  - [ ] デプロイが存在しない場合の処理

- [ ] **6.4 ステータス表示**
  - [ ] `internal/display/status.go` 作成
  - [ ] ヘッダー表示
    - [ ] Configuration情報
  - [ ] デプロイ情報表示
    - [ ] デプロイ番号
    - [ ] ステータス（DEPLOYING / COMPLETE / ROLLED_BACK / BAKING）
    - [ ] 開始時刻
    - [ ] 完了時刻（完了済みの場合）
    - [ ] デプロイ戦略名
  - [ ] 設定バージョン情報
  - [ ] Description表示

- [ ] **6.5 進捗表示（デプロイ中の場合）**
  - [ ] 進捗パーセンテージ
  - [ ] 経過時間
  - [ ] 推定残り時間（可能な場合）
  - [ ] イベントタイムライン
    - [ ] デプロイ開始
    - [ ] 各段階の完了（50%, 100%等）

- [ ] **6.6 タイムライン表示（完了済みの場合）**
  - [ ] デプロイ開始時刻
  - [ ] 各フェーズの完了時刻
  - [ ] Baking期間完了時刻
  - [ ] 総所要時間

- [ ] **6.7 特殊ケース処理**
  - [ ] デプロイが存在しない場合
    - [ ] "No deployments found" メッセージ
    - [ ] Next steps ガイド
  - [ ] ロールバック発生時
    - [ ] ロールバック理由表示
    - [ ] CloudWatch Alarms情報（取得可能な場合）

- [ ] **6.8 エラーハンドリング**
  - [ ] 設定ファイル読み込みエラー
  - [ ] リソース解決エラー
  - [ ] デプロイ番号不正エラー
  - [ ] API呼び出しエラー

- [ ] **6.9 Epic 6 完了確認（MUST）**
  - [ ] `make ci` 実行
    - [ ] `make test` - すべてのテストがパス
    - [ ] `make lint` - リンターエラーを修正
    - [ ] `make modernize` - 最新化の問題を修正
    - [ ] `make fmt` - フォーマット適用
  - [ ] すべてのチェックがパスするまで修正を繰り返す
  - [ ] Gitコミット・プッシュ
    - [ ] `git add .`
    - [ ] `git commit -m "feat: complete Epic 6 - status command implementation"`
    - [ ] `git push origin main`

---

## Epic 7: ドキュメントと最終テスト

**目的**: ドキュメントを整備し、最終的な統合テストを実施する

**依存**: Epic 1-6

**注**: 各Epicで既にTDDによりユニットテストは実装済み。このEpicでは統合テストとドキュメント整備に注力。

### タスクチェックリスト

- [ ] **7.1 統合テストの追加**
  - [ ] `tests/integration/` ディレクトリ作成
  - [ ] エンドツーエンドシナリオテスト作成
    - [ ] init → diff → deploy → status の一連の流れ
    - [ ] Feature Flags シナリオ
    - [ ] Freeform (JSON/YAML/Text) シナリオ
  - [ ] エラーシナリオテスト
    - [ ] リソース不存在
    - [ ] デプロイ中の再デプロイ試行
    - [ ] バリデーションエラー

- [ ] **7.2 テストカバレッジ確認**
  - [ ] カバレッジレポート生成 (`go test -cover`)
  - [ ] カバレッジ80%以上を目標に不足箇所を補完
  - [ ] エッジケースの追加テスト

- [ ] **7.3 テストデータ整備**
  - [ ] `testdata/` ディレクトリ整理
  - [ ] サンプル設定ファイル
    - [ ] Feature Flags (JSON)
    - [ ] Freeform (JSON)
    - [ ] Freeform (YAML)
    - [ ] Freeform (Text)
  - [ ] エラーケース用データ
    - [ ] 不正なJSON
    - [ ] サイズ超過データ
    - [ ] 不正なYAML

- [ ] **7.4 README作成**
  - [ ] プロジェクト概要
  - [ ] インストール方法
    - [ ] バイナリダウンロード
    - [ ] `go install`
    - [ ] ビルド方法（Go 1.25+必須）
  - [ ] クイックスタート
    - [ ] 前提条件（AWS リソース、IAM権限）
    - [ ] 基本的な使い方
  - [ ] 設定ファイルの説明
  - [ ] コマンドリファレンス（簡易版）
  - [ ] トラブルシューティング
  - [ ] TDD開発手法の説明
  - [ ] ライセンス情報

- [ ] **7.5 コマンドリファレンス**
  - [ ] `docs/commands.md` 作成
  - [ ] init コマンド詳細
    - [ ] 全フラグの説明
    - [ ] 使用例
  - [ ] deploy コマンド詳細
  - [ ] diff コマンド詳細
  - [ ] status コマンド詳細

- [ ] **7.6 設定ファイルリファレンス**
  - [ ] `docs/configuration.md` 作成
  - [ ] apcdeploy.yml の全フィールド説明
  - [ ] Feature Flags データ構造
  - [ ] Freeform データ構造
  - [ ] サンプル集

- [ ] **7.7 サンプルファイル**
  - [ ] `examples/` ディレクトリ作成
  - [ ] Feature Flags サンプル
    - [ ] `examples/feature-flags/apcdeploy.yml`
    - [ ] `examples/feature-flags/flags.json`
  - [ ] Freeform (JSON) サンプル
  - [ ] Freeform (YAML) サンプル
  - [ ] Freeform (Text) サンプル
  - [ ] 複数環境管理の例

- [ ] **7.8 CI/CD設定**
  - [ ] GitHub Actions ワークフロー作成
    - [ ] `.github/workflows/test.yml` - テスト実行（Go 1.25使用）
    - [ ] `.github/workflows/release.yml` - リリース作成
  - [ ] ビルドスクリプト
    - [ ] クロスコンパイル対応
    - [ ] バージョン埋め込み
  - [ ] リリースプロセス文書化

- [ ] **7.9 その他ドキュメント**
  - [ ] CONTRIBUTING.md（開発者向け）
    - [ ] TDD開発フローの説明
    - [ ] テスト実行方法
    - [ ] `make ci` の説明
  - [ ] CHANGELOG.md
  - [ ] IAM権限の詳細ガイド
  - [ ] AWS AppConfig リソース作成ガイド（Terraform例）

- [ ] **7.10 Epic 7 完了確認（MUST）**
  - [ ] `make ci` 実行
    - [ ] `make test` - すべてのテストがパス
    - [ ] `make lint` - リンターエラーを修正
    - [ ] `make modernize` - 最新化の問題を修正
    - [ ] `make fmt` - フォーマット適用
  - [ ] すべてのチェックがパスするまで修正を繰り返す
  - [ ] 最終確認: すべてのコマンド（init/deploy/diff/status）が動作する
  - [ ] Gitコミット・プッシュ
    - [ ] `git add .`
    - [ ] `git commit -m "feat: complete Epic 7 - documentation and final testing"`
    - [ ] `git push origin main`

---

## 完成の定義（Definition of Done）

各Epicが完了とみなされる基準（TDD準拠）:

- [ ] すべてのタスクが完了している
- [ ] **実装前にテストが書かれている**（TDDの原則）
- [ ] すべてのユニットテストが通過している
- [ ] テストカバレッジが80%以上である
- [ ] **`make ci` がパスする**（MUST）
  - [ ] `make test` - すべてのテストがパス
  - [ ] `make lint` - リンターチェックがパス
  - [ ] `make modernize` - 最新化チェックがパス
  - [ ] `make fmt` - フォーマットチェックがパス
- [ ] コードレビューが完了している（チーム開発の場合）
- [ ] ドキュメントが更新されている
- [ ] 手動での動作確認が完了している
- [ ] エラーハンドリングが適切に実装され、テストされている
- [ ] ヘルプメッセージが整備されている

### TDDサイクルの確認

各機能実装において以下のサイクルが守られていること:

1. ✅ **Red**: 失敗するテストを書く
2. ✅ **Green**: テストを通す最小限の実装
3. ✅ **Refactor**: コードをリファクタリング（テストは維持）

### MUSTルールの再確認

**`make ci` が各Epic完了時に必ずパスすること**

これは絶対に守られる必要があります。各Epicの最後のタスクとして `make ci` の実行と修正が含まれています。
