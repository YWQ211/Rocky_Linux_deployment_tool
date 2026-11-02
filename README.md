# Rockdev 1.0.1

## 文档语言 / Documentation / ドキュメント

| 简体中文 | English | 日本語 |
|---|---|---|
| [**完整中文说明（首页）**](README.md) | [**Read the full English guide →**](README.en.md) | [**日本語の説明はこちら →**](README.ja.md) |

三种语言均有完整说明。GitHub 首页保留中文，请通过上方入口切换。

文档：[完整功能说明](docs/release-1.0.md) · [操作指南](docs/vm-quickstart.md) · [1.0.1 复测报告](docs/test-results-1.0.1.md)

1.0.1 修复 Rocky 8 系统 Python 路径误判。[补丁说明与复测](docs/release-1.0.1.md)。

运行在 Rocky Linux 上的 Go 命令行工具，用于系统检查、镜像切换、本地 RPM 仓库和开发环境安装。

首次使用请先看[源码编译步骤](#编译)。克隆获得的是源码，不会自动安装工具或下载开发环境。

## 运行

以下示例适用于已经拿到的编译程序。使用对应 CPU 架构的 Linux 程序；运行不需要 Go，源码编译才需要。目前 GitHub 尚未发布预编译 Release 附件，克隆后请按页末步骤构建。

```bash
chmod +x rockdev-linux-amd64
./rockdev-linux-amd64
./rockdev-linux-amd64 doctor
./rockdev-linux-amd64 --lang en
./rockdev-linux-amd64 --lang ja
```

`--lang zh|en|ja` 选择中文、英文或日文，也可以设置 `ROCKDEV_LANG`。命令名称保持一致，DNF 等系统程序的原始输出不翻译。

构建脚本还会生成默认英文的 `rockdev-linux-amd64-en` 和默认日文的 `rockdev-linux-amd64-ja`，ARM64 也有对应文件。

## 官方源与不同地区的使用

安装开发环境和创建 RPM 仓库时，程序使用机器当前启用的 DNF 仓库，不会自动强制切换到中国镜像。已有的官方 mirrorlist 配置可以继续使用。只有执行 `mirror use` 才会切换系统源；`--lang en` 仅改变界面语言，不改变下载源。

| 下载对象 | 官方提供者 ID | 已内置的官方地址 | 其他预设 |
|---|---|---|---|
| Rocky RPM | `mirror ... official` | `https://dl.rockylinux.org/pub/rocky` | `aliyun`、`tuna`、`ustc` |
| Python / pip | `source ... pip official` | `https://pypi.org/simple/` | `tuna`、`aliyun` |
| Node / npm | `source ... npm official` | `https://registry.npmjs.org/` | `npmmirror` |
| Maven | `source ... maven official` | `https://repo.maven.apache.org/maven2/` | `aliyun` |
| Go 模块 | `source ... go official` | `https://proxy.golang.org` | `goproxy.cn` |

需要显式使用官方源时：

```bash
rockdev mirror check official
sudo rockdev mirror use official
rockdev source use pip official
rockdev source use npm official
rockdev source use maven official
rockdev source use go official
```

这里有两类配置：`mirror` 管理 Rocky 的系统 RPM 源，`source` 管理当前用户的开发包配置。切换 pip 源不会改变 DNF 源。

`mirror use auto` 只在上述四个系统源中检查元数据可用性并比较检查耗时，不是全球镜像发现、RPM 下载带宽测试或持续故障切换。`official` 是固定官方地址，与恢复系统原始 mirrorlist 不是一回事。当前没有自定义 URL 或按国家分组功能；某个镜像可用不代表对所有地区都最快。

## 命令

1.0 新增环境检查、开发包源、JSON 清单部署、离线包导入导出。完整选项与限制见 [1.0 使用说明](docs/release-1.0.md)。

```bash
rockdev env check
rockdev env install python --venv /home/your-user/project/.venv
sudo rockdev env install python java go --update
rockdev source use pip tuna
rockdev profile apply examples/user-sources.json --dry-run
rockdev bundle export --from /srv/rockdev/python --file "$HOME/python.tar.gz"
rockdev bundle import --file "$HOME/python.tar.gz" --dir "$HOME/offline-python"
```

已有环境默认保留，交互时可选择更新；`--yes` 不授权更新，须明确选择或使用 `--update`。不完整的已有环境也先保留。自定义运行时使用原管理器更新；Python venv 单独创建和使用。包源配置属于当前用户，sudo 下属于 root。

| 命令 | 用途 |
|---|---|
| `doctor` | 显示系统、架构、root 权限和 DNF 状态 |
| `mirror list` | 列出官方、阿里云、清华 TUNA、科大镜像 |
| `mirror check [名称]` | 检查当前系统对应的仓库元数据 |
| `mirror use <名称或auto>` | 备份配置、切换镜像、刷新缓存 |
| `mirror restore <备份ID>` | 恢复软件源配置 |
| `repo create --dir <路径> <包名...>` | 下载 RPM 及依赖，校验签名并生成本地仓库 |
| `env list` | 显示环境预设及 PATH 中的命令 |
| `env install <环境...>` | 检查已有环境，按选择安装或更新 |
| `env check [环境...]` | 显示路径、版本和潜在冲突 |
| `source list/show/check/use/restore` | 配置和恢复开发包源 |
| `profile validate/export/apply` | 验证、导出和应用部署清单 |
| `bundle export/import` | 导出、验证和导入离线 RPM 仓库 |
| `version` | 显示版本 |

不带参数时打开菜单。系统换源、RPM 仓库创建、系统开发包安装需要 root；环境检查、用户包源配置、新 venv，以及有目录权限的离线包操作可由普通用户执行。清单权限取决于包含的任务。修改前显示计划并提示确认；`--yes` 跳过确认，但不授权更新已有环境。`--dry-run` 不联网、不写文件、不执行安装。

## 使用示例

```bash
./rockdev-linux-amd64 env install python go --dry-run
sudo ./rockdev-linux-amd64 mirror use auto
sudo ./rockdev-linux-amd64 env install python java go node build
sudo mkdir -p /srv/rockdev
sudo ./rockdev-linux-amd64 repo create --dir /srv/rockdev/python python3 python3-pip
```

本地仓库的父目录须存在，目标目录须为新目录。生成的 `rockdev-local.repo` 默认禁用，使用时显式启用：

```bash
sudo cp /srv/rockdev/python/rockdev-local.repo /etc/yum.repos.d/
sudo dnf --disablerepo='*' --enablerepo=rockdev-local install python3 python3-pip
```

## 开发环境

| 预设 | 内容 |
|---|---|
| `python` | Python 3、pip，检查 venv |
| `java` | JDK 17；Rocky 10 使用 JDK 21，附带 Maven |
| `go` | Go |
| `node` | Node.js、npm |
| `build` | Git、GCC、G++、make、pkg-config |

版本由当前 DNF 仓库决定，不提供任意版本切换，也不替换系统 Python。Python 项目环境可用 `python3 -m venv .venv` 创建。

## 各项功能怎样使用

1. **系统与环境检查**：先执行 `rockdev doctor` 判断系统和架构，再用 `rockdev env check python java go node` 查看已有命令的路径、版本、重复 PATH 候选、JAVA_HOME 不一致和 Python 隔离状态。这是部署前检查，不会修复项目依赖或自动改环境。
2. **系统镜像切换**：`mirror check` 先检查当前 Rocky 版本和架构所需的仓库；`mirror use` 备份、修改已有 BaseOS/AppStream/extras 段并刷新缓存，刷新失败会恢复原配置。保留输出的备份 ID，供 `mirror restore` 使用。
3. **本地 RPM 仓库**：`repo create` 下载指定包及依赖，即使依赖已经装在本机也纳入下载，校验签名后生成仓库元数据和默认禁用的 `.repo` 文件。适合一组机器复用同一批包，不是全量 Rocky 镜像或自动启动的 HTTP 服务。
4. **开发环境安装**：按预设安装 Python、Java、Go、Node 或编译工具。已有环境默认保留；显式 `--update` 才允许通过 DNF 更新系统运行时，自定义运行时须用原管理器。项目隔离可用 `rockdev env install python --venv "$HOME/project/.venv"`，父目录须存在、目标须为新目录；该命令不安装系统 Python，也不覆盖已有 venv。
5. **开发包源配置**：先用 `source show pip` 查看目标配置及环境变量覆盖提示，再用 `source use pip official` 等命令修改当前用户配置。保留无关配置，打印备份 ID；`source restore` 在摘要匹配时恢复，手动编辑后的文件不会被覆盖。sudo 会配置 root；已有私有 Maven central 或通配镜像会拒绝自动覆盖。
6. **配置清单部署**：`profile export --file dev.json python go` 生成清单；`profile validate dev.json` 校验；`profile apply dev.json --dry-run` 预览后再应用。JSON 可以声明环境、venv 和包源；默认 `update=false`，不执行任意 shell。任务遇到首个失败即停止，已安装软件不会自动卸载。
7. **离线导出与导入**：联网机器先 `repo create`，再 `bundle export`；把归档复制到目标机器后 `bundle import`。导入核对完整 Rocky 版本、CPU 架构、文件大小、SHA256 和可信 RPM 签名，只准备本地仓库，不自动安装。目标机器须已有对应可信公钥；不同小版本或架构的归档不能混用。

清单和离线安装的完整示例见[功能说明](docs/release-1.0.md)。以下命令只使用所准备的本地仓库，不修改全局仓库文件：

```bash
sudo dnf --setopt=reposdir="$HOME/offline-python" --disablerepo='*' --enablerepo=rockdev-local --setopt=rockdev-local.gpgcheck=1 install python3 python3-pip
```

## 文件与限制

- 系统操作日志：`/var/lib/rockdev/logs/`；系统软件源备份：`/var/lib/rockdev/backups/`。用户操作日志在 `~/.local/state/rockdev/logs/`，开发包源备份在 `~/.local/state/rockdev/sources/`。
- 换源仅修改已有 BaseOS、AppStream、extras 段，保留其他仓库。缓存刷新失败会恢复原文件。
- 恢复前校验文件摘要，遇到手动编辑的配置会停止。
- 本地仓库是所选 RPM 的集合，使用 `module_hotfixes=1`，客户端需信任对应 RPM 公钥。
- 安装中断不自动卸载已有软件。不包含 HTTP 仓库服务或 npm/pip/Maven/Go 模块镜像。
- 候选系统为 Rocky 8/9/10、x86_64/aarch64。1.0.1 在 Rocky 8.10、9.8 x86_64 各通过 59 项集成测试；ARM64 只交叉构建。Rocky 10.2 是 1.0.0 的历史运行验证，详见[最新测试报告](docs/test-results-1.0.1.md)。其他发行版没有系统部署支持。

## 编译

需要 Git 和 Go **1.22 或更高版本**，先用 `git --version`、`go version` 检查。DNF 提供的 Go 可能较旧，不满足要求时按 [Go 官方安装说明](https://go.dev/doc/install)准备编译器。

`git clone` 只复制源码；`dist/` 不纳入 Git，必须先构建可执行文件。当前仓库仍为私有，需要 GitHub 访问权限；将来公开后可以直接用下面的 HTTPS 地址克隆，无需仓库访问授权。不要把令牌写入 URL。克隆本身不会修改系统配置。

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
go test ./...
go vet ./...
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev
```

若希望在任意目录使用上文的 `rockdev` 命令，可选择安装到 PATH：

```bash
sudo install -m 0755 ./dist/rockdev /usr/local/bin/rockdev
rockdev --help
```

Windows 可用 `scripts/build.ps1` 交叉构建 Linux 程序；Windows 构建本身不代表支持在 Windows 部署 Rocky 环境。详细操作见 [VM 手动运行说明](docs/vm-quickstart.md)。

## 公开与许可证状态

当前尚无 `LICENSE`，也没有 GitHub Release 二进制附件。将仓库设为 Public 可以让别人获取源码；若作为开源项目发布，还需要维护者选定并加入许可证，说明使用、修改和再分发权限。参见 [GitHub 许可证说明](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/licensing-a-repository)。
