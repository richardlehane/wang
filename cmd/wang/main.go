package main

import (
	"fmt"
	"os"

	"github.com/richardlehane/wang"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Print("Please supply a command (meta or dump) and target\n")
		os.Exit(1)
	}
	f, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rdr, err := wang.New(f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "meta":
		for _, f := range rdr.Files {
			fmt.Println(f)
		}
	case "dump":
		err = rdr.DumpSectors()
	case "files":
		err = rdr.DumpFiles()
	case "scan":
		mp := rdr.Scan()
		for k, v := range mp {
			fmt.Printf("%s:\n", k.String())
			fmt.Printf("List (%d):\n%v\n\nInternal list (%d):\n%v\n\nRaw sectors (%d):\n%v\n\n\n",
				len(v[0]), v[0], len(v[1]), v[1], len(v[2]), v[2])
		}
	default:
		fmt.Print("Invalid command must be meta or dump\n")
		os.Exit(1)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
