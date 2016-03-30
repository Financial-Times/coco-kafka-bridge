package main

import (
	"fmt"
	"log"
	"os"
)

type CombinedLogger interface {
	info(msg string)
	warn(msg string)
	error(msg string)
	access(msg string)
	Write(p []byte) (int, error)
}

type simpleCombinedLogger struct {
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
	Access  *log.Logger
}

func (l simpleCombinedLogger) info(msg string) {
	l.Info.Println(msg)
}

func (l simpleCombinedLogger) warn(msg string) {
	l.Warning.Println(msg)
}

func (l simpleCombinedLogger) error(msg string) {
	l.Error.Println(msg)
}

func (l simpleCombinedLogger) access(msg string) {
	l.Access.Println(msg)
}

func (l simpleCombinedLogger) Write(p []byte) (int, error) {
	msg := string(p)
	l.Access.Print(msg)
	return len(msg), nil
}

var logger CombinedLogger

func initLoggers() {
	logger = simpleCombinedLogger{
		log.New(os.Stdout, "INFO    - ", log.Ldate|log.Ltime),
		log.New(os.Stdout, "WARNING - ", log.Ldate|log.Ltime),
		log.New(os.Stderr, "ERROR   - ", log.Ldate|log.Ltime),
		log.New(os.Stdout, "ACCESS  - ", log.Ldate|log.Ltime),
	}
}

type TxCombinedLogger struct {
	wrapped CombinedLogger
	txID    string
}

func (l TxCombinedLogger) info(msg string) {
	l.wrapped.info(fmt.Sprintf("transaction_id=%+v - %+v", l.txID, msg))
}

func (l TxCombinedLogger) warn(msg string) {
	l.wrapped.warn(fmt.Sprintf("transaction_id=%+v - %+v", l.txID, msg))
}

func (l TxCombinedLogger) error(msg string) {
	l.wrapped.error(fmt.Sprintf("transaction_id=%+v - %+v", l.txID, msg))
}

func (l TxCombinedLogger) access(msg string) {
	l.wrapped.access(fmt.Sprintf("transaction_id=%+v - %+v", l.txID, msg))
}

func (l TxCombinedLogger) Write(p []byte) (int, error) {
	msg := string(p)
	l.access(msg)
	return len(msg), nil
}
