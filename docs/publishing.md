# 发布与许可证

## 用户入口

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
./rockdev
```

不需要 Go。首次下载固定版本的 Linux amd64/arm64 程序，以仓库中的 SHA256 核验；后续使用每次校验的缓存。普通用户缓存位于 `~/.cache/rockdev/`（或绝对路径 XDG_CACHE_HOME 下），root 使用 `/var/cache/rockdev/`。下载失败、摘要不符、非 Rocky 或架构不支持时拒绝启动；不自动执行系统安装。

## 维护者发布

1. `LICENSE` 为 MIT，`2026 YWQ211` 为版权署名；二进制分发同时保留 `THIRD_PARTY_NOTICES`。
2. `release/VERSION` 与 Go CLI 版本一致。`release/SHA256SUMS` 固定两种架构的实际产物摘要。根入口在 Git 中须为 `100755`，`.gitattributes` 保持 LF。
3. 发布工作流固定 Go 工具链版本，运行 Go 测试、vet 和启动器回归，构建两架构程序。只有构建结果匹配仓库摘要才允许发布。
4. 测试通过后显式创建指向本次提交的新标签，再创建含五个附件的 draft Release，核验标签提交后发布。已有同版本标签或 Release 时拒绝覆盖；失败遗留的标签和 draft 需维护者检查后处理，不静默替换附件。
5. 主分支修改入口、版本、摘要或发布流程会触发构建，也可以在 Actions 手动运行。发布地址为 `releases/download/v<VERSION>/rockdev-linux-<arch>`。
6. 仓库仍为私有时，Release 也受访问权限限制。准备完成后由维护者确认改为 Public；许可证、Release 发布与仓库公开是三个不同动作。

发布新的程序版本时，需要一起更新版本、程序和对应摘要；不复用原版本号替换已发布字节。源码 ZIP 可由 GitHub 对应 tag 自动生成。无需把大体积二进制提交进 Git。

## 验证

`python3 scripts/test-launcher.py` 使用下载替身测试首次下载、缓存、摘要、参数转发、错误系统、并发和符号链接拒绝。它不会联网，也不替代真实 Release 下载验证。Windows 的符号链接用例可能因权限跳过，Linux 应全部执行。

公开后应在新的 Rocky 环境验证真实 `git clone`、首次下载、再次断网启动；测试后保留日志并回收容器。准备阶段的私有 Release 不可据此宣称匿名下载已通过。
