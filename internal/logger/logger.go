package logger

import (
	"fmt"
	"time"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
)

func Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[%s]%s %s%s%s\n", Gray, time.Now().Format("15:04:05"), Reset, Blue, msg, Reset)
}

func Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[%s]%s %s%s%s\n", Gray, time.Now().Format("15:04:05"), Reset, Green, msg, Reset)
}

func Warn(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[%s]%s %s%s%s\n", Gray, time.Now().Format("15:04:05"), Reset, Yellow, msg, Reset)
}

func Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[%s]%s %s%s%s\n", Gray, time.Now().Format("15:04:05"), Reset, Red, msg, Reset)
}

func Debug(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[%s] [DEBUG] %s%s\n", Gray, time.Now().Format("15:04:05"), msg, Reset)
}

func Progress(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("\r%s[%s]%s %s%s%s", Gray, time.Now().Format("15:04:05"), Reset, Cyan, msg, Reset)
}
