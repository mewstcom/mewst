package usecase

import "errors"

// ErrNotFound はリソースが見つからない場合に返すエラー
var ErrNotFound = errors.New("not found")
