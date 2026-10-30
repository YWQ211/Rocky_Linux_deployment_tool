# VM 手动运行说明

目标：先获取代码或预编译程序，再由你登录 VM 自行运行功能。克隆和拷贝程序不会自动安装软件，也不会修改软件源。

## 获取代码

当前仓库为私有，需要给 Git 配置已有权限的 GitHub SSH 密钥或使用 HTTPS 令牌。不要把令牌写进仓库 URL、脚本或提交文件。GitHub 不接受 Linux 用户密码作为 Git HTTPS 密码。

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
```

若 VM 没有 GitHub 认证，HTTPS 克隆会失败或要求用户名/令牌；这不是工具的安装功能失败。此时可以先使用单独拷贝到 VM 的 Linux 程序验证功能，GitHub 访问配置完成后再克隆。

## 有 Go 时从源码运行

```bash
go version
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev --help
./dist/rockdev doctor
./dist/rockdev env install python go --dry-run
./dist/rockdev
```

Go 只用于编译工具，已编译的 rockdev 本身不依赖系统 Go。没有 Go 时不必先为了运行工具安装开发环境，使用下节的预编译文件。

## 使用 Linux 预编译程序

在含有 `rockdev-linux-amd64` 文件的目录中（ARM64 使用 arm64 文件）：

```bash
chmod +x rockdev-linux-amd64
./rockdev-linux-amd64 --help
./rockdev-linux-amd64 doctor
./rockdev-linux-amd64 env install python go --dry-run
./rockdev-linux-amd64
```

## 由你手动验证三项功能

以下命令里的 `rockdev` 换成实际文件路径，如 `./rockdev-linux-amd64` 或 `./dist/rockdev`。

```bash
# 系统检查（只读）
rockdev doctor

# 镜像检查（只访问网络）
rockdev mirror list
rockdev mirror check

# 执行换源（需要 sudo，执行前展示计划）
sudo rockdev mirror use tuna

# 安装开发环境（需要 sudo）
sudo rockdev env install python java go node build

# 创建按需 RPM 仓库（需要 sudo、网络和磁盘空间）
sudo mkdir -p /srv/rockdev
sudo rockdev repo create --dir /srv/rockdev/python python3 python3-pip
```

需要恢复软件源时，使用换源输出的真实备份 ID：`sudo rockdev mirror restore <备份ID>`。日志位于 `/var/lib/rockdev/logs/`。

第一次建议使用 VM 快照；完整验收见 [vm-acceptance.md](vm-acceptance.md)。Rocky 版本和真实 DNF 安装效果须通过 VM 实测确认，源码测试通过不等同于安装验证通过。
