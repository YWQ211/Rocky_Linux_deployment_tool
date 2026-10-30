# Rockdev 0.1

中文 · [English](README.en.md) · [日本語](README.ja.md)

运行在 Rocky Linux 上的 Go 命令行工具，用于系统检查、镜像切换、本地 RPM 仓库和开发环境安装。

## 运行

使用对应 CPU 架构的 Linux 程序。运行不需要 Go，只有从源码编译时才需要。

```bash
chmod +x rockdev-linux-amd64
./rockdev-linux-amd64
./rockdev-linux-amd64 doctor
./rockdev-linux-amd64 --lang en
./rockdev-linux-amd64 --lang ja
```

`--lang zh|en|ja` 选择中文、英文或日文，也可以设置 `ROCKDEV_LANG`。命令名称保持一致，DNF 等系统程序的原始输出不翻译。

构建脚本还会生成默认英文的 `rockdev-linux-amd64-en` 和默认日文的 `rockdev-linux-amd64-ja`，ARM64 也有对应文件。

## 命令

| 命令 | 用途 |
|---|---|
| `doctor` | 显示系统、架构、root 权限和 DNF 状态 |
| `mirror list` | 列出官方、阿里云、清华 TUNA、科大镜像 |
| `mirror check [名称]` | 检查当前系统对应的仓库元数据 |
| `mirror use <名称或auto>` | 备份配置、切换镜像、刷新缓存 |
| `mirror restore <备份ID>` | 恢复软件源配置 |
| `repo create --dir <路径> <包名...>` | 下载 RPM 及依赖，校验签名并生成本地仓库 |
| `env list` | 显示环境预设及 PATH 中的命令 |
| `env install <环境...>` | 安装开发工具并检查命令 |
| `version` | 显示版本 |

不带参数时打开菜单。修改操作需要 root 并提示确认；`--yes` 跳过确认。`--dry-run` 只展示计划，不联网、不写文件、不执行安装。

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

## 文件与限制

- 日志：`/var/lib/rockdev/logs/`；软件源备份：`/var/lib/rockdev/backups/`。
- 换源仅修改已有 BaseOS、AppStream、extras 段，保留其他仓库。缓存刷新失败会恢复原文件。
- 恢复前校验文件摘要，遇到手动编辑的配置会停止。
- 本地仓库是所选 RPM 的集合，使用 `module_hotfixes=1`，客户端需信任对应 RPM 公钥。
- 安装中断不自动卸载已有软件。不包含 HTTP 仓库服务或 npm/pip/Maven/Go 模块镜像。
- 代码适配 Rocky 8/9/10、x86_64/aarch64，实际验证范围见[测试结果](docs/test-results.md)。

## 编译

私有仓库克隆需要 GitHub 认证，不要把令牌写入 URL。

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
go test ./...
go vet ./...
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev
```

要求 Go 1.22 或更高版本。Windows 可使用 `scripts/build.ps1`。详细操作见 [VM 手动运行说明](docs/vm-quickstart.md)。
