[English](README.md) | 中文

# fgroup
安全的使用 go routine，panic恢复和堆栈记录，管理一组goroutine的执行、并发数量

* [背景](#背景)
* [特性](#特性)
* [安装](#安装)
* [Go](#Go)
* [Group](#Group)
* [并发](#并发)
* [Context](#Context)
* [日志](#记录日志)
* [routine注意/技巧](#注意)
* [测试](#测试)

[TOC]

## 背景

- 野生goroutine一但发生panic将导致进程崩溃，主进程的recover并不会生效。 需要一种安全的方式去使用goroutine，而非直接go func()。
- 野生goroutine发生panic时进程崩溃，信息丢失，造成难以复现和追踪。所以在goroutine中发生不可预知的panic时需要记录panic信息、上下文以及发生panic的调用栈。
- 需要并发的去执行一组goroutine的时候，错误难以管理，等待组完成代码难写。所以需要一个方便的方式去运行一组goroutine
- 执行一组goroutine的时候，并发难以管理。直接全部放到go中执行不限制并发，导致大量系统资源占用，执行效率降低
- 执行一组goroutine的时候，如果发生了错误，应该视为失败，需要终止执行，提前释放资源； 而不是傻傻的去把声明了的func全部执行完，再来wait有没有错误

这个包的目标就是解决以上问题



## 特性

- 管理野生goroutine
- 等待一组gorotine的结果
- goroutine并发限制
- context生命管理
- 记录panic堆栈
- 0依赖




## 安装

```
go get github.com/fuguohong/fgroup
```



## 用法

### Go

异步的执行某个/某些操作， 不关注他们什么时候完成， 也不关注他们的error或在内部处理error

**不推荐**的用法：

```go
// 不推荐在代码中直接用go开启协程，野生goroutine如果发生panic整个进程会崩溃。入口出的recover是不会生效的
go func(){
  // do job
}()

go func(){
  // do job
}()
```
**推荐**的用法

```go
// 以下两个函数会在两个新的go routine中异步并发执行，如果发生panic，会自动恢复并记录panic堆栈
fgroup.Go(ctx,func(){
  // do job
})
fgroup.Go(ctx,func(){
  // do job
})
```



### Group

异步的执行一组操作，需要等待这一组操作全部完成再执行后序代码；使用group简化操作



注意：**group捕获到error后将不再执行还没有执行的函数**！ 捕获到error包括以下情况： 

-  传入Go的函数return error
-  传入Go的函数发生了panic
-  context失活，如超时、被取消等，这取决于传入的context

如果要达到部分函数发生了error也不影响其他函数执行的效果，则将传入group的函数都return nil。panic和context失活硬性要求终止执行

```go
g := fgroup.NewGroup(ctx)

g.Go(func()error{
  // do job
  return nil
})
g.Go(func()error{
  if err != nil{
    return err
  }
  // do job
})
...

// 等待所有任务执行完
// 只要group中任意一个func返回了error，则err不为nil
err := g.Wait()
if err .....
// 后续逻辑...
```



#### 并发

```go
// group将保证最多只有5个任务在同时运行
g,_ := NewGroupWithParallel(ctx, 5)
for _,job := range jobs{
  j := job
  g.Go(func()error{
  	// process j
  	return nil
  })
}
err := g.Wait()
```



#### context

context不再存活后，group将中断后续任务的执行

```go
ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)
g := NewGroup(ctx)
i := 0
g.Go(func() error {
  i += 1
  return nil
})
time.Sleep(time.Millisecond * 110)
// 此时context已经死亡，这个任务不会执行
g.Go(func() error {
  i += 1
  return nil
})
err := g.Wait() // context deadline exceeded
fmt.PrintLn(i) // 1
```



`NewGroupWithCancel`以及`NewGroupWithParallel`额外返回一个子context， 这个context会在group捕获到error后被取消。将这个子context传递给goroutine使用，可以在发生错误后提前终止正在执行的goroutine

**建议都使用这种方式来使用Group**

```go
g, childctx := NewGroupWithCancel(ctx)
ch := make(chan bool)
g.Go(func() error {
  defer close(ch)
  return errors.New("normal error")
})
g.Go(func() error{
  // 等待error发生
  <- ch
  // 注意这里传递的是子context
  // 这个sql不会执行，因为上面的func返回error后childctx已经被取消了
  err := db.WithContext(childctx).Table("user").First(&result).Error 
  fmt.Println(err) // context canceled
  if err != nil{
    return err
  }
  // more ....
})
err := g.Wait() // normal error
morejob(ctx)
```



### 记录日志

```go
// init.go
// 在项目初始化代码中注入记日志函数，如果发生panic,将通过注入的日志函数记录panic堆栈信息.如果未注入日志函数，则不会记录日志

// 如果goroutine发生了未捕获的panic，会用log.Error记录日志
// 传入的第一个参数为发生panic的上下文，第二个参数为panic错误，第三个参数为发生panic的调用栈
fgroup.Log = func(ctx context.Context, ipanic interface{}, stack string) {
  logger.WithContext(ctx).Error(ipanic, stack)
}

// 记录日志时会附带堆栈信息，堆栈追踪深度默认为8
// 修改堆栈深度
fgroup.TraceDepth = 16
// 关闭堆栈追踪
fgroup.TraceDepth = 0
```



## 注意

一些使用协程的注意事项。为了方便理解，下面的代码使用的是原生go开启协程。 不管是用原生还是fgroup.Go或者其他方法开启协程，这些问题都是存在的

1. **不要滥用协程，协程的创建管理调度有不小的开销；使用不当还可能造成死锁、协程泄漏等情况**

2. **使用协程时注意局部变量的变换**

```go
//  ======= 错误的用法 ========
// 变量job的值一直在变化，子协程执行时job的值是不确定的，很大可能是十个10，而不是期待的0-9，这取决于父子协程的执行进度
for job := 0; job < 10; job++ {
    go func() {
        fmt.Println(job)
    }()
}

// ========= 正确的用法 ==========
// 用一个新的变量保存job的值
for job := 0; job < 10; job++ {
    j := job // 重要！！！
    go func() {
        fmt.Println(j)
    }()
}
```

3. **慎用channel， 注意协程泄漏**

```go
// 这是一个goroutine泄漏的简单例子

func DoSomeThing(){
  ch := gen(jobIds)
  // process 异常中断，gen内部开启的协程将泄漏
  go process(ch)
}

func gen(jobIds)chan job{
  ch := make(chan job)
  go func(){
    // 重要，可保process不泄漏；否则函数异常中断，process将泄漏
    defer close(ch)
    for _, id := range jobIds{
      job, err := GetJob(id)
      // if err ...
      ch <- job
  	}
  }()
  return ch
}

func process(ch chan job){
  for {
    job, isopen <- := ch
    if !isopen{
      return
    }
    err := job.Do()
    // 异常中断
    // if err != nil{
    //  return
    // }
    // 或者意料之外的panic
  }
}
```


## 测试

`go test -cover .`

测试用例已完成100%代码覆盖

`./batcjtest.sh`
多次运行测试，避免偶现的高并发问题


