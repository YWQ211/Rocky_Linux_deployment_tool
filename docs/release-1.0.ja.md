# Rockdev 1.0.0

[中文](release-1.0.md) · [English](release-1.0.en.md) · 日本語

## 既存環境の確認と更新

```bash
rockdev env check
rockdev env check python java go
rockdev env install python go
sudo rockdev env install python go --update
rockdev env install python --venv /home/your-user/project/.venv
```

コマンドのパス、バージョン、PATH の重複候補、JAVA_HOME の不一致、Python の分離状態を確認します。バージョン照会を実行しますが、プロジェクトの動作試験を代替するものではありません。

既存環境は初期状態では維持します。対話モードでは環境ごとに更新を確認します。`--yes` は実行確認を省略するだけで、更新の許可にはなりません。システム環境の更新には `--update` が必要です。カスタム環境は元の管理ツールを使用してください。システムパッケージの追加をカスタム環境の更新として扱いません。

実際の Python venv/Conda を検出した場合は、グローバル環境を変更しません。`--venv` は DNF を使わず新しい Python 仮想環境を作成します。Python と親ディレクトリが必要で、保存先は新規でなければなりません。既存 venv の上書きや更新は行いません。

## 開発パッケージの取得元

```bash
rockdev source list
rockdev source show pip
rockdev source check pip tuna
rockdev source use pip tuna
rockdev source use npm npmmirror
rockdev source use maven aliyun
rockdev source use go goproxy.cn
rockdev source restore BACKUP_ID
```

pip、npm、Maven、Go 用の HTTPS 設定を用意しています。対象は実行中のユーザーです。sudo 使用時は root の設定となり、sudo 前のユーザーを自動選択しません。保存先とバックアップ ID は予定に表示します。

無関係の設定や npm 認証情報を維持し、Go のチェックサムデータベースも無効化しません。既存 Maven central/ワイルドカードのミラーはプライベートリポジトリに関係するため、自動上書きを拒否します。設定を上書きする環境変数は名前だけ表示します。

pip は明示的な PIP_CONFIG_FILE または設定済み仮想環境の pip.conf を優先し、それ以外はユーザー設定を使用します。npm userconfig と GOENV にも対応します。復元時はバックアップと現在の内容を確認し、手動編集された設定の上書きを拒否します。バックアップのハッシュは破損検出用であり、同じユーザーが全体を書き換えて再計算する攻撃への認証ではありません。

## 設定によるデプロイ

```bash
rockdev profile export --file dev.json python java go
rockdev profile validate dev.json
rockdev profile apply dev.json --dry-run
sudo rockdev profile apply dev.json
rockdev profile apply examples/user-sources.json
```

設定はバージョン付き JSON で、シェルコマンドは実行しません。不明な項目、重複や大文字小文字違いのキー、不明な環境や取得元は実行前に拒否します。

```json
{
  "schema": 1,
  "environments": ["python", "go"],
  "update": false,
  "sources": [{"tool": "go", "provider": "goproxy.cn"}]
}
```

`update` の既定値は false です。更新する場合は設定に true を明記してください。`profile apply --yes` は更新方針を変えません。システムとユーザーの設定を sudo でまとめて適用すると、取得元も root 用になります。必要に応じて分けて適用してください。

Python と絶対パスの `venv` を指定すると新規環境を作成します。同じ設定内の pip 取得元は、その venv の pip.conf に保存します。親ディレクトリが必要で、既存の保存先は使えません。`examples/` のパスは使用前に変更してください。

途中で失敗した場合は停止します。完了済みのインストールは自動で取り消しません。ログとバックアップを保管してください。

## オフラインパッケージの入出力

```bash
sudo rockdev repo create --dir /srv/rockdev/python python3 python3-pip
rockdev bundle export --from /srv/rockdev/python --file "$HOME/python.tar.gz"
rockdev bundle import --file "$HOME/python.tar.gz" --dir "$HOME/offline-python"
sudo dnf --setopt=reposdir="$HOME/offline-python" --disablerepo='*' --enablerepo=rockdev-local --setopt=rockdev-local.gpgcheck=1 install python3 python3-pip
```

インポートはリポジトリを準備するだけです。対象ディレクトリを指定し、他のリポジトリを無効にするインストールコマンドを表示します。

保存先は新規に限ります。Rocky の完全なバージョンとアーキテクチャ、一覧、サイズ、SHA256、信頼済み RPM 署名を確認します。パスの逸脱、リンク、重複、隠れた追記アーカイブは拒否します。ハッシュは整合性、RPM 署名はパッケージの出所確認に使用します。対象 OS には対応する信頼済み RPM 鍵が必要で、不明な鍵を自動登録しません。

ストリーム処理を使用します。展開量は最大 4 GiB、ファイルは 10000 個、マニフェストと各メタデータは 8 MiB までです。全署名の確認後にリポジトリを公開します。小バージョン間でも依存関係が異なるため、Rocky の完全なバージョン一致を要求します。

## 対象範囲

中国語・英語・日本語の表示と各言語が既定のビルドを継続します。複数バージョン管理、全量ミラーサーバー、自動アンインストールは含みません。[1.0 テスト結果](test-results-1.0.ja.md)を参照してください。0.1 の履歴は元の実施日のまま保管します。
