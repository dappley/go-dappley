package storage

import (
	"fmt"
	"os"

	logger "github.com/sirupsen/logrus"
)

type RamFileLoader struct {
	dirpath  string
	filename string
	File     *FileLoader
}

func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func NewRamFileLoader(dirpath string, filename string) *RamFileLoader {
	//create fake files in fakeFileLoaders directory if not exist
	_, err := os.Stat(dirpath)
	if os.IsNotExist(err) {
		err := os.Mkdir(dirpath, os.ModePerm)
		if err != nil {
			logger.Errorf("Create test account file folder error: %v", err.Error())
		}
	} else if err != nil {
		fmt.Println("Fail to create path!")
		return nil
	}

	if !Exist(dirpath + filename) {
		_, err := os.Create(dirpath + filename)
		if err != nil {
			fmt.Println("Fail to create file!")
			return nil
		}
	}
	//initial fileLoader
	newFileLoader := NewFileLoader(dirpath + filename)
	newRamFileLoader := &RamFileLoader{
		dirpath:  dirpath,
		filename: filename,
		File:     newFileLoader,
	}
	return newRamFileLoader
}

func (rfl *RamFileLoader) DeleteFolder() {
	//delete the file
	err := os.Remove(rfl.dirpath + rfl.filename)
	if err != nil {
		fmt.Println("Fail to delete file!")

	}

}
