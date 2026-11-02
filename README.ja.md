# Rockdev 1.0.1

## ドキュメント / 文档语言 / Documentation

| 简体中文 | English | 日本語 |
|---|---|---|
| [**完整中文说明（首页）**](README.md) | [**Read the full English guide →**](README.en.md) | [**日本語の完全版 — このページ**](README.ja.md) |

3 言語とも完全な説明があります。リポジトリのトップページは中国語です。上記のリンクで切り替えてください。

ドキュメント：[機能の詳細](docs/release-1.0.ja.md) · [VM 操作手順](docs/vm-quickstart.ja.md) · [1.0.1 再検証結果](docs/test-results-1.0.1.ja.md)

1.0.1 で Rocky 8 のシステム Python 判定を修正しました。[変更点と再検証](docs/release-1.0.1.ja.md)。

Rocky Linux 向けの Go 製 CLI ツールです。システム確認、パッケージミラーの変更、ローカル RPM リポジトリの作成、開発環境のインストールを行います。

初めて利用する場合は[ビルド手順](#ビルド)から始めてください。クローンで取得するのはソースです。ツールや開発環境が自動でインストールされるわけではありません。

## 実行

以下は CPU に合った Linux 実行ファイルを入手済みの場合の例です。Go が必要なのはソースからビルドするときだけです。現在 GitHub Release にビルド済み添付ファイルはありません。クローン後は末尾の手順でビルドしてください。

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

## 公式の取得先と地域別の利用

開発環境のインストールと RPM リポジトリの作成は、そのマシンで有効な DNF リポジトリを使います。中国のミラーへ自動変更しません。既存の公式 mirrorlist 設定もそのまま使えます。システムの取得先を変更するのは `mirror use` 実行時だけです。`--lang ja` は表示言語のみを変更します。

| 対象 | 公式プロバイダーの指定 | 組み込み済みの公式 URL | ほかの設定 |
|---|---|---|---|
| Rocky RPM | `mirror ... official` | `https://dl.rockylinux.org/pub/rocky` | `aliyun`、`tuna`、`ustc` |
| Python / pip | `source ... pip official` | `https://pypi.org/simple/` | `tuna`、`aliyun` |
| Node / npm | `source ... npm official` | `https://registry.npmjs.org/` | `npmmirror` |
| Maven | `source ... maven official` | `https://repo.maven.apache.org/maven2/` | `aliyun` |
| Go モジュール | `source ... go official` | `https://proxy.golang.org` | `goproxy.cn` |

公式の取得先を明示的に指定する場合：

```bash
rockdev mirror check official
sudo rockdev mirror use official
rockdev source use pip official
rockdev source use npm official
rockdev source use maven official
rockdev source use go official
```

`mirror` は Rocky のシステム RPM、`source` は現在のユーザーの開発パッケージ設定を管理します。pip の取得先を変更しても DNF の設定は変わりません。

`mirror use auto` は組み込みのシステム取得先 4 種でメタデータの到達性と確認時間を比較します。世界中のミラーの検出、RPM 帯域の測定、継続的な障害切替ではありません。`official` は固定の公式 URL で、元の mirrorlist の復元とは異なります。任意 URL や国別グループの機能はありません。到達可能なミラーがどの地域でも最速とは限りません。

## コマンド

1.0 では環境診断、取得元の設定、JSON によるデプロイ、オフラインパッケージ入出力を追加しました。詳細は [1.0 操作説明](docs/release-1.0.ja.md)を参照してください。

```bash
rockdev env check
rockdev env install python --venv /home/your-user/project/.venv
sudo rockdev env install python java go --update
rockdev source use pip official
rockdev profile export --file dev.json python go
rockdev profile validate dev.json
rockdev profile apply dev.json --dry-run
rockdev bundle export --from /srv/rockdev/python --file "$HOME/python.tar.gz"
rockdev bundle import --file "$HOME/python.tar.gz" --dir "$HOME/offline-python"
```

不完全な環境も含め、既存環境は更新を選択するまで維持します。`--yes` は更新の許可ではありません。`--update` または対話で明示してください。カスタム環境は元の管理ツールを使い、Python venv は独立して扱います。取得元は現在のユーザー用で、sudo 使用時は root 用です。

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
| `env check [環境...]` | パス、バージョン、競合候補を表示 |
| `source list/show/check/use/restore` | 取得元の設定と復元 |
| `profile validate/export/apply` | JSON 設定の検証、保存、適用 |
| `bundle export/import` | オフライン RPM リポジトリの入出力 |

引数なしで対話メニューを開きます。システムのミラー変更、RPM リポジトリ作成、システムパッケージの導入は root 権限が必要です。診断、ユーザー用の取得先、新規 venv、アクセス権のある場所でのオフライン操作は一般ユーザーで実行できます。構成ファイルは含まれる作業によって権限が変わります。変更前に予定と確認を表示し、`--yes` は確認のみ省略します。既存環境の更新許可にはなりません。`--dry-run` は通信・ファイル変更・インストールを行いません。

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

## 各機能の使い方

1. **システムと環境の診断**：`rockdev doctor` で OS と CPU を確認し、`rockdev env check python java go node` でパス、バージョン、PATH の重複、JAVA_HOME の不一致、Python の分離状態を確認します。プロジェクトの依存関係を修復したり環境を書き換えたりする機能ではありません。
2. **システムミラー**：`mirror check` は現在の Rocky と CPU に対応するリポジトリを確認します。`mirror use` は既存の BaseOS/AppStream/extras 設定を保存して変更し、キャッシュを更新します。更新失敗時は元に戻します。表示されたバックアップ ID は `mirror restore` に使います。
3. **ローカル RPM リポジトリ**：`repo create` は指定した RPM と依存関係を取得します。取得元マシンに導入済みの依存パッケージも含めます。署名検証後にメタデータと初期無効の `.repo` を作成します。複数マシンで同じパッケージを使うための集合であり、完全ミラーや HTTP サーバーではありません。
4. **開発環境の導入**：Python、Java、Go、Node、ビルドツールを選択します。既存環境は初期設定で維持し、`--update` 指定時のみ DNF によるシステム環境更新を許可します。独自環境は元の管理ツールを使います。プロジェクトの分離には `rockdev env install python --venv "$HOME/project/.venv"` を使えます。親フォルダーが必要で、作成先は新規に限ります。システム Python の導入や既存 venv の上書きは行いません。
5. **開発パッケージの取得先**：`source show pip` で対象ファイルと環境変数の優先設定を確認し、`source use pip official` などで現在のユーザーの設定を変更します。無関係の設定を保ち、バックアップ ID を表示します。`source restore` はハッシュを確認し、手動編集後の上書きを防ぎます。sudo 使用時は root 用です。既存の非公開 Maven central ミラーや全体ミラーは自動で置換しません。
6. **JSON 構成によるデプロイ**：`profile export --file dev.json python go` で作成し、`profile validate dev.json` で検証し、`profile apply dev.json --dry-run` で確認後に適用します。環境、venv、取得先を宣言できます。`update` の既定値は false で、任意の shell は実行しません。最初の失敗で停止し、導入済みソフトウェアは自動削除しません。
7. **オフライン入出力**：通信可能なマシンで `repo create`、続いて `bundle export` を実行し、アーカイブを転送して `bundle import` します。完全な Rocky バージョン、CPU、サイズ、SHA256、信頼済み RPM 署名を確認します。取り込みはリポジトリの準備のみで、パッケージを自動導入しません。対応する公開鍵が事前に信頼されている必要があります。小バージョンや CPU が異なる場合は共用できません。

構成とオフライン作業の詳細は[機能説明](docs/release-1.0.ja.md)を参照してください。以下はグローバルなリポジトリ設定を変更せず、準備したローカルリポジトリだけを使う例です。

```bash
sudo dnf --setopt=reposdir="$HOME/offline-python" --disablerepo='*' --enablerepo=rockdev-local --setopt=rockdev-local.gpgcheck=1 install python3 python3-pip
```

## 保存先と制限

- システム操作のログ：`/var/lib/rockdev/logs/`。システムミラーのバックアップ：`/var/lib/rockdev/backups/`。ユーザー操作のログは `~/.local/state/rockdev/logs/`、開発パッケージ取得先のバックアップは `~/.local/state/rockdev/sources/` です。
- ミラー変更の対象は既存の BaseOS、AppStream、extras です。他のリポジトリは維持します。キャッシュ更新に失敗した場合は元のファイルを復元します。
- 復元時はハッシュを確認し、手動編集された設定の上書きを防ぎます。
- ローカルリポジトリは `module_hotfixes=1` を使う選択済み RPM の集合です。完全なディストリビューションミラーではありません。クライアントには対応する信頼済み RPM 鍵が必要です。
- 途中で失敗しても、インストール済みパッケージの自動削除は行いません。HTTP 配信や npm/pip/Maven/Go モジュールのミラーは含みません。
- 対象候補は Rocky 8/9/10、x86_64/aarch64 です。1.0.1 は Rocky 8.10 と 9.8 の x86_64 で各 59 項目の統合テストが成功しました。ARM64 はクロスビルドのみ、Rocky 10.2 は 1.0.0 の過去の実行検証です。[最新テスト結果](docs/test-results-1.0.1.ja.md)を参照してください。ほかのディストリビューションのシステム導入には対応していません。

## ビルド

Git と **Go 1.22 以上**が必要です。先に `git --version`、`go version` を確認してください。DNF の Go が要件より古い場合は [Go 公式の導入手順](https://go.dev/doc/install)で適切なコンパイラーを用意してください。

`git clone` で取得するのはソースです。`dist/` は Git 管理外なので、実行前にビルドが必要です。現在のリポジトリは非公開でアクセス権が必要ですが、公開後は同じ HTTPS URL からリポジトリ権限なしでクローンできます。URL にトークンを含めないでください。クローンだけでシステム設定は変わりません。

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
go test ./...
go vet ./...
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev --lang ja
```

どのディレクトリからも `rockdev` を使いたい場合は、必要に応じて PATH 内に導入します。

```bash
sudo install -m 0755 ./dist/rockdev /usr/local/bin/rockdev
rockdev --help
```

Windows では `scripts/build.ps1` で Linux 向けにクロスビルドできます。Windows 版のビルドは、Windows 上で Rocky の環境構築ができることを意味しません。[VM での操作手順](docs/vm-quickstart.ja.md)も参照してください。

## 公開とライセンスの状態

現在 `LICENSE` はなく、GitHub Release にビルド済み添付ファイルもありません。Public にすると第三者がソースを取得できます。オープンソースとして公開するには、利用・変更・再配布の権限を示すライセンスを管理者が選んで追加する必要があります。[GitHub のライセンス説明](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/licensing-a-repository)を参照してください。
