package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func PromptInt(in io.Reader, out io.Writer, label string, defaultValue int, minValue int) int {
	reader := bufio.NewReader(in)
	_, _ = fmt.Fprintf(out, "%s（默认 %d）: ", label, defaultValue)
	raw, _ := reader.ReadString('\n')
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		_, _ = fmt.Fprintln(out, "输入无效，使用默认值")
		return defaultValue
	}
	if value < minValue {
		_, _ = fmt.Fprintf(out, "输入过小，使用最小值 %d\n", minValue)
		return minValue
	}
	return value
}

func ChooseModeInteractive(in io.Reader, out io.Writer) string {
	reader := bufio.NewReader(in)
	_, _ = fmt.Fprintln(out, "\n请选择操作:")
	_, _ = fmt.Fprintln(out, "1) 仅检查 401 并导出")
	_, _ = fmt.Fprintln(out, "2) 检查 401 并立即删除")
	_, _ = fmt.Fprintln(out, "3) 直接删除 output 文件中的账号")
	_, _ = fmt.Fprintln(out, "0) 退出")
	for {
		_, _ = fmt.Fprint(out, "请输入选项编号: ")
		choice, _ := reader.ReadString('\n')
		s := strings.TrimSpace(choice)
		switch s {
		case "1":
			return "check"
		case "2":
			return "check_delete"
		case "3":
			return "delete_from_output"
		case "0":
			return "exit"
		default:
			_, _ = fmt.Fprintln(out, "无效选项，请重新输入。")
		}
	}
}

func PromptToken(in io.Reader, out io.Writer) string {
	reader := bufio.NewReader(in)
	_, _ = fmt.Fprint(out, "请输入管理 token（Bearer 后面的值）: ")
	v, _ := reader.ReadString('\n')
	return strings.TrimSpace(v)
}

func ConfirmDelete(in io.Reader, out io.Writer, count int) bool {
	reader := bufio.NewReader(in)
	_, _ = fmt.Fprintf(out, "即将删除 %d 个账号，输入 DELETE 确认: ", count)
	v, _ := reader.ReadString('\n')
	return strings.TrimSpace(v) == "DELETE"
}
