// E2E fixture: runs the syralit dev supervisor on a project directory. The
// e2e test needs the supervisor as a separate, killable process to assert the
// dev window auto-quits when the supervisor dies.
//
// Usage: devsup <project-dir> <port>
package main

import (
	"log"
	"os"
	"strconv"

	sy "github.com/HazelnutParadise/syralit"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: devsup <project-dir> <port>")
	}
	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("bad port: %v", err)
	}
	log.Fatal(sy.RunDev(sy.DevOptions{Dir: os.Args[1], Port: port}))
}
