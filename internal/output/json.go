package output

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadNamesFromOutput(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 output 文件失败: %w", err)
	}
	var rows any
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, fmt.Errorf("读取 output 文件失败: %w", err)
	}

	arr, _ := rows.([]any)
	names := make([]string, 0, len(arr))
	for _, row := range arr {
		m, _ := row.(map[string]any)
		name, _ := m["name"].(string)
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}
