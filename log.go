package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type (
	LogSTR struct {
		Lenx int
		Logs []string
		sync.RWMutex
	}
)

var (
	logs = NewLogSTR(100)
)

func NewLogSTR(s int) *LogSTR {
	return &LogSTR{
		Lenx: s,
		Logs: make([]string, 0),
	}
}

func (logx *LogSTR) AddLog(l string, args ...any) {
	logx.Lock()
	defer logx.Unlock()

	lx := fmt.Sprintf(timestamp()+" "+l, args...)
	fmt.Printf(lx + "\n")

	logx.Logs = append(logx.Logs, lx)
	if len(logx.Logs) > logs.Lenx {
		logx.Logs = logx.Logs[1:]
	}
}

func (logx *LogSTR) GetLogs() string {
	logx.RLock()
	defer logx.RUnlock()

	output := ""

	for i := range logx.Logs {
		output += logx.Logs[i] + "\n"
	}

	return output
}

func (logx *LogSTR) GetLogsHTML() string {
	d := logx.GetLogs()
	dd := strings.Split(d, "\n")
	ddd := strings.Join(dd, "<br />")
	return ddd
}

func timestamp() string {
	t := time.Now()
	y, m, d := t.Date()
	o := fmt.Sprintf("%02d %v %04d %02d:%02d:%02d", d, m, y, t.Hour(), t.Minute(), t.Second())
	return o
}
