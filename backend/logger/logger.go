package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

var (
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
)

func InitLogger(logFilePath string) {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open logfile: %v", err)
	}

	multiInfo := io.MultiWriter(os.Stdout, logFile)
	multiWarn := io.MultiWriter(os.Stdout, logFile)
	multiErr := io.MultiWriter(os.Stderr, logFile)
	multiDebug := io.MultiWriter(os.Stdout, logFile)

	infoLogger = log.New(multiInfo, ColorGreen+"[INFO] "+ColorReset, log.LstdFlags)
	warnLogger = log.New(multiWarn, ColorYellow+"[WARN] "+ColorReset, log.LstdFlags)
	errorLogger = log.New(multiErr, ColorRed+"[ERROR] "+ColorReset, log.LstdFlags)
	debugLogger = log.New(multiDebug, ColorCyan+"[DEBUG] "+ColorReset, log.LstdFlags)
}

func Info(msg string, args ...any) {
	infoLogger.Printf(msg, args...)
}

func Warn(msg string, args ...any) {
	warnLogger.Printf(msg, args...)
}

func Error(msg string, args ...any) {
	errorLogger.Printf(msg, args...)
}

func Debug(msg string, args ...any) {
	debugLogger.Printf(msg, args...)
}

func InfoSection(title string) {
	fmt.Println(ColorBlue + "\n🧠 " + title + ColorReset)
}

