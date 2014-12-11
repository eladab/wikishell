package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func StartListening() {
	signal.Notify(winResizeChan, syscall.SIGWINCH)

	go func(ch chan []byte, eCh chan error) {
		for {
			b := make([]byte, 1)
			_, err := os.Stdin.Read(b)
			if err != nil {
				eCh <- err
				return
			}
			ch <- b
		}
	}(stdinChan, errorChan)
}

func ReadChar() (byte, bool) {
	select {
	case <-winResizeChan:
		return 0, true
		break
	case b := <-stdinChan:
		return b[0], false
		break
	case err := <-errorChan:
		fmt.Println(err)
		return 0, false
		break
	}
	return 0, false
}

func ListenToOSSignals() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			fmt.Println(sig)
			RestoreShell()
			os.Exit(0)
		}
	}()
}

func ListenToWinChange() {
	c := make(chan os.Signal, 20)
	signal.Notify(c, syscall.SIGWINCH)
	go func() {
		for sig := range c {
			fmt.Println(sig)
		}
	}()
}
