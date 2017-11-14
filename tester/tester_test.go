package tester

import (
	"testing"
)

func TestSimple(t *testing.T) {
	value := GoverallsTester()
	expected := "hello world"
	if value != expected {
		t.Fatalf("Expected %v, but %v:", value, expected)
	}
}
