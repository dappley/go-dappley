package contract

import (
	"testing"
)

func TestScEngine_Execute(t *testing.T) {
	sc := NewScEngine("3 + 115")
	sc.Execute()
}