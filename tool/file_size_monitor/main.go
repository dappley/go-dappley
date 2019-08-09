package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	logger "github.com/sirupsen/logrus"
)

const dbpath = "../../bin/node1.db"
const logFilePath = "log.csv"

func main() {

	s := int64(0)
	var arr []string
	prev := s
	for {
		time.Sleep(time.Second)
		arr, s, _ = DirSize(dbpath)
		if s == prev {
			continue
		}
		logger.WithFields(logger.Fields{
			"size": s,
		}).Info("Read data size")
		recordFileSize(s, arr)
		prev = s
	}
}

func recordFileSize(size int64, s []string) {
	f, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Panic("Open file failed while recording transactions")
	}
	w := csv.NewWriter(f)

	ret := []string{time.Now().String(), fmt.Sprint(size)}
	ret = append(ret, s...)
	w.Write(ret)
	w.Flush()
}

func DirSize(path string) ([]string, int64, error) {
	var size int64
	var s []string
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
			s = append(s, fmt.Sprintf("%s:%v", info.Name(), info.Size()))
		}
		return err
	})
	return s, size, err
}
