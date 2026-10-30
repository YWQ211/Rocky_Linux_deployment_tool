package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"rockdev/internal/core"
)

func (a *App) menu(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		fmt.Fprintln(a.Out, "\nRockdev 0.1\n  1. 系统检查\n  2. 镜像检测与切换\n  3. 创建按需 RPM 仓库\n  4. 安装开发环境\n  5. 查看开发环境\n  6. 恢复镜像配置备份\n  0. 退出")
		choice, err := a.prompt(ctx, "请选择: ")
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var args []string
		switch choice {
		case "0":
			return nil
		case "1":
			args = []string{"doctor"}
		case "2":
			for _, m := range core.Mirrors {
				fmt.Fprintf(a.Out, "  %s: %s\n", m.ID, m.Name)
			}
			id, e := a.prompt(ctx, "镜像名 / auto 自动选择 / check 仅检测: ")
			if e != nil {
				return menuInputError(e)
			}
			if id == "check" {
				args = []string{"mirror", "check"}
			} else {
				args = []string{"mirror", "use", id}
			}
		case "3":
			dir, e := a.prompt(ctx, "新仓库绝对路径（父目录须已存在）: ")
			if e != nil {
				return menuInputError(e)
			}
			packages, e := a.prompt(ctx, "RPM 包名，用空格分隔: ")
			if e != nil {
				return menuInputError(e)
			}
			args = append([]string{"repo", "create", "--dir", dir}, strings.Fields(packages)...)
		case "4":
			names, e := a.prompt(ctx, "环境 python java go node build（空格分隔）: ")
			if e != nil {
				return menuInputError(e)
			}
			args = append([]string{"env", "install"}, strings.Fields(names)...)
		case "5":
			args = []string{"env", "list"}
		case "6":
			id, e := a.prompt(ctx, "备份 ID: ")
			if e != nil {
				return menuInputError(e)
			}
			args = []string{"mirror", "restore", id}
		default:
			fmt.Fprintln(a.Out, "无效选择。")
			continue
		}
		if err = a.Run(ctx, args); err != nil {
			fmt.Fprintln(a.Err, "错误:", err)
		}
	}
}

func menuInputError(err error) error {
	if err == io.EOF {
		return nil
	}
	return err
}
