# 构建检查

日期：2026-10-30。构建主机 Windows，工具链 Go 1.27.1；项目语言最低版本 Go 1.22，最低编译器版本未单独验证。

- `go test ./... -count=1`：21 个测试函数通过。
- `go vet ./...`：通过。
- CGO_ENABLED=0，Linux amd64/arm64、Windows amd64：构建通过。
- Linux ELF 头：x86_64 machine=62、aarch64 machine=183。
- Windows 二进制：帮助、菜单退出、镜像清单通过；doctor 对 Windows 返回失败，未执行系统修改。
- 代码审查及复查：模块 RPM 配置、仓库级签名策略和交互取消问题均已修复并有回归测试。

Rocky 10.2 隔离容器中的实际安装、镜像和离线仓库测试见 [测试记录](test-results.md)。ARM64 仅交叉编译，未运行。
