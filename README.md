# go-recovery
A small library for panic handling in golang

## Simple example
### Code
```go
package main

import (
	"fmt"
	recovery "github.com/protogrammer/go-recovery"
	"log"
)

var rc = recovery.CreateConfig(func(msg recovery.PanicMessage) {
	log.Print("\n" + msg.String())
}, nil)

func fun(a, b int) int {
	defer rc.Recur("fun")
	defer recovery.CommentResult(func() string {
		return fmt.Sprintf("a = %d, b = %d", a, b)
	})
	return a / b
}

func main() {
	defer rc.Block("main")
	x := fun(5, 0)
	log.Print(x)
}
```

### Possible output
```
2022/09/21 07:07:24
Panic: runtime error: integer divide by zero
Type: runtime.errorString
Call stack:
    main
 -> fun
    Comment: a = 5, b = 0
```
