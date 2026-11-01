# Rockdev 0.1

[中文](README.md) · [English](README.en.md) · 日本語

ドキュメント：[VM 操作手順](docs/vm-quickstart.ja.md) · [テスト結果](docs/test-results.ja.md)

Rocky Linux 向けの Go 製 CLI ツールです。システム確認、パッケージミラーの変更、ローカル RPM リポジトリの作成、開発環境のインストールを行います。

## 実行

CPU に合った Linux バイナリを使用してください。Go が必要なのはソースからビルドするときだけです。

```bash
chmod +x rockdev-linux-amd64
./rockdev-linux-amd64 --lang ja
./rockdev-linux-amd64 --lang ja doctor
```

`--lang zh|en|ja` で中国語・英語・日本語を選択できます。既定の言語は `ROCKDEV_LANG` でも指定できます。コマンド名は共通です。DNF など外部ツールの出力は翻訳しません。

ビルドスクリプトは、日本語が既定の `rockdev-linux-amd64-ja`、英語が既定の `-en`、および ARM64 版も生成します。

```bash
export ROCKDEV_LANG=ja
./rockdev-linux-amd64 env install python go --dry-run
```

## コマンド

| コマンド | 内容 |
|---|---|
| `doctor` | OS、アーキテクチャ、root 権限、DNF の有無を表示 |
| `mirror list` | Rocky 公式、阿里雲、清華大学 TUNA、中国科学技術大学の一覧 |
| `mirror check [name]` | 現在の OS に対応するメタデータを確認 |
| `mirror use <name\|auto>` | 設定を保存してミラーを変更し、キャッシュを更新 |
| `mirror restore <backup-id>` | 保存したリポジトリ設定を復元 |
| `repo create --dir <path> <packages...>` | RPM と依存関係を取得し、署名とメタデータを処理 |
| `env list` | 開発環境の一覧と PATH 上のコマンドを表示 |
| `env install <presets...>` | 開発環境をインストールしてコマンドを確認 |
| `version` | バージョンを表示 |

引数なしで対話メニューを開きます。変更には root 権限と実行確認が必要です。`--yes` は確認を省略します。`--dry-run` は予定を表示するだけで、通信、ファイル変更、インストールは行いません。

## 使用例

```bash
sudo ./rockdev-linux-amd64 --lang ja mirror use auto
sudo ./rockdev-linux-amd64 --lang ja env install python java go node build
sudo mkdir -p /srv/rockdev
sudo ./rockdev-linux-amd64 --lang ja repo create --dir /srv/rockdev/python python3 python3-pip
```

親ディレクトリは事前に作成し、保存先には新しいディレクトリを指定してください。生成される `rockdev-local.repo` は初期状態で無効です。使用する場合は設定ファイルをコピーし、対象の操作で有効化します。

```bash
sudo cp /srv/rockdev/python/rockdev-local.repo /etc/yum.repos.d/
sudo dnf --disablerepo='*' --enablerepo=rockdev-local install python3 python3-pip
```

## 開発環境

| 指定名 | インストール内容 |
|---|---|
| `python` | Python 3、pip。venv の利用可否も確認 |
| `java` | JDK 17（Rocky 10 は JDK 21）、Maven |
| `go` | Go |
| `node` | Node.js、npm |
| `build` | Git、GCC、G++、make、pkg-config |

有効な DNF リポジトリのバージョンを使用します。言語のバージョン指定やシステム Python の置き換えは行いません。Python のプロジェクト環境には `python3 -m venv .venv` を使用できます。

## 保存先と制限

- ログ：`/var/lib/rockdev/logs/`。設定のバックアップ：`/var/lib/rockdev/backups/`。
- ミラー変更の対象は既存の BaseOS、AppStream、extras です。他のリポジトリは維持します。キャッシュ更新に失敗した場合は元のファイルを復元します。
- 復元時はハッシュを確認し、手動編集された設定の上書きを防ぎます。
- ローカルリポジトリは `module_hotfixes=1` を使う選択済み RPM の集合です。完全なディストリビューションミラーではありません。クライアントには対応する信頼済み RPM 鍵が必要です。
- 途中で失敗しても、インストール済みパッケージの自動削除は行いません。HTTP 配信や npm/pip/Maven/Go モジュールのミラーは含みません。
- コードの対応範囲は Rocky 8/9/10、x86_64/aarch64 です。実際の検証範囲は[テスト結果](docs/test-results.ja.md)を参照してください。

## ビルド

非公開リポジトリのため、クローンには GitHub 認証が必要です。URL にトークンを含めないでください。

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
go test ./...
go vet ./...
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev --lang ja
```

Go 1.22 以降が必要です。Windows では `scripts/build.ps1` を利用できます。[VM での操作手順](docs/vm-quickstart.ja.md)も参照してください。
