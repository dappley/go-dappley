package storage

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	confDir = "fakeFileLoaders/"
)

func deleteConfFolderFiles() error {
	dir, err := ioutil.ReadDir(confDir)
	if err != nil {
		return err
	}
	for _, d := range dir {
		os.RemoveAll(path.Join([]string{confDir, d.Name()}...))
	}
	return nil
}

func TestRamFileLoader_Create(t *testing.T) {
	_ = NewRamFileLoader(confDir, "test.conf")
	targetFilename := confDir + "test.conf"
	flag := Exist(targetFilename)
	assert.Equal(t, flag, true)
}

func TestRamFileLoader_Close(t *testing.T) {
	rfl := NewRamFileLoader(confDir, "test.conf")
	targetFilename := confDir + "test.conf"
	err := rfl.DeleteFolder()
	flag := Exist(targetFilename)
	assert.Nil(t, err)
	assert.Equal(t, flag, false)
}

func TestRamFileLoader_CreateAndRemoveAll(t *testing.T) {
	_ = NewRamFileLoader(confDir, "test1.conf")
	_ = NewRamFileLoader(confDir, "test2.conf")
	flag1 := Exist(confDir+"test1.conf") && Exist(confDir+"test2.conf")
	assert.Equal(t, flag1, true)
	err := deleteConfFolderFiles()
	flag2 := Exist(confDir+"test1.conf") || Exist(confDir+"test2.conf")
	assert.Equal(t, flag2, false)
	assert.Nil(t, err)
}
