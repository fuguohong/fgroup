package fgroup

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

// Log 发生panic时记录日志
// 第一个参数为发生panic的上下文，第二个参数为panic错误，第三个参数为发生panic的调用栈
var Log func(context.Context, interface{}, string)

// TraceDepth 最大堆栈深度
var TraceDepth = 8

func log(ctx context.Context, skip int, ipanic interface{}) {
	if Log == nil {
		return
	}
	Log(ctx, ipanic, " xroutine panic recovered, stack: \n"+getStack(skip+1))
}

func getStack(skip int) string {
	if TraceDepth <= 0 {
		return ""
	}
	callers := make([]uintptr, TraceDepth)
	n := runtime.Callers(skip+2, callers)
	if n == 0 {
		return ""
	}

	frames := runtime.CallersFrames(callers[0:n])
	stacks := make([]string, 0, n)
	for {
		f, more := frames.Next()
		if !more {
			break
		}
		stacks = append(stacks, fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Function))
	}
	return strings.Join(stacks, "\n")
}
