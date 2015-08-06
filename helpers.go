package main

import (
	"fmt"
)

func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}
