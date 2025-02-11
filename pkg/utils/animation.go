package utils

import (
	"fmt"
	"time"
)

func ShowLoadingAnimation(done chan bool, text string) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	for {
		select {
		case <-done:
			return
		default:
			fmt.Printf("\r%s %s", text, frames[i%len(frames)])
			i++
			time.Sleep(100 * time.Millisecond) // Adjust speed if needed
		}
	}
}
