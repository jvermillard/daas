package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestRandString(t *testing.T) {
	value := randString(5)
	fmt.Println(value)

	if len(value) != 5 {
		t.Log("Expected 5")
		t.Fail()
	}

	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for i := 0; i < len(value); i++ {
		if strings.IndexBytE(alphanum, value[i]) == -1 {
			t.Fail()
		}
	}
}
