package storage

import(
	"os"
	"fmt"
)

type RamFileLoader struct {
	dirpath   string
	filename  string
	File      *FileLoader
}

func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func NewRamFileLoader(dirpath string, filename string) *RamFileLoader {
	//create fake files in fakeFileLoaders directory if not exist
	if !Exist(dirpath + filename) {
		_, err := os.Create(dirpath + filename)
		if err != nil {
			fmt.Println("Fail to create file!")
			return nil
		}
	}
	//initial fileLoader
	newFileLoader := NewFileLoader(dirpath + filename)
	newRamFileLoader :=  &RamFileLoader{
		dirpath: dirpath,
		filename: filename,
		File: newFileLoader,
	}
	return newRamFileLoader
}

func (rfl *RamFileLoader) Close() error {
	//delete the file
	err := os.Remove(rfl.dirpath + rfl.filename)
	if err != nil {
		fmt.Println("Fail to delete file!")
		return err
	}
	return nil
}




