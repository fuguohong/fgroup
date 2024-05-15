package fgroup

import "context"

// Go 安全的执行goroutine，避免panic导致进程崩溃
func Go(ctx context.Context, fn func()) {
	go func() {
		defer func() {
			if e := recover(); e != nil {
				log(ctx, 2, e)
			}
		}()
		fn()
	}()
}
