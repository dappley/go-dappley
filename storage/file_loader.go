package storage

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
)

func GetFileConnection(file string) ([]byte, error) {

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	} else if err != nil {
		log.Panic(err)
	}

	fileContent, err := ioutil.ReadFile(file)
	if err != nil {
		log.Panic(err)
	}
	return fileContent, nil

}

func SaveToFile(file string, buffer bytes.Buffer) {

	err := ioutil.WriteFile(file, buffer.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}
