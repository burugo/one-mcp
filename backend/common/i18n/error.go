package i18n

import (
	"errors"
	mcperrors "one-mcp/backend/common/errors"
)

// I18nError 表示一个国际化错误
type I18nError struct {
	Code string
	Msg  string
	Err  error
}

// 实现 error 接口
func (e *I18nError) Error() string {
	return e.Msg
}

// 获取错误码
func (e *I18nError) ErrorCode() string {
	return e.Code
}

// 获取原始错误
func (e *I18nError) Unwrap() error {
	return e.Err
}

// New 创建一个新的国际化错误
func New(code string, lang string, args ...interface{}) *I18nError {
	msg := Translate(code, lang, args...)
	return &I18nError{
		Code: code,
		Msg:  msg,
		Err:  errors.New(msg),
	}
}

// Wrap 包装一个已有错误
func Wrap(err error, code string, lang string, args ...interface{}) *I18nError {
	msg := Translate(code, lang, args...)
	return &I18nError{
		Code: code,
		Msg:  msg,
		Err:  err,
	}
}

// 便捷方法：创建通用错误
func InternalServerError(lang string) *I18nError {
	return New(mcperrors.ErrInternalServer, lang)
}

func InvalidParamError(lang string, param string) *I18nError {
	return New(mcperrors.ErrInvalidParam, lang, param)
}

// 是否为特定错误码
func IsErrorCode(err error, code string) bool {
	var i18nErr *I18nError
	if errors.As(err, &i18nErr) {
		return i18nErr.Code == code
	}
	return false
}
