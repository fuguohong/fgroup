English | [中文](README-CN.md)

# fgroup
A safe way to use goroutines with panic recovery and stack recording, managing the execution and concurrency of a group of goroutines.


## index
* [Why](#why)
* [Features](#Features)
* [Installation](#Installation)
* [Go](#Go)
* [Group](#Group)
* [Limit the concurrency of Goroutines](#Concurrency)
* [Context](#Context)
* [Logging](#Logging)
* [Testing](#Testing)


## Why

- If a goroutine without recovery mechanism encounters a panic, it will cause the process to crash, and the recover in the main goroutine will not take effect. A safe way is needed to use goroutines instead of directly using `go func()`.
- When a goroutine encounters an unpredictable panic, the process crashes and the information is lost, making it difficult to reproduce and trace. Therefore, when an unpredictable panic occurs in a goroutine, it is necessary to record the panic information, context, and the call stack where the panic occurred.
- When executing a group of goroutines concurrently, error management becomes difficult, and writing code to wait for the group to complete becomes challenging. Therefore, a convenient way is needed to run a group of goroutines.
- When executing a group of goroutines, managing concurrency becomes difficult. Executing all goroutines in a `go` statement without limiting concurrency leads to excessive consumption of system resources and reduced execution efficiency.
- When executing a group of goroutines, if an error occurs, it should be treated as a failure and the execution should be terminated to release resources early, instead of blindly executing all declared functions and then waiting to see if there are any errors.

The goal of this package is to solve the above problems.



## Features

- Management of goroutines without recovery mechanism
- Waiting for a group of goroutines to complete
- Limiting the concurrency of goroutines
- Context lifecycle management
- Recording panic stack traces
- Zero dependencies



## Installation

```
go get github.com/fuguohong/fgroup
```



## Usage

### Go

Asynchronously execute one or more operations without caring about when they will complete or handling errors internally.

**Not recommended** usage:

```go
// Not recommended to directly use `go` to start a goroutine in the code. If a goroutine without recovery mechanism encounters a panic, 
// the entire process will crash. The recover in the main goroutine will not take effect.
go func() {
  // do job
}()

go func() {
  // do job
}()
```

**Recommended** usage:

```go
// The following two functions will be executed asynchronously in two new goroutines. If a panic occurs, 
// it will be automatically recovered and the panic stack will be recorded.
fgroup.Go(ctx, func() {
  // do job
})
fgroup.Go(ctx, func() {
  // do job
})
```



### Group

Asynchronously execute a group of operations and wait for all of them to complete before executing subsequent code. Use the group to simplify the operation.

Note: **Once the group captures an error, it will no longer execute functions that have not been executed yet!** Capturing an error includes the following cases:

- The function passed to `Go` returns an error.
- The function passed to `Go` encounters a panic.
- The context becomes inactive, such as timeout, cancellation, etc., depending on the context passed in.

If you want to achieve the effect where some functions can return errors without affecting the execution of other functions, all functions passed to the group should return `nil`. Panics and inactive contexts are hard requirements for terminating execution.

```go
g := fgroup.NewGroup(ctx)

g.Go(func() error {
  // do job
  return nil
})
g.Go(func() error {
  if err != nil {
    return err
  }
  // do job
})
// ...

// Wait for all tasks to complete
// If any function in the group returns an error, `err` will not be `nil`
err := g.Wait()
if err != nil {
  // handle error
}
// more logic...
```



#### Concurrency
Limit the concurrency of Goroutines
```go
// The group will ensure that at most 5 tasks are running simultaneously
g, _ := NewGroupWithParallel(ctx, 5)
for _, job := range jobs {
  j := job
  g.Go(func() error {
    // process j
    return nil
  })
}
err := g.Wait()
```



#### Context

Once the context is no longer alive, the group will interrupt the execution of subsequent tasks.

```go
ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)
g := NewGroup(ctx)
i := 0
g.Go(func() error {
  i += 1
  return nil
})
time.Sleep(time.Millisecond * 110)
// At this point, the context has died, and this task will not be executed
g.Go(func() error {
  i += 1
  return nil
})
err := g.Wait() // context deadline exceeded
fmt.Println(i) // 1
```

`NewGroupWithCancel` and `NewGroupWithParallel` additionally return a child context. This context will be canceled by the group after capturing an error. Passing this child context to goroutines allows for early termination of executing goroutines after an error occurs.

**It is recommended to use this approach when using the Group.**

```go
g, childctx := NewGroupWithCancel(ctx)
ch := make(chan bool)
g.Go(func() error {
  defer close(ch)
  return errors.New("normal error")
})
g.Go(func() error {
  // Wait for the error to occur
  <-ch
  // Note that the child context is passed here
  // This SQL statement will not be executed because the childctx has been canceled after the previous function returned an error
  err := db.WithContext(childctx).Table("user").First(&result).Error
  fmt.Println(err) // context canceled
  if err != nil {
    return err
  }
  // more ....
})
err := g.Wait() // normal error
morejob(ctx)
```



### Logging

```go
// init.go
// Inject the logging function in the project initialization code. 
// If a panic occurs, the panic stack trace will be recorded using the injected logging function. 
// If no logging function is injected, the stack trace will not be recorded.

// If a goroutine encounters an uncaught panic, it will be logged using log.Error
// The first parameter passed is the context in which the panic occurred, 
// the second parameter is the panic error, and the third parameter is the call stack where the panic occurred
fgroup.Log = func(ctx context.Context, ipanic interface{}, stack string) {
  logger.WithContext(ctx).Error(ipanic, stack)
}

// The stack trace will be recorded with the log, and the default stack trace depth is 8
// Modify the stack trace depth
fgroup.TraceDepth = 16
// Disable stack trace
fgroup.TraceDepth = 0
```



## Testing
```
ok      github.com/fuguohong/fgroup     (cached)        coverage: 100.0% of statements
```