package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-version" || os.Args[1] == "version" || os.Args[1] == "-v") {
		buildInfo := GetBuildInfo()
		fmt.Println(buildInfo)
		os.Exit(0)
	}

	fmt.Println("Hello, World!")
}
