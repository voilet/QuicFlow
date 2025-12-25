package callback

import "errors"

var (
	// ErrPromiseTimeout Promise 超时
	ErrPromiseTimeout = errors.New("promise timeout")

	// ErrPromiseNotFound Promise 未找到
	ErrPromiseNotFound = errors.New("promise not found")

	// ErrPromiseCapacityFull Promise 容量已满
	ErrPromiseCapacityFull = errors.New("promise capacity full (50,000 limit)")
)
