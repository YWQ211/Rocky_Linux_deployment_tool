package cli

import (
	"fmt"
	"os"
	"strings"

	"rockdev/internal/i18n"
)

// DefaultLanguage may be set with -ldflags for a language-specific build.
var DefaultLanguage = "zh"

func (a *App) Text(key string) string { return i18n.Text(a.Language, key) }
func (a *App) tr(key string) string   { return a.Text(key) }
func (a *App) selectLanguage(args []string) ([]string, error) {
	lang := a.Language
	if lang == "" {
		lang = os.Getenv("ROCKDEV_LANG")
	}
	if lang == "" {
		lang = DefaultLanguage
	}
	var rest []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--lang" {
			i++
			if i == len(args) {
				return nil, fmt.Errorf("--lang requires zh, en or ja")
			}
			lang = args[i]
		} else if strings.HasPrefix(args[i], "--lang=") {
			lang = strings.TrimPrefix(args[i], "--lang=")
		} else {
			rest = append(rest, args[i])
		}
	}
	if lang != "zh" && lang != "en" && lang != "ja" {
		return nil, fmt.Errorf("Unsupported language %q; choose zh, en or ja", lang)
	}
	a.Language = lang
	return rest, nil
}
func (a *App) helpText() string {
	if a.Language == "zh" {
		return help + "\n--lang zh|en|ja  选择界面语言（也可设置 ROCKDEV_LANG）\n"
	}
	intro, end := "Rocky Linux development environment tool", "Without arguments, opens the interactive menu.\n--dry-run  Show actions without changing files or installing packages.\n--yes      Skip confirmation.\n--lang     zh, en or ja (or set ROCKDEV_LANG).\nRPM repositories only; npm/pip/Maven/Go module mirrors are not included.\n"
	if a.Language == "ja" {
		intro = "Rocky Linux 開発環境ツール"
		end = "引数なしで対話メニューを開きます。\n--dry-run  ファイル変更やインストールを行わず、手順を表示。\n--yes      実行確認を省略。\n--lang     zh、en、ja（ROCKDEV_LANG でも指定可能）。\nRPM リポジトリのみ対応。npm/pip/Maven/Go モジュールのミラーは含みません。\n"
	}
	return "rockdev 0.1.0 — " + intro + `
  rockdev doctor
  rockdev mirror list
  rockdev mirror check [mirror]
  rockdev mirror use <official|aliyun|tuna|ustc|auto>
  rockdev mirror restore <backup-id>
  rockdev repo create --dir <absolute-path> <package...>
  rockdev env list
  rockdev env install <python|java|go|node|build>...
  rockdev version

` + end
}
