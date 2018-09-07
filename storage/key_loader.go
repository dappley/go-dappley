package storage

import (
	"os"
	"log"
	"io/ioutil"
)

func GetKeyFileConnection(file string) ([]byte, error) {

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


