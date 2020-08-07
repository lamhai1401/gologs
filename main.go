package main

import (
	"context"
	"fmt"
	"time"
)

func test() {
	resultChann := make(chan string, 10)
	helloChan := make(chan string, 10)
	defer close(resultChann)

	parent := context.Background()
	timeOut := 3 * time.Second

	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(parent, timeOut)

	go func() {
		i := 0
		for {
			time.Sleep(1 * time.Second)
			helloChan <- "hello"
			if i == 10 {
				time.Sleep(10 * time.Second)
				i = 0
			} else {
				i++
			}
		}
	}()

	for {
		select {
		case str := <-helloChan:
			resultChann <- str
			break
		case <-ctx.Done():
			resultChann <- "context"
			break
		}
		result := <-resultChann
		fmt.Println(result)
		cancel()
		ctx, cancel = context.WithTimeout(parent, timeOut)
	}
}

func main() {
	n := NewNode("123")
	fmt.Println(n)
	select {}
}
