# VM での実行手順

[中文](vm-quickstart.md) · [English](vm-quickstart.en.md) · 日本語

## ソースの取得

非公開リポジトリのため、アクセス権のある GitHub SSH 鍵または HTTPS トークンが必要です。トークンを URL、スクリプト、コミットに含めないでください。Linux のログインパスワードは GitHub 認証には使用できません。

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
```

VM に GitHub 認証を設定していない場合は、設定が完了するまで別途提供された Linux バイナリを使用できます。

## ソースからビルド

Go 1.22 以降が必要です。ビルド済みプログラムの実行には Go は不要です。

```bash
go version
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev --lang ja --help
./dist/rockdev --lang ja doctor
./dist/rockdev --lang ja env install python go --dry-run
./dist/rockdev --lang ja
```

## ビルド済みバイナリを使用

CPU に合ったファイルを選択してください。ARM64 ではファイル名の `amd64` を `arm64` に置き換えます。

```bash
chmod +x rockdev-linux-amd64
./rockdev-linux-amd64 --lang ja --help
./rockdev-linux-amd64 --lang ja doctor
./rockdev-linux-amd64 --lang ja env install python go --dry-run
./rockdev-linux-amd64 --lang ja
```

`rockdev-linux-amd64-ja` は日本語が既定です。すべてのビルドで `--lang zh|en|ja` を指定でき、`ROCKDEV_LANG` でも既定の言語を設定できます。DNF など外部ツールの出力は翻訳しません。

## 各機能を試す

以下の `rockdev` は、`./rockdev-linux-amd64` や `./dist/rockdev` など実際のパスに置き換えてください。sudo 使用時も `--lang ja` を指定すると日本語で表示します。

```bash
# 読み取り専用のシステム確認
rockdev --lang ja doctor

# 設定を変更せずにミラーを確認
rockdev --lang ja mirror list
rockdev --lang ja mirror check

# 現在利用可能なミラーへ変更
sudo rockdev --lang ja mirror use auto

# 開発環境をインストール
sudo rockdev --lang ja env install python java go node build

# ローカル RPM リポジトリを作成（保存先は新しいディレクトリ）
sudo mkdir -p /srv/rockdev
sudo rockdev --lang ja repo create --dir /srv/rockdev/python python3 python3-pip
```

設定を復元するには、ミラー変更時に表示されたバックアップ ID を使います。

```bash
sudo rockdev --lang ja mirror restore <backup-id>
```

ログは `/var/lib/rockdev/logs/` に保存されます。変更操作は実行前に確認します。`--yes` は確認を省略し、`--dry-run` は通信、ファイル変更、インストールを行わずに手順だけを表示します。

初回の操作前に VM スナップショットを取得してください。検証済みの環境、確認項目、制限は[テスト結果](test-results.ja.md)を参照してください。
