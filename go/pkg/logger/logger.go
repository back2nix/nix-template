package logger

import "log"

// Stub logger - replace with zerolog/zap later
func Info(msg string) {
	log.Println("[INFO]", msg)
}

func Error(msg string) {
	log.Println("[ERROR]", msg)
}
