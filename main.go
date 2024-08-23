package main

import (
	"bufio"
	"fmt"
	"os"
	"phoenixbuilder/fastbuilder/core"
	"time"

	"github.com/pterm/pterm"
)

func main() {
	userEnter := make(chan struct{}, 1)
	continueRun := make(chan struct{}, 1)

	go func() {
		pterm.Info.Printf("回车以启动程序，或等待 1 分钟后由程序自动启动。")
		bufio.NewReader(os.Stdin).ReadString('\n')
		userEnter <- struct{}{}
		close(userEnter)
	}()

	go func() {
		timer := time.NewTimer(time.Minute * 1)
		defer func() {
			continueRun <- struct{}{}
			close(continueRun)
			timer.Stop()
		}()
		select {
		case <-timer.C:
			fmt.Printf("\n")
		case <-userEnter:
		}
	}()

	<-continueRun
	core.Bootstrap()
}
