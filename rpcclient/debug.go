package rpcclient

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const sysflag = "rpcclient"

var ShowDebugInfo bool = true
var FULL_LOG = true
var START_LOG = true
var LOG_FILE_PATH = "rpcclient.log"
var rootDir = "./"

func output(info string, format string, v ...interface{}) {
	if FULL_LOG {
		pc, file, line, _ := runtime.Caller(2)
		short := file

		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		f := runtime.FuncForPC(pc)
		fn := f.Name()

		for i := len(fn) - 1; i > 0; i-- {
			if fn[i] == '.' {
				fn = fn[i+1:]
				break
			}
		}

		if format == "" {
			log.Printf("|%v|%v|%v|%v()|%v|%v", info, sysflag, short, fn, line, fmt.Sprintln(v...))
		} else {
			log.Printf("|%v|%v|%v|%v()|%v|%v", info, sysflag, short, fn, line, fmt.Sprintf(format, v...))
		}

	} else {
		if format == "" {
			log.Printf("[%s]|%v", info, fmt.Sprintln(v...))
		} else {
			log.Printf("[%s]|%v", info, fmt.Sprintf(format, v...))
		}
	}

}

func Debug(v ...interface{}) {
	if ShowDebugInfo {
		output("DEBUG", "", v...)
	}
}
func Debugf(format string, v ...interface{}) {
	if ShowDebugInfo {
		output("DEBUG", format, v...)
	}
}

func Error(v ...interface{}) {
	output("ERROR", "", v...)
}
func Errorf(format string, v ...interface{}) {
	output("ERROR", format, v...)
}

func Warning(v ...interface{}) {
	output("WARNING", "", v...)
}
func Warningf(format string, v ...interface{}) {
	output("WARNING", format, v...)
}

func Info(v ...interface{}) {
	output("INFO", "", v...)
}
func Infof(format string, v ...interface{}) {
	output("INFO", format, v...)
}

func Logger() *os.File {
	log.SetFlags(log.LstdFlags)
	f, err := os.OpenFile(rootDir+LOG_FILE_PATH, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	if START_LOG == true {
		log.SetOutput(f)
	}
	return f
}

func setDebugLevel(level string) {
	if level == "DEBUG" {
		ShowDebugInfo = true
	} else {
		ShowDebugInfo = false
	}
}

func signalHandle() {
	t1 := time.NewTimer(1 * time.Hour)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR1, syscall.SIGUSR2)
	for {
		select {
		case sig := <-ch:
			switch sig {
			case syscall.SIGUSR1:
				setDebugLevel("DEBUG")
				t1.Reset(1 * time.Hour)
			case syscall.SIGUSR2:
				setDebugLevel("ERROR")
				t1.Reset(1 * time.Hour)
				t1.Stop()
			default:
			}
		case <-t1.C:
			setDebugLevel("ERROR")
			t1.Stop()
		}
	}
}
