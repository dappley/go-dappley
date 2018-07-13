package storage

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testFile = "../bin/test.dat"

func TestSaveToFile(t *testing.T) {
	SaveToFile(testFile, bytes.Buffer{})
}

func TestGetFileConnection(t *testing.T) {
	fileContent, err := GetFileConnection(testFile)
	assert.Nil(t, err)
	assert.NotNil(t, fileContent)
}
