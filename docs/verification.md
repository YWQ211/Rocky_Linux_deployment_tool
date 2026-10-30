# 0.1.0 本次验证

日期：2026-10-30。构建主机 Windows，工具链 Go 1.27.1；项目语言最低版本 Go 1.22，最低编译器版本未单独验证。

- `go test ./... -count=1`：16 个测试函数通过。
- `go vet ./...`：通过。
- CGO_ENABLED=0，Linux amd64/arm64、Windows amd64：构建通过。
- Linux ELF 头：x86_64 machine=62、aarch64 machine=183。
- Windows 二进制：帮助、菜单退出、镜像清单通过；doctor 对 Windows 返回失败，未执行系统修改。
- 代码审查及复查：模块 RPM 配置、仓库级签名策略和交互取消问题均已修复并有回归测试。

未验证：Rocky VM 实际部署、各候选镜像在 VM 上的连通性、离线依赖闭包、真实 RPM 签名输出与包版本、Linux 锁及 SIGTERM 的真实运行行为。自动化测试中的外部安装命令由假运行器模拟；元数据检测使用本地 HTTP 服务器。后续按 vm-acceptance.md 在目标 VM 执行。
