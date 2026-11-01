# Rockdev 1.0.0

中文 · [English](release-1.0.en.md) · [日本語](release-1.0.ja.md)

## 环境检查与更新

```bash
rockdev env check
rockdev env check python java go
rockdev env install python go
sudo rockdev env install python go --update
rockdev env install python --venv /home/your-user/project/.venv
```

环境检查显示命令路径、版本、重复 PATH 候选、JAVA_HOME 不一致和 Python 隔离状态。它会执行版本查询，不能代替应用项目测试。

Go 检查子进程使用 `GOTOOLCHAIN=local`，避免自动下载工具链；不改变调用者的环境变量。

已有环境默认保留。交互模式逐项询问是否更新；`--yes` 只跳过执行确认，不授权升级。`--update` 才允许更新已有的系统运行环境。自定义路径下的运行环境需通过原管理器更新，程序不会把新装系统包误报成自定义环境已升级。

检测到真实 Python venv/Conda 时保留隔离环境，不更新全局包。`--venv` 只创建新的 Python venv，不调用 DNF；系统须先有 Python，目标父目录须存在，目标目录须不存在。不提供已有 venv 的覆盖或自动升级。

## 开发包源

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

支持 pip、npm、Maven、Go 的预设 HTTPS 源。默认配置当前用户；使用 sudo 会配置 root，不会自动配置 sudo 前的用户。计划会显示实际文件路径和备份 ID。

保留无关配置和 npm 凭据，Go 校验数据库不被禁用。已有 Maven central 或通配镜像可能涉及私有仓库，程序会拒绝自动覆盖，交由使用者处理。环境变量可能覆盖文件配置，`source show/use` 会提示变量名，不输出其值。

pip 优先使用显式 `PIP_CONFIG_FILE` 或已配置虚拟环境的 pip.conf，否则使用用户配置目录；npm 支持 userconfig，Go 支持 GOENV。恢复会核对备份元数据与当前文件，手动编辑后的配置不会被覆盖。备份摘要用于检测损坏，不是抵御同用户恶意重写的认证机制。

## 配置清单

```bash
rockdev profile export --file dev.json python java go
rockdev profile validate dev.json
rockdev profile apply dev.json --dry-run
sudo rockdev profile apply dev.json
rockdev profile apply examples/user-sources.json
```

清单使用版本化 JSON，不执行 shell。完整验证通过后才执行，拒绝未知字段、重复键、大小写变体及未知环境/包源。

```json
{
  "schema": 1,
  "environments": ["python", "go"],
  "update": false,
  "sources": [{"tool": "go", "provider": "goproxy.cn"}]
}
```

`update` 默认 false。允许更新必须在清单里写 true，`profile apply --yes` 不会改变这一策略。混合系统安装与用户包源的清单若用 sudo 执行，包源也属于 root；建议系统环境与用户配置分别应用。

包含 Python 和绝对路径 `venv` 时创建独立环境；同一清单里的 pip 包源写入该 venv 的 pip.conf。目标父目录须已存在，已有环境不会被覆盖。示例见 `examples/`，使用前修改示例路径。

执行在首个失败任务处停止，已完成的软件安装不会自动撤销。计划、备份及日志保留用于排查。

## 离线包导出、导入

```bash
sudo rockdev repo create --dir /srv/rockdev/python python3 python3-pip
rockdev bundle export --from /srv/rockdev/python --file "$HOME/python.tar.gz"
rockdev bundle import --file "$HOME/python.tar.gz" --dir "$HOME/offline-python"
```

导入只准备仓库，不自动安装包。使用导入后给出的命令，显式指定仓库目录、禁用其他仓库并启用本地源，例如：

```bash
sudo dnf --setopt=reposdir="$HOME/offline-python" --disablerepo='*' --enablerepo=rockdev-local --setopt=rockdev-local.gpgcheck=1 install python3 python3-pip
```

导出与导入目标都必须是新文件/新目录。导入检查 Rocky 完整版本和架构、文件清单、大小、SHA256 和 RPM 可信签名，拒绝路径穿越、符号链接、硬链接、重复及隐藏的附加归档。哈希用于完整性，RPM 签名用于包来源校验；目标系统须已信任对应 RPM 公钥，程序不会导入未知钥匙。

归档按流处理，解压总量最多 4 GiB、文件最多 10000 个，清单和单份元数据最多 8 MiB。GPG 校验全部通过后才发布仓库。不同系统小版本的依赖可能不兼容，因此即使同属 Rocky 10，也要求版本一致。

## 版本边界

仍提供中文、英文、日文界面和独立默认语言构建。当前更新不包含多版本运行时管理、全量镜像服务器或自动卸载。测试结果见 [1.0 测试报告](test-results-1.0.md)；0.1 历史结果仍保留原日期。
