package scEngine

import (
	"testing"
)

func TestScEngine_Execute(t *testing.T) {
	sc := NewScEngine("3 + 18")
	sc.Execute()
}