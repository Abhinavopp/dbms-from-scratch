package chapter00

import "testing"

func TestIntro(t *testing.T) {
	msg := Intro()
	if msg == "" {
		t.Fatal("intro message should not be empty")
	}
}
