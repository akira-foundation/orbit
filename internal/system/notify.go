package system

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func Notify(title, body string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	script := fmt.Sprintf("display notification %s with title %s", appleQuote(body), appleQuote(title))
	return exec.Command("osascript", "-e", script).Run()
}

func appleQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
