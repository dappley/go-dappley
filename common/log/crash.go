package log

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

// CrashHandler write crash log info into crash log file
func CrashHandler() {
	errs := recover()
	if errs == nil {
		return
	}
	// get program name
	exeName := os.Args[0]
	now := time.Now()
	pid := os.Getpid()

	time_str := now.Format("20060102150405")
	fname := fmt.Sprintf("%s-%d-%s-crash.log", exeName, pid, time_str)
	fmt.Println("dump to file ", fname)

	f, err := os.Create("./log/"+fname)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(fmt.Sprintf("%v\r\n", errs))
	f.WriteString("========\r\n")

	f.WriteString(string(debug.Stack()))
}
