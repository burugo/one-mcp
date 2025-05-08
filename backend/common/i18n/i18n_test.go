package i18n

import (
	"one-mcp/backend/common/errors"
	"testing"
)

func TestTranslate(t *testing.T) {
	// 初始化资源文件
	if err := Init("../../../locales"); err != nil {
		t.Fatalf("Failed to init i18n: %v", err)
	}

	tests := []struct {
		code     string
		lang     string
		expected string
	}{
		{errors.ErrEmptyID, "en", "ID is empty"},
		{errors.ErrEmptyID, "zh", "ID 为空"},
		{errors.ErrUserNotFound, "en", "User not found"},
		{errors.ErrUserNotFound, "zh", "未找到用户"},
		{errors.ErrEmptyCredentials, "en", "Username or password is empty"},
		{errors.ErrEmptyCredentials, "zh", "用户名或密码为空"},
		// 测试不存在的语言是否回退到默认语言
		{errors.ErrEmptyID, "fr", "ID is empty"},
		// 测试不存在的错误码是否返回错误码本身
		{"UNKNOWN_ERROR", "en", "UNKNOWN_ERROR"},
	}

	for _, tt := range tests {
		result := Translate(tt.code, tt.lang)
		if result != tt.expected {
			t.Errorf("Translate(%s, %s) = %s, want %s", tt.code, tt.lang, result, tt.expected)
		}
	}
}

func TestNewError(t *testing.T) {
	// 初始化资源文件
	if err := Init("../../../locales"); err != nil {
		t.Fatalf("Failed to init i18n: %v", err)
	}

	// 测试英文错误
	err := New(errors.ErrEmptyID, "en")
	if err.Error() != "ID is empty" {
		t.Errorf("NewError(ErrEmptyID, en).Error() = %s, want 'ID is empty'", err.Error())
	}
	if err.Code != errors.ErrEmptyID {
		t.Errorf("NewError(ErrEmptyID, en).Code = %s, want %s", err.Code, errors.ErrEmptyID)
	}

	// 测试中文错误
	err = New(errors.ErrEmptyID, "zh")
	if err.Error() != "ID 为空" {
		t.Errorf("NewError(ErrEmptyID, zh).Error() = %s, want 'ID 为空'", err.Error())
	}

	// 测试错误码检查
	if !IsErrorCode(err, errors.ErrEmptyID) {
		t.Errorf("IsErrorCode() failed, error code not matching")
	}
	if IsErrorCode(err, "WRONG_CODE") {
		t.Errorf("IsErrorCode() failed, incorrectly matched wrong code")
	}
}

func TestErrorWithParams(t *testing.T) {
	// 手动添加带参数的消息模板用于测试
	messagesLock.Lock()
	if messages["en"] == nil {
		messages["en"] = make(map[string]string)
	}
	messages["en"]["TEST_WITH_PARAM"] = "Error with param: %s"
	messagesLock.Unlock()

	err := New("TEST_WITH_PARAM", "en", "test value")
	if err.Error() != "Error with param: test value" {
		t.Errorf("NewError with param failed, got '%s'", err.Error())
	}
} 