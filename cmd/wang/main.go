package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/richardlehane/wang"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Print("Please supply a command (meta, csv, files, text, rtf, dump or fix) and target\n")
		os.Exit(1)
	}
	f, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if os.Args[1] == "fix" {
		rdr, err := wang.Fix(f)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		ext := filepath.Ext(os.Args[2])
		rdr.WriteFile(strings.TrimSuffix(os.Args[2], ext) + "_fix" + ext)
		os.Exit(0)
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
	case "csv":
		c := csv.NewWriter(os.Stdout)
		err = c.Write([]string{"Archive ID",
			"Document Name",
			"Author",
			"Operator",
			"Comments",
			"Created",
			"Modified",
		})
		if err == nil {
			for _, f := range rdr.Files {
				err = c.Write(f.CSV())
				if err != nil {
					break
				}
			}
			c.Flush()
		}
	case "files":
		err = rdr.DumpFiles("")
	case "rtf":
		err = rdr.DumpRTF("")
	case "text":
		err = rdr.DumpText("")
	case "dump":
		err = rdr.DumpSectors("")
	default:
		fmt.Print("Invalid command must be meta, csv, files, text, rtf, dump or fix\n")
		os.Exit(1)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
