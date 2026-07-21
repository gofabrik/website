// Command site generates gofabrik.dev from the site/ sources.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gofabrik/website/internal/tools/site"
)

func main() {
	root := flag.String("root", ".", "site root: the directory containing site/ and receiving the output")
	addr := flag.String("addr", ":8080", "listen address for run")
	flag.Parse()

	var err error
	switch flag.Arg(0) {
	case "build":
		err = site.Build(*root)
	case "run":
		err = site.Serve(*root, *addr)
	default:
		fmt.Fprintln(os.Stderr, "usage: site [-root dir] [-addr :8080] build|run")
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "site:", err)
		os.Exit(1)
	}
}
