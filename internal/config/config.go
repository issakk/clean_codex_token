package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadConfigJSON(configPath string) (map[string]any, error) {
	b, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var conf any
	if err := json.Unmarshal(b, &conf); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	obj, ok := conf.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("配置文件格式错误: 顶层必须是 JSON 对象")
	}
	return obj, nil
}
