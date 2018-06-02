package main

import (
	"fmt"
	"testing"
)

func TestPlaceholder(t *testing.T){
	fmt.Println("Running TestPlaceholder...")
	if 1 != 1 {
		t.Error("1 != 1")
	}
}
