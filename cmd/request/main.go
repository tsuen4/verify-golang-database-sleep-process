package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
)

/*
command example:

	"go run cmd/request/main.go <path> [<count>]"
	"go run cmd/request/main.go select-ctx 10"
*/
func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	path := ""
	if len(args) > 1 {
		path = args[1]
	}

	count := 1
	if len(args) > 2 {
		var err error
		count, err = strconv.Atoi(args[2])
		if err != nil {
			count = 1
		}
	}

	client := &http.Client{}
	for i := 0; i < count; i++ {
		_, err := client.Get("http://localhost:8080/" + path)
		if err != nil {
			return fmt.Errorf("client.Get: %w", err)
		}
	}

	return nil
}
