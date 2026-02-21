package utils

import (
	"fmt"
	"runtime/debug"
)

func genErr(e error, stack []byte) error {
	return fmt.Errorf("error: %v\nstack:\n%s", e, stack)
}

func Panic[T any](res T, err error) T {
	if err != nil {
		panic(genErr(err, debug.Stack()))
	}
	return res
}

func PanicVoid(err error) {
	if err != nil {
		panic(genErr(err, debug.Stack()))
	}
}

func PanicMap[T any](panicFn func(error)) func(res T, err error) T {
	return func(res T, err error) T {
		if err != nil {
			panicFn(genErr(err, debug.Stack()))
			return res
		}
		return res
	}
}

func PanicMapVoid(panicFn func(error)) func(err error) {
	return func(err error) {
		if err != nil {
			panicFn(genErr(err, debug.Stack()))
		}
	}
}
