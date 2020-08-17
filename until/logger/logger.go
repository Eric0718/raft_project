package logger

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	PanicLevel int = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

const (
	color_red     = uint8(iota + 91)
	color_green   //	绿
	color_yellow  //	黄
	color_blue    // 	蓝
	color_magenta //	洋红
)

const (
	fatalPrefix = "[FATAL] "
	errorPrefix = "[ERROR] "
	warnPrefix  = "[WARN] "
	infoPrefix  = "[INFO] "
	debugPrefix = "[DEBUG] "
)

const (
	ByDay int = iota
	ByWeek
	ByMonth
	BySize
)

type LogFile struct {
	level    int    // 日志等级
	saveMode int    // 保存模式
	saveDays int    // 日志保存天数
	logTime  int64  //
	fileName string // 日志文件名
	filesize int64  // 文件大小, 需要设置 saveMode 为 BySize 生效
	fileFd   *os.File
}

var logFile LogFile

func init() {
	logFile.saveMode = ByDay // 默认按天保存
	logFile.saveDays = 2     // 默认保存三天的
	logFile.level = ErrorLevel
	logFile.filesize = 1024 * 1024 // 默认1M， 需要设置 saveMode 为 BySize
}

func Config(logFolder string, level int) {
	logFile.fileName = logFolder
	logFile.level = level

	log.SetOutput(logFile)
	//log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	log.SetFlags(log.Llongfile | log.Ldate | log.Ltime)
}
func GetLogFile() *LogFile {
	return &logFile
}

func SetLevel(level int) {
	logFile.level = level
}

func SetSaveMode(saveMode int) {
	logFile.saveMode = saveMode
}

func SetSaveDays(saveDays int) {
	logFile.saveDays = saveDays
}

func SetSaveSize(saveSize int64) {
	logFile.filesize = saveSize
}

func Debugf(format string, args ...interface{}) {
	if logFile.level >= DebugLevel {
		log.SetPrefix(blue(debugPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func Infof(format string, args ...interface{}) {
	if logFile.level >= InfoLevel {
		log.SetPrefix(green(infoPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func Warnf(format string, args ...interface{}) {
	if logFile.level >= WarnLevel {
		log.SetPrefix(magenta(warnPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func Errorf(format string, args ...interface{}) {
	if logFile.level >= ErrorLevel {
		log.SetPrefix(red(errorPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func Fatalf(format string, args ...interface{}) {
	if logFile.level >= FatalLevel {
		log.SetPrefix(red(fatalPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func GetRedPrefix(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_red, s)
}

func red(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_red, s)
}

func green(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_green, s)
}

func yellow(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_yellow, s)
}

func blue(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_blue, s)
}

func magenta(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_magenta, s)
}

func (lf LogFile) Write(buf []byte) (n int, err error) {
	if lf.fileName == "" {
		fmt.Printf("consol: %s", buf)
		return len(buf), nil
	}

	switch logFile.saveMode {
	case BySize:
		fileInfo, err := os.Stat(logFile.fileName)
		if err != nil {
			logFile.createLogFile()
			logFile.logTime = time.Now().Unix()
		} else {
			filesize := fileInfo.Size()
			if logFile.fileFd == nil ||
				filesize > logFile.filesize {
				logFile.createLogFile()
				logFile.logTime = time.Now().Unix()
			}
		}
	default: // 默认按天  ByDay
		if logFile.logTime+3600 < time.Now().Unix() {
			logFile.createLogFile()
			logFile.logTime = time.Now().Unix()
		}
	}

	if logFile.fileFd == nil {
		fmt.Printf("log fileFd is nil !\n")
		return len(buf), nil
	}

	return logFile.fileFd.Write(buf)
}

func (lf *LogFile) createLogFile() {
	logdir := "./log/log"
	if index := strings.LastIndex(lf.fileName, "/"); index != -1 {
		logdir = lf.fileName[0:index] + "/"
		os.MkdirAll(lf.fileName[0:index], os.ModePerm)
	}

	now := time.Now()
	filename := fmt.Sprintf("%s_%04d%02d%02d_%02d%02d",
		lf.fileName, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())
	if err := os.Rename(lf.fileName, filename); err == nil {
		go func() {
			tarCmd := exec.Command("tar", "-zcf", filename+".tar.gz", filename, "--remove-files")
			tarCmd.Run()

			rmCmd := exec.Command("/bin/sh", "-c",
				"find "+logdir+` -type f -mtime +`+string(logFile.saveDays)+` -exec rm {} \;`)
			rmCmd.Run()
		}()
	}

	for index := 0; index < 10; index++ {
		if fd, err := os.OpenFile(lf.fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModeExclusive); nil == err {
			lf.fileFd.Sync()
			lf.fileFd.Close()
			lf.fileFd = fd
			break
		} else {
			fmt.Println("Open logfile error! err: ", err.Error())
		}
		lf.fileFd = nil
	}
}
