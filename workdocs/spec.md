# AWS AppConfig デプロイツール 要件定義書

**ツール名**: `apcdeploy`  
**バージョン**: 1.0  
**最終更新**: 2025年10月3日

---

## 1. 目的

### 1.1 解決する課題

AWS AppConfigへの設定デプロイにおける以下の課題を解決する:

- **複雑な手順**: 複数のAWS CLIコマンド実行が必要
  - `create-hosted-configuration-version`
  - `start-deployment`
  - リソースID取得のための事前クエリ
- **バージョン管理の煩雑さ**: バージョン番号の手動管理
- **エラーのリスク**: リソースID間違い、JSON構文エラー、設定ミス
- **差分確認の困難さ**: デプロイ前の変更内容確認が手間
- **作業の重複**: 複数環境へのデプロイで同じ作業を繰り返す

### 1.2 実現する価値

- **ワンコマンドデプロイ**: 設定ファイル編集後、1コマンドでデプロイ完了
- **宣言的管理**: YAMLファイルでリソースを定義し、コードとして管理
- **CI/CD統合**: パイプラインへの組み込みが容易
- **エラー削減**: 自動検証とわかりやすいエラーメッセージ
- **開発速度向上**: 設定変更のイテレーション速度を大幅に改善

---

## 2. ターゲットユーザー

- **アプリケーション開発者**: 頻繁に設定を変更・テストする
- **DevOpsエンジニア**: CI/CDパイプラインを構築・運用する
- **SREチーム**: 本番環境の設定を安全に管理する

---

## 3. スコープ定義

### 3.1 サポートする機能

| カテゴリ | サポート内容 |
|---------|------------|
| **設定プロファイルタイプ** | • Feature Flags<br>• Freeform Configuration |
| **データフォーマット** | • JSON (Feature Flags / Freeform)<br>• YAML (Freeform のみ)<br>• Text (Freeform のみ) |
| **保存場所** | • Hosted Configuration Store のみ |
| **バリデーション** | • Configuration Profileに設定されたValidatorを使用<br>  - JSON Schema<br>  - Lambda Function |
| **デプロイ戦略** | • AWS提供の事前定義戦略<br>• ユーザー作成のカスタム戦略 |
| **リソース識別** | • 名前ベース (直感的) |
| **環境** | • 単一環境へのデプロイ |
| **デプロイ監視** | • 基本ステータス表示<br>• デプロイ完了まで待機 (オプション) |

### 3.2 実装するコマンド

```bash
# 既存リソースから設定ファイルを生成
apcdeploy init --app <APP_NAME> --profile <PROFILE_NAME> --env <ENV_NAME>

# デプロイ前の差分表示
apcdeploy diff --config apconfig.yml

# デプロイ実行
apcdeploy deploy --config apconfig.yml

# デプロイ状況確認
apcdeploy status --config apconfig.yml
```

### 3.3 初期実装での対象外

- ❌ ロールバック機能
- ❌ テンプレート機能 (環境変数展開、SSM/Secrets Manager参照)
- ❌ 複数環境への一括デプロイ
- ❌ S3/SSM Parameter Store/Secrets Manager 保存先
- ❌ Multi-variant Feature Flags
- ❌ リソース作成機能 (Application/Profile/Environment)
- ❌ Extensions サポート
- ❌ CodePipeline 統合
- ❌ CloudWatch Alarms 設定

---

## 4. 機能詳細

### 4.1 設定プロファイルタイプ

#### 4.1.1 Feature Flags

**概要**: 機能フラグによるアプリケーション動作の動的制御

**仕様**:
- データフォーマット: JSON のみ
- スキーマ: `AWS.AppConfig.FeatureFlags` (固定)
- 保存場所: Hosted Configuration Store 必須
- ContentType: `application/json`

**データ構造**:
```json
{
  "version": "1",
  "flags": {
    "flag_key": {
      "name": "Human Readable Name",
      "description": "Optional description",
      "attributes": {
        "attribute_name": {
          "description": "Attribute description",
          "constraints": {
            "type": "string|number|boolean|array",
            "required": true
          }
        }
      }
    }
  },
  "values": {
    "flag_key": {
      "enabled": true,
      "attribute_name": "value"
    }
  }
}
```

**特徴**:
- 自動バリデーション (AWS AppConfigによる組み込みスキーマ検証)
- `flags`セクション: フラグ定義とメタデータ
- `values`セクション: 実際のフラグ値

#### 4.1.2 Freeform Configuration

**概要**: 任意の形式でアプリケーション設定を管理

**仕様**:
- データフォーマット: JSON / YAML / Text
- 保存場所: Hosted Configuration Store
- 任意のデータ構造

**サポートするContentType**:

| フォーマット | ContentType | 用途 |
|-------------|-------------|------|
| JSON | `application/json` | 構造化データ |
| YAML | `application/x-yaml` | 構造化データ (可読性重視) |
| Text | `text/plain` | 任意のテキストデータ |

**データ例**:

```json
// JSON
{
  "api": {
    "endpoint": "https://api.example.com",
    "timeout": 30
  }
}
```

```yaml
# YAML
api:
  endpoint: https://api.example.com
  timeout: 30
```

```text
# Text
API_ENDPOINT=https://api.example.com
TIMEOUT=30
MAX_CONNECTIONS=100
```

### 4.2 保存場所

**Hosted Configuration Store**:
- AWS AppConfig組み込みストレージ
- 最大サイズ: 2MB
- バージョン管理: 自動
- 料金: 無料
- IAM設定: 不要 (AppConfig権限のみ)
- URI: `hosted`

### 4.3 バリデーション

#### 4.3.1 バリデーションの仕組み

**前提**: Configuration Profileにvalidatorを事前設定

**処理フロー**:
```
1. ツールが設定データを読み込み
2. 基本的なチェック (構文、サイズ)
3. CreateHostedConfigurationVersion API呼び出し
4. AWS AppConfigが自動的にValidatorを実行
   - JSON Schema
   - Lambda Function
5. 検証成功 → バージョン作成完了
   検証失敗 → エラー返却
6. ツールがエラーをわかりやすく表示
```

**ツールの役割**:
- ✅ ファイル読み込み・構文チェック
- ✅ サイズ確認 (2MB以下)
- ✅ AWS APIエラーの適切な表示
- ❌ 詳細なバリデーション実行 (AWS側に委譲)

#### 4.3.2 サポートするValidator

| Validator | Feature Flags | Freeform | 説明 |
|-----------|--------------|----------|------|
| **JSON Schema** | 自動適用 | オプション | 構文・構造の検証 |
| **Lambda** | オプション | オプション | カスタムロジック |

### 4.4 デプロイ戦略

**サポート範囲**:
- ✅ AWS提供の事前定義戦略
- ✅ ユーザー作成のカスタム戦略

**AWS提供の戦略例**:
- `AppConfig.AllAtOnce` - 即座にデプロイ
- `AppConfig.Linear50PercentEvery30Seconds` - 30秒ごとに50%ずつ
- `AppConfig.Linear20PercentEvery6Minutes` - 6分ごとに20%ずつ
- `AppConfig.Canary10Percent20Minutes` - 10%を20分、残り90%を即座に

**指定方法**:
- 名前で指定 (推奨): `deployment_strategy: "AppConfig.AllAtOnce"`
- IDで指定: `deployment_strategy_id: "abc123def"`

**処理方法**:
```
1. ListDeploymentStrategies APIで全戦略を取得
2. 名前で完全一致マッチング (大文字小文字区別あり)
3. IDを取得
4. StartDeployment APIに渡す
```

---

## 5. 設定ファイル仕様

### 5.1 メイン設定ファイル (apcdeploy.yml)

**ファイル名**: `apcdeploy.yml` (デフォルト)

**基本構造**:
```yaml
# AWS設定
region: ap-northeast-1

# AppConfigリソース (名前で指定)
application: "MyApp"
configuration_profile: "MyProfile"
environment: "Production"

# 設定データファイル
configuration_data: "config.json"  # または config.yaml, config.txt

# デプロイ戦略
deployment_strategy: "AppConfig.Linear50PercentEvery30Seconds"

# デプロイオプション (オプション)
deployment:
  description: "Deployed by apcdeploy"
  wait: true           # デプロイ完了まで待機
  timeout: "10m"       # タイムアウト
```

**必須フィールド**:
- `region`
- `application`
- `configuration_profile`
- `environment`
- `configuration_data`

**オプションフィールド**:
- `deployment_strategy` (デフォルト: `AppConfig.AllAtOnce`)
- `deployment.description`
- `deployment.wait`
- `deployment.timeout`

### 5.2 設定データファイル

**Feature Flags の例 (JSON)**:
```json
{
  "version": "1",
  "flags": {
    "new_ui": {
      "name": "New UI Feature"
    },
    "beta_features": {
      "name": "Beta Features"
    }
  },
  "values": {
    "new_ui": {
      "enabled": true
    },
    "beta_features": {
      "enabled": false
    }
  }
}
```

**Freeform の例 (JSON)**:
```json
{
  "api_endpoint": "https://api.example.com",
  "timeout": 30,
  "max_connections": 100
}
```

**Freeform の例 (YAML)**:
```yaml
api_endpoint: https://api.example.com
timeout: 30
max_connections: 100
```

**Freeform の例 (Text)**:
```text
API_ENDPOINT=https://api.example.com
TIMEOUT=30
MAX_CONNECTIONS=100
```

---

## 6. コマンド仕様

### 6.1 init コマンド

**用途**: 既存のAppConfigリソースから設定ファイルを生成

**基本構文**:
```bash
apcdeploy init --app <APP_NAME> --profile <PROFILE_NAME> --env <ENV_NAME> [OPTIONS]
```

**必須オプション**:
```
--app           Application名
--profile       Configuration Profile名
--env           Environment名
```

**オプション**:
```
--region        AWSリージョン (デフォルト: AWS_REGIONまたはデフォルトプロファイル)
--config, -c    出力する設定ファイルのパス (デフォルト: apcdeploy.yml)
--output-data   設定データの出力ファイルパス (デフォルト: 自動判定)
```

**実行例**:
```bash
# 基本的な使用
apcdeploy init --app MyApp --profile MyProfile --env Production

# 出力ファイル名を指定
apcdeploy init --app MyApp --profile MyProfile --env Production \
  --config prod.yml \
  --output-data config.prod.json
```

**処理フロー**:
```
1. 指定されたリソースの存在確認
   - Application
   - Configuration Profile
   - Environment
2. Configuration Profileの情報取得
   - Type (Feature Flags / Freeform)
   - LocationUri
   - Validators
3. 最新のHosted Configuration Versionを取得
4. apcdeploy.ymlファイルを生成
5. 設定データファイルを生成
   - Feature Flags → data.json
   - Freeform (JSON) → data.json
   - Freeform (YAML) → data.yaml
   - Freeform (Text) → data.txt
6. 生成されたファイルを表示
```

**生成される出力**:
```
✓ Initializing configuration from AWS AppConfig

Application: MyApp (app-abc123)
Profile: MyProfile (prof-def456) [Type: AWS.Freeform]
Environment: Production (env-ghi789)
Latest Version: 42

✓ Generated configuration files:
  - apcdeploy.yml
  - config.json (1.2 KB)

Next steps:
  1. Edit config.json as needed
  2. Run: apcdeploy diff
  3. Run: apcdeploy deploy
```

### 6.2 diff コマンド

**用途**: ローカルの設定データと現在デプロイされている設定の差分を表示

**基本構文**:
```bash
apcdeploy diff [OPTIONS]
```

**オプション**:
```
--config, -c    設定ファイルのパス (デフォルト: apcdeploy.yml)
```

**実行例**:
```bash
# デフォルト設定ファイルを使用
apcdeploy diff

# 設定ファイルを指定
apcdeploy diff --config production.yml
```

**処理フロー**:
```
1. 設定ファイル (apcdeploy.yml) を読み込み
2. ローカルの設定データファイルを読み込み
3. 最新のデプロイを取得
   - ListDeployments → GetDeployment で最新のデプロイを特定
   - デプロイ中(DEPLOYING)の場合、警告を表示
   - GetHostedConfigurationVersion で設定データを取得
4. 差分を計算・表示
```

**Note**: 最新のデプロイ(完了済み、または進行中)の設定バージョンを比較元とする。デプロイ中の場合は状況が変わる可能性があるため警告を表示。

**差分表示形式**:
```
Configuration: MyApp / MyProfile / Production

Local file:  config.json
Remote version: 42 (deployed)

--- Remote (Version 42)
+++ Local (config.json)

@@ -1,5 +1,5 @@
 {
   "api_endpoint": "https://api.example.com",
-  "timeout": 30,
+  "timeout": 60,
   "max_connections": 100
 }

Summary:
  1 line changed
```

**デプロイ中の場合**:
```
Configuration: MyApp / MyProfile / Production

⚠ Warning: A deployment is currently in progress (Deployment #124)
The comparison below may not reflect the actual deployed state.

Local file:  config.json
Remote version: 43 (deploying)

--- Remote (Version 43)
+++ Local (config.json)

@@ -1,5 +1,5 @@
 {
   "api_endpoint": "https://api.example.com",
-  "timeout": 60,
+  "timeout": 90,
   "max_connections": 100
 }

Summary:
  1 line changed
```

**デプロイされていない場合**:
```
Configuration: MyApp / MyProfile / Production

No deployment found for this environment.
This will be the first deployment.

Local file: config.json (1.2 KB)
```

### 6.3 deploy コマンド

**用途**: 設定をデプロイする

**基本構文**:
```bash
apcdeploy deploy [OPTIONS]
```

**オプション**:
```
--config, -c    設定ファイルのパス (デフォルト: apcdeploy.yml)
--no-wait       デプロイ完了を待たずに終了
--timeout       タイムアウト時間 (デフォルト: 設定ファイルまたは10m)
```

**実行例**:
```bash
# デフォルト設定ファイルを使用
apcdeploy deploy

# 設定ファイルを指定
apcdeploy deploy --config production.yml

# デプロイ開始後すぐに終了
apcdeploy deploy --no-wait
```

**処理フロー**:
```
1. 設定ファイル (apcdeploy.yml) を読み込み
2. 設定データファイルを読み込み
3. リソース名→ID変換
   - Application
   - Configuration Profile
   - Environment
   - Deployment Strategy
4. 既存デプロイの確認
   - ListDeployments APIで進行中のデプロイをチェック
   - デプロイ中の場合はエラーを表示して終了
5. プロファイルタイプを取得 (Feature Flags / Freeform)
6. ContentTypeを決定
7. 基本バリデーション
   - ファイル存在確認
   - 構文チェック (JSON/YAML)
   - サイズチェック (2MB以下)
8. CreateHostedConfigurationVersion
   - AWS側でValidator実行
9. StartDeployment
10. (オプション) デプロイ完了まで待機
11. 結果表示
```

**同時デプロイ制限**:
AWS AppConfigは1環境につき1つのデプロイのみ同時実行可能。進行中のデプロイが存在する場合、以下のエラーメッセージを表示してコマンドを終了する:

```
Error: A deployment is already in progress for this environment

Current deployment: #123 (DEPLOYING)
Started: 2025-10-03 14:30:00
Strategy: AppConfig.Linear50PercentEvery30Seconds

Please wait for the current deployment to complete, or use:
  apcdeploy status --config apcdeploy.yml
```

**成功時の出力例**:
```
✓ Loading configuration from apcdeploy.yml
✓ Reading configuration data from config.json (1.2 KB)
✓ Resolving resource IDs...
  Application: MyApp (app-abc123)
  Profile: MyProfile (prof-def456) [Type: AWS.Freeform]
  Environment: Production (env-ghi789)
  Strategy: AppConfig.Linear50PercentEvery30Seconds

✓ Creating configuration version...
  Version: 42
  ContentType: application/json

✓ Starting deployment...
  Deployment: #123
  Strategy: Linear (50% every 30 seconds)
  
⏳ Waiting for deployment to complete...

✓ Deployment completed successfully

Summary:
  Version: 42
  Deployment: #123
  Duration: 1m 30s
```

### 6.4 status コマンド

**用途**: 現在のデプロイ状況を確認

**基本構文**:
```bash
apcdeploy status [OPTIONS]
```

**オプション**:
```
--config, -c    設定ファイルのパス (デフォルト: apcdeploy.yml)
--deployment    特定のデプロイ番号を指定
```

**実行例**:
```bash
# 最新のデプロイ状況を表示
apcdeploy status

# 特定のデプロイを表示
apcdeploy status --deployment 123
```

**処理フロー**:
```
1. 設定ファイルを読み込み
2. リソースIDを解決
3. GetDeployment APIで状況を取得
4. デプロイ情報を整形して表示
```

**出力例 (デプロイ中)**:
```
Configuration: MyApp / MyProfile / Production

Current Deployment: #123
Status: DEPLOYING
Started: 2025-10-03 14:30:00
Strategy: AppConfig.Linear50PercentEvery30Seconds

Progress:
  Completed: 50%
  Duration: 1m 0s / ~2m 0s

Configuration Version: 42
Description: Deployed by apcdeploy

Events:
  [14:30:00] Deployment started
  [14:30:30] Deployed to 50% of targets
```

**出力例 (デプロイ完了)**:
```
Configuration: MyApp / MyProfile / Production

Latest Deployment: #123
Status: COMPLETE
Completed: 2025-10-03 14:32:00
Duration: 2m 0s

Configuration Version: 42
Strategy: AppConfig.Linear50PercentEvery30Seconds

Timeline:
  [14:30:00] Deployment started
  [14:30:30] 50% of targets updated
  [14:31:00] 100% of targets updated
  [14:32:00] Baking period completed
```

**出力例 (デプロイなし)**:
```
Configuration: MyApp / MyProfile / Production

No deployments found for this environment.

Use 'apcdeploy deploy' to create the first deployment.
```

---

## 7. エラーハンドリング

### 7.1 設計原則

**わかりやすいエラーメッセージ**:
1. **何が起きたか** を明確に
2. **どこで** 起きたかを特定
3. **なぜ** 起きたかを説明
4. **どうすればいいか** のヒントを提供

### 7.2 エラーの種類

**言語方針**: すべてのエラーメッセージは英語で統一

| エラー種別 | 検出タイミング | 対応 |
|-----------|-------------|------|
| 設定ファイルエラー | 起動時 | ファイルパス、構文エラーを明示 |
| リソース不存在 | API呼び出し時 | 利用可能なリソース一覧を表示 |
| ファイル読み込みエラー | 読み込み時 | パス、権限、サイズを確認 |
| 構文エラー | パース時 | エラー行・列を表示 |
| Validatorエラー | バージョン作成時 | Validator詳細とヒントを表示 |
| デプロイ中エラー | デプロイ開始前 | 現在のデプロイ状況を表示して終了 |
| デプロイエラー | デプロイ時 | デプロイ状態と原因を表示 |
| IAM権限エラー | API呼び出し時 | 必要な権限を明示 |

---

## 8. 技術仕様

### 8.1 実装言語

**Go言語**

**選定理由**:
- シングルバイナリ配布が容易
- クロスプラットフォーム対応
- AWS SDK for Go v2 の充実したサポート
- ecspressoと同じ言語 (参考実装)
- 高速な実行速度

### 8.2 使用するライブラリ・フレームワーク

**CLIフレームワーク**:
- **[cobra](https://github.com/spf13/cobra)** - CLIアプリケーション構築
  - サブコマンド管理
  - フラグ・引数パース
  - ヘルプ生成
  - シェル補完

**AWS SDK**:
- **[AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)** - AWS API呼び出し
  - `github.com/aws/aws-sdk-go-v2/service/appconfig`

**設定ファイル処理**:
- **[github.com/goccy/go-yaml](https://github.com/goccy/go-yaml)** - YAML読み書き
- **encoding/json** (標準ライブラリ) - JSON読み書き

**その他**:
- **[go-diff](https://github.com/sergi/go-diff)** - 差分表示

### 8.3 使用するAWS API

**リソース解決**:
- `ListApplications` - Application名→ID
- `ListConfigurationProfiles` - Profile名→ID
- `ListEnvironments` - Environment名→ID
- `ListDeploymentStrategies` - Strategy名→ID

**プロファイル情報取得**:
- `GetConfigurationProfile` - Type, LocationUri, Validators
- `GetHostedConfigurationVersion` - 設定データ取得 (diff用)
- `ListHostedConfigurationVersions` - バージョン一覧取得 (init用)

**バージョン作成・デプロイ**:
- `CreateHostedConfigurationVersion` - 設定バージョン作成
- `StartDeployment` - デプロイ開始
- `GetDeployment` - デプロイ状況確認
- `ListDeployments` - デプロイ一覧取得

### 8.4 前提条件

**既存リソース**:
以下はツール使用前に作成済みであること (Terraform等で管理):
- Application
- Configuration Profile (Freeform または Feature Flags)
- Environment
- Deployment Strategy (オプション: AWS提供のものを使用可能)

**IAM権限**:
実行ユーザー/ロールに以下の権限が必要:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "appconfig:GetApplication",
        "appconfig:ListApplications",
        "appconfig:GetConfigurationProfile",
        "appconfig:ListConfigurationProfiles",
        "appconfig:GetEnvironment",
        "appconfig:ListEnvironments",
        "appconfig:ListDeploymentStrategies",
        "appconfig:GetHostedConfigurationVersion",
        "appconfig:ListHostedConfigurationVersions",
        "appconfig:CreateHostedConfigurationVersion",
        "appconfig:StartDeployment",
        "appconfig:GetDeployment",
        "appconfig:ListDeployments"
      ],
      "Resource": "*"
    }
  ]
}
```

### 8.5 動作環境

**対応OS**:
- Linux (x86_64, ARM64)
- macOS (x86_64, ARM64)
- Windows (x86_64)

**前提ソフトウェア**:
- AWS認証情報の設定 (AWS CLI / 環境変数 / IAMロール)

---

## 9. ユースケース

### 9.1 初回セットアップ

```bash
# 1. 既存リソースから設定ファイルを生成
apcdeploy init --app MyApp --profile MyProfile --env Production

# 出力:
# ✓ Generated configuration files:
#   - apcdeploy.yml
#   - config.json

# 2. 設定ファイルをGitにコミット
git add apcdeploy.yml config.json
git commit -m "Add AppConfig management"
```

### 9.2 日常的な設定変更

```bash
# 1. 設定を編集
vim config.json

# 2. 差分確認
apcdeploy diff

# 3. デプロイ
apcdeploy deploy

# 4. 状況確認
apcdeploy status
```

### 9.3 CI/CDパイプラインでの使用

```yaml
# GitHub Actions例
name: Deploy AppConfig
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          role-to-assume: arn:aws:iam::123456789012:role/GitHubActions
          aws-region: ap-northeast-1
      
      - name: Show diff
        run: apcdeploy diff
      
      - name: Deploy
        run: apcdeploy deploy
```

### 9.4 複数環境の管理

```bash
# 環境ごとに設定ファイルを用意
apcdeploy init --app MyApp --profile MyProfile --env Development \
  --config dev.yml --output-data config.dev.json

apcdeploy init --app MyApp --profile MyProfile --env Staging \
  --config stg.yml --output-data config.stg.json

apcdeploy init --app MyApp --profile MyProfile --env Production \
  --config prod.yml --output-data config.prod.json

# 環境ごとにデプロイ
apcdeploy deploy --config dev.yml
apcdeploy deploy --config stg.yml
apcdeploy deploy --config prod.yml
```

---

## 10. 成功基準

### 10.1 目標

**定量的目標**:
- デプロイ時間: 従来の5-10分 → 1分以内
- 操作ステップ: 5-10ステップ → 1コマンド
- エラー率: 設定ミスによるエラーを50%削減

**定性的目標**:
- ✅ 既存リソースからの設定ファイル生成が簡単
- ✅ デプロイ前に差分確認ができる
- ✅ デプロイ状況がリアルタイムで確認できる
- ✅ エラーメッセージが理解しやすい
- ✅ CI/CDに組み込みやすい

### 10.2 受け入れ基準

**機能要件**:
- [ ] `init`コマンドで既存リソースから設定ファイル生成
- [ ] `diff`コマンドで差分表示
- [ ] `deploy`コマンドでデプロイ実行
- [ ] `status`コマンドでデプロイ状況確認
- [ ] Feature Flags (JSON) のサポート
- [ ] Freeform (JSON/YAML/Text) のサポート
- [ ] 名前ベースのリソース指定
- [ ] AWS提供・ユーザー作成のデプロイ戦略対応
- [ ] Validatorエラーの適切な表示
- [ ] デプロイ完了まで待機

**非機能要件**:
- [ ] シングルバイナリで配布可能
- [ ] Linux/macOS/Windowsで動作
- [ ] エラーメッセージが理解しやすい
- [ ] 実行時間が1分以内 (通常のデプロイ)

**ドキュメント**:
- [ ] README.mdに基本的な使い方を記載
- [ ] 設定ファイルのサンプルを提供
- [ ] コマンドリファレンス

---

## 11. 制約事項

### 11.1 技術的制約

- Hosted Configuration Storeのサイズ制限: 2MB
- Feature FlagsはJSON形式のみサポート
- Feature FlagsはHosted Store必須
- デプロイ戦略はユーザーが事前作成済みであること

### 11.2 機能的制約

- リソース作成機能なし (Terraformで管理)
- 単一環境のみサポート (複数環境は設定ファイルを分けて対応)
- ロールバック機能なし (手動で旧バージョンを再デプロイ)
- テンプレート機能なし (環境変数展開等は未対応)

### 11.3 運用上の考慮事項

**リソース管理の責任分担**:
- **Terraform**: Application, Profile, Environment, Validators, Alarms
- **apcdeploy**: 設定データのバージョン作成とデプロイのみ

**バージョン管理**:
- 設定ファイル (apcdeploy.yml, config.json) はGitで管理
- AppConfig側のバージョン番号は自動採番

---

## 付録A: 用語集

| 用語 | 説明 |
|------|------|
| **Application** | AppConfigの名前空間 (フォルダのようなもの) |
| **Configuration Profile** | 設定データの場所と型を定義 |
| **Environment** | デプロイ先の論理グループ (Production, Staging等) |
| **Hosted Configuration** | AppConfig内に保存された設定データ |
| **Feature Flags** | 機能のON/OFFを制御する設定タイプ |
| **Freeform Configuration** | 任意の形式の設定タイプ |
| **Deployment Strategy** | デプロイの速度と方法を定義 |
| **Validator** | 設定データの検証ルール |
| **Version** | 設定データのバージョン番号 |

---

## 付録B: FAQ

**Q: Terraformとの役割分担は?**  
A: Terraformはインフラ管理、apcdeployは設定データのデプロイのみを担当します。

**Q: Feature FlagsでYAMLは使えない?**  
A: AWS AppConfigの仕様上、Feature FlagsはJSON形式のみです。

**Q: 複数環境に同時デプロイしたい**  
A: 環境ごとに設定ファイルを分けて、順次実行してください。

**Q: デプロイが失敗した時のロールバックは?**  
A: CloudWatch Alarmsによる自動ロールバックはAppConfig側の機能です。手動ロールバックは旧バージョンを再デプロイしてください。

**Q: 既存の手動管理からの移行方法は?**  
A: `apcdeploy init`コマンドで既存リソースから設定ファイルを生成できます。

---

**ドキュメント履歴**:
- 2025-10-03: 要件定義 初版作成
