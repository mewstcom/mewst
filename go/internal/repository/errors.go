package repository

import "errors"

// ErrNotFound はリソースが見つからない場合のエラー
var ErrNotFound = errors.New("resource not found")
