# Rockdev 0.1.0

Go 编写的 Rocky Linux CLI：系统检查、国内镜像切换、按需 RPM 本地仓库，以及开发环境安装。无第三方 Go 依赖。

**状态：本机自动测试与 Linux 交叉编译；尚未在 Rocky VM 执行部署。** Rocky 8/9/10 和 x86_64/aarch64 是代码的候选适配范围，不代表已完成兼容认证。

## 快速使用

项目仓库：[YWQ211/Rocky_Linux_deployment_tool](https://github.com/YWQ211/Rocky_Linux_deployment_tool)。当前为私有仓库，克隆需要有访问权限的 GitHub SSH 密钥或 HTTPS 令牌；Linux 登录密码不能用作 GitHub 密码。

从源码构建（需要预先安装 Go 1.22 或更高版本）：

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev doctor
./dist/rockdev                     # 打开交互菜单
```

Git 仓库提交源码、测试和文档，`dist/`、编译缓存和本地工具链不提交。没有 Go 时，可使用另行提供的 Linux 预编译程序。详细步骤见 [VM 手动运行说明](docs/vm-quickstart.md)。

将 `dist/rockdev-linux-amd64` 传到 x86_64 VM（ARM64 使用另一个文件）：

```bash
chmod +x rockdev-linux-amd64
./rockdev-linux-amd64 --help
./rockdev-linux-amd64 doctor
./rockdev-linux-amd64                     # 交互菜单
./rockdev-linux-amd64 mirror check
./rockdev-linux-amd64 env install python go --dry-run
sudo ./rockdev-linux-amd64 mirror use auto
sudo ./rockdev-linux-amd64 env install python java go node build
```

需要加入 PATH 时：

```bash
sudo install -m 0755 rockdev-linux-amd64 /usr/local/bin/rockdev
rockdev --help
```

修改操作展示计划，默认需要输入 `y`；`--yes` 用于无人值守脚本。`--dry-run` 不联网、不写文件、不执行子进程。真实操作要求 root、Linux、Rocky 和 DNF。退出码 0 表示成功或用户取消；失败为 1。Ctrl+C / SIGTERM 会取消当前操作；已完成的软件安装不会自动撤销。

## 镜像

```bash
rockdev mirror list
rockdev mirror check tuna
sudo rockdev mirror use aliyun
sudo rockdev mirror use auto --yes
sudo rockdev mirror restore backup-20261030-120000.123456789
```

候选：Rocky 官方、阿里云、清华 TUNA、中国科大。检测当前系统主版本和架构的 BaseOS/AppStream/extras 元数据。`auto` 从本次可用候选中选择元数据响应最快的站点；这不是大文件带宽测速。每个站点检测最多 8 秒。

只修改现有 `.repo` 中的 `[baseos]`、`[appstream]`、`[extras]`，保留其余仓库和 GPG 配置。未创建不存在的仓库，不自动启用 CRB/EPEL。修改前保存原文件；刷新失败恢复文件。重复切换到相同 URL 不重复备份。备份位于 `/var/lib/rockdev/backups/<ID>`；恢复会校验摘要并拒绝覆盖手动编辑。恢复失败可能已经恢复部分文件，须保留备份。进程被强制结束或系统断电时，使用日志中的备份 ID 手动恢复；不声称跨多个文件的断电事务保证。

镜像可用性依目标 VM 网络实时检测，不保证每个站点始终包含每个版本。

## 按需 RPM 本地仓库

```bash
sudo mkdir -p /srv/rockdev
rockdev repo create --dir /srv/rockdev/python python3 python3-pip --dry-run
sudo rockdev repo create --dir /srv/rockdev/python python3 python3-pip
```

父目录须存在，目标目录须不存在。流程：安装 `dnf-plugins-core`/`createrepo_c`，下载指定包及全部解析依赖（包括已安装依赖），逐个检查 RPM 签名，生成元数据，发布新目录。失败会清理临时下载目录；先前安装的仓库工具包会保留。空间不足会通过具体步骤错误退出；目前不预估完整下载体积、不自动清理磁盘。

生成 `<目录>/rockdev-local.repo`，默认 `enabled=0`。可显式拷贝并验证使用：

```bash
sudo cp /srv/rockdev/python/rockdev-local.repo /etc/yum.repos.d/rockdev-local.repo
sudo dnf --disablerepo='*' --enablerepo=rockdev-local makecache
sudo dnf --disablerepo='*' --enablerepo=rockdev-local install python3 python3-pip
```

这是所选 RPM 的仓库，不是全量镜像。为容纳已选择的模块 RPM，配置 `module_hotfixes=1`，不保留完整 modulemd；不提供模块流切换。请在相同系统版本、架构的干净 VM 上验证依赖闭包，再用于离线环境。客户端须信任对应发行版 RPM 公钥；工具不会导入未知公钥。首次发布后不自动更新仓库，更新用新目录。若移动仓库，须调整配置中的 `baseurl`。

不包含 HTTP 仓库服务器、全量同步、npm/pip/Maven/Go 模块私有镜像，也不配置这些语言的国内下载代理。

## 开发环境

```bash
rockdev env list
sudo rockdev env install python go
sudo rockdev env install java node build --yes
```

| 预设 | 软件包 | 安装后检查 |
|---|---|---|
| python | python3、python3-pip | Python、venv 导入、pip |
| java | JDK 17（Rocky 10 为 JDK 21）、Maven | java、javac、mvn 版本 |
| go | golang | go version |
| node | nodejs、npm | Node 执行脚本、npm 版本 |
| build | Git、GCC、G++、make、pkg-config | Git、GCC、make 版本 |

使用当前已启用 DNF 仓库的可用版本。包缺失、冲突或未启用的模块流会明确失败，不偷偷切换来源、覆盖系统 Python 或执行全系统升级。DNF 可能更新已请求包及依赖；Java alternatives 等由发行版包脚本管理。`env list` 显示主命令是否在 PATH，并非完整安装健康报告。不提供任意语言版本选择、自动配置 shell、卸载和包事务回滚。

Python 项目环境可自行创建：

```bash
python3 -m venv .venv
. .venv/bin/activate
```

## 日志与执行

- 日志：`/var/lib/rockdev/logs/run-*.log`，含执行计划、子命令输出和最终状态。
- 全局进程锁阻止多个 rockdev 修改操作并发；进程退出后内核释放锁。
- 外部命令使用参数数组调用，不经 shell。子进程使用英文 locale 以稳定解析 RPM 签名结果。
- 安装操作对全局及各仓库强制开启 GPG 包校验；本地仓库额外拒绝无签名/缺公钥的 RPM。
- 所有命令在首个失败步骤停止。重跑环境安装由 DNF 判断已有软件包；重复创建仓库须用新目录。

## 编译与测试

Go 1.22 或更高版本：

```bash
go test ./...
go vet ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o dist/rockdev-linux-amd64 ./cmd/rockdev
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o dist/rockdev-linux-arm64 ./cmd/rockdev
```

Windows PowerShell 使用 `./scripts/build.ps1 -GoPath <go.exe路径>`，生成 Linux 双架构、Windows 冒烟测试程序、源码 ZIP 及 SHA256SUMS。Windows 程序仅用于菜单/帮助/只读清单验证，不支持系统修改。

测试使用临时目录、本地 HTTP 服务器和假命令运行器，覆盖系统限制、元数据错误、配置回退、备份恢复、重复换源、签名拒绝、依赖下载参数、失败停止和预览无副作用。Linux 文件锁与真实 DNF/RPM 行为仍需 VM 验收，见 [VM 验收清单](docs/vm-acceptance.md)。

实现依据：[DNF download](https://dnf-plugins-core.readthedocs.io/en/latest/download.html)、[DNF 模块与 hotfix 仓库](https://dnf.readthedocs.io/en/stable/modularity.html)、[DNF 命令选项](https://dnf.readthedocs.io/en/latest/command_ref.html)、[Rocky 本地仓库方案](https://docs.rockylinux.org/gemstones/setup_local_repo/)。
