package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// 默认语言
const (
	DefaultLang = "en"
)

// 语言资源映射
var (
	messages     = make(map[string]map[string]string)
	messagesLock sync.RWMutex
)

// 初始化i18n，加载所有语言资源
func Init(localesDir string) error {
	// 确保目录存在
	if _, err := os.Stat(localesDir); os.IsNotExist(err) {
		return fmt.Errorf("locales directory not found: %s", localesDir)
	}

	// 读取所有语言文件
	files, err := os.ReadDir(localesDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// 提取语言代码 (文件名去掉扩展名)
		lang := file.Name()[:len(file.Name())-5]

		// 读取语言文件
		data, err := os.ReadFile(filepath.Join(localesDir, file.Name()))
		if err != nil {
			return err
		}

		// 解析JSON
		var langMessages map[string]string
		if err := json.Unmarshal(data, &langMessages); err != nil {
			return err
		}

		// 添加到消息映射
		messagesLock.Lock()
		messages[lang] = langMessages
		messagesLock.Unlock()
	}

	return nil
}

// 获取消息模板
func getMessageTemplate(code string, lang string) string {
	messagesLock.RLock()
	defer messagesLock.RUnlock()

	// 检查语言是否存在
	langMessages, ok := messages[lang]
	if !ok {
		// 回退到默认语言
		langMessages = messages[DefaultLang]
		// 如果默认语言也不存在，直接返回错误码
		if langMessages == nil {
			return code
		}
	}

	// 检查消息是否存在
	message, ok := langMessages[code]
	if !ok {
		// 如果没有找到消息，尝试使用默认语言
		if lang != DefaultLang && messages[DefaultLang] != nil {
			if defaultMsg, ok := messages[DefaultLang][code]; ok {
				return defaultMsg
			}
		}
		// 最后回退到错误码本身
		return code
	}

	return message
}

// 翻译错误码为消息
func Translate(code string, lang string, args ...interface{}) string {
	template := getMessageTemplate(code, lang)

	// 如果有参数，格式化消息
	if len(args) > 0 {
		return fmt.Sprintf(template, args...)
	}

	return template
}
