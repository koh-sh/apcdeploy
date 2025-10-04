# 動作確認して発見したバグ

## init

### 優先度 high

- [x] initで作成された設定のdeployment_strategyが "" になって名前が設定されていない
- [x] 生成されたファイルにregionが記載されていない。initだけ明示的にregionの指定が必要だが、それ以外のコマンドは設定ファイルからregionを読み込む
- [ ] initで作成された設定の deployment_strategyが直近使用されたデプロイ戦略ではなく固定でAppConfig.AllAtOnceになっている

### 優先度 mid

- [ ] --regionフラグなしかつ、AWS_REGION/AWS_DEFAULT_REGIONも指定されていない場合はクライアントに設定されているデフォルトregionを読みたい
  - [ ] regionフラグが指定されていればその値、されていなければaws sdkでdefault regionの解決。明示的に AWS_REGIONなどの環境変数を読む処理を動かさずに sdk に任せたい
- [ ] output-dataの方は既存のファイルの有無を確認せず強制上書きしているので、設定ファイルの方と振る舞いが一致していない。設定ファイルと同様に既存のファイルがあったら実行を止めたい

### 優先度 low

- [ ] ファイルの上書きができずにエラーになった際は、usageの表示は邪魔
- [ ] ファイルがすでに存在してエラーになった際に--force オプションについて言及しているがヘルプ表示には --force オプションについて表記されていない
- [ ] 設定ファイルがすでにあるかは先にチェックしたい。現状はappやenvの解決をした後のエラー表示される

## status

### 優先度 mid

- [ ] statusがROLLED_BACKなのに Descriptionが Deploying new configuration
- [ ] ROLLED_BACKの要因は CloudWatch Alermとは限らないので、理由を取得できたら取得して表示したい。できなければrollbackされたという事実だけ表示する
- [ ] statusの表示にデプロイ戦略を表示したい

### 優先度 low

- [ ] status実行の際は Resolving resources... の表示はいらない

## diff

### 優先度 high

- [x] 最新のデプロイがROLLBACKして反映されていないのに、そのデプロイの設定データを取得してしまっている。最後に成功したデプロイから取得する必要がある
- [x] diffが非常に見にくい。300というパラメータを修正した際に 3050 (2桁目の0が赤、3桁目の5が緑) という表記になっている
- [x] Summaryが完全に機能していない。どういうケースでも 0 add 0 deleteになっている

### 優先度 mid

- [ ] 既存のパラメータ修正の際に+0 additions, -0 deletionsとなっている。1 add 1 delete とするか、modified のような表記が欲しい
- [ ] deploy中のwarningが目立たないので見逃してしまう。Fetching latest deploymentの表記から一行あけたい
- [ ] featureflagの時は _updatedAt,_createdAt を差分表示しないようにしたい

### 優先度 low

- [ ] diff実行の際は Resolving resources... の表示はいらない

## deploy

### 優先度 high

- [ ] featureflagの場合、updatedAtが通常の運用の場合初回initした時の値から更新されないためずっと古いままになってしまう。initする際に意図的に_updatedAtのキーを削除する and diffの際に差分として表示しないように読み捨てる必要がある
- [ ] _createdAt も削除する and diffに表示しない

### 優先度 mid

- [ ] waitingのあとrollbackされたときにusageを表示するのはおかしい。こちらも理由がわかれば表示、わからなければrollbackされたという事実だけでいい
- [ ] デフォルトが AppConfig.AllAtOnce で処理時間10minなのに、timeoutのデフォルトが300secなのはおかしい
- [ ] デプロイ中にデプロイした時のエラーでusageを出すのはおかしい
- [ ] デフォルトは wait なしでいい。waitしたいときに --waitをつけるようにしたい
- [ ] Configurationのvaridationに失敗した際はusageを表示しない
- [ ] validationに失敗した時は、可能であればその理由を出力したい

### 優先度 low

- [ ] Timeout in seconds for deployment (default: 300) (default 300) のdefaultの表示が重複
- [ ] サブコマンド名を deploy 別の単語にしたい。実際に実行するときに apcdeploy deploy となるので違和感。候補 run/apply/exec
