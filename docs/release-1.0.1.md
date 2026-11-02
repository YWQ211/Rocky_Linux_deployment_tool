# Rockdev 1.0.1

中文 · [English](release-1.0.1.en.md) · [日本語](release-1.0.1.ja.md)

Rocky 8 的系统 Python 通过 alternatives 指向 `/usr/libexec/platform-python3.6`。1.0.0 会把它视为自定义环境，阻止包含 Python 的显式批量更新。1.0.1 为标准 Python 入口接受这一准确目标；其他 libexec 程序、自定义目录和隔离环境保护保持原有规则。

集成脚本改为读取实际 pip 安装命令解析出的 index-url，兼容 Rocky 8 自带的 pip 9。拒绝测试要求 CLI 返回 1，缺命令和超时不能算通过；任何用例失败时脚本返回非零。

命令格式不变。新产物位于 `dist/1.0.1/`，原 1.0.0 保留。完整功能见 [1.0 使用说明](release-1.0.md)，本次验证见 [复测记录](test-results-1.0.1.md)。
