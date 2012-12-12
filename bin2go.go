package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	name = flag.String("name", "", "use this name for the variable instead of one generated based on the input")
	out  = flag.String("out", "", "use this filename for output instead of inputfile.go")
	pkg  = flag.String("pkg", "", "use this package name instead of the parent directory of the input")
)

// Exit error codes
const (
	NO_ERROR = iota
	WRONG_ARGS
	INPUT_FAIL
	OUTPUT_FAIL
)

func printUsage() {
	fmt.Printf("usage: %s [-name=<name>] [-out=<path>] [-pkg=<name>] <inputfile>\n\t%s [-pkg=<name>] <inputfile>...", os.Args[0], os.Args[0])
	flag.PrintDefaults()
}

func printUsageAndExit() {
	flag.Usage()
	os.Exit(WRONG_ARGS)
}

func readInput(filename string) []byte {
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %s\n", err)
		os.Exit(INPUT_FAIL)
	}
	return data
}

func checkOutputFailure(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %s\n", err)
		os.Exit(OUTPUT_FAIL)
	}
}

func writeData(filename string, data []byte, out io.Writer) {
	var varname string

	if *name != "" {
		varname = *name
	} else {
		pieces := strings.FieldsFunc(filepath.Base(filename), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})

		for _, piece := range pieces {
			p := []rune(piece)
			varname += string(unicode.ToUpper(p[0])) + string(p[1:])
		}
	}

	// write header
	_, err := fmt.Fprintf(out, "var %s = [...]byte{\n\t", varname)
	checkOutputFailure(err)

	lastbytei := len(data) - 1
	n := 8
	for i, b := range data {
		// write single byte
		_, err = fmt.Fprintf(out, "0x%.2x,", b)
		checkOutputFailure(err)

		n += 6

		// if this is not the last byte
		if i != lastbytei {
			// be readable, break line after 78 characters
			if n >= 78 {
				_, err = fmt.Fprint(out, "\n\t")
				checkOutputFailure(err)

				n = 8
			} else {
				// if we're not breaking the line, insert space
				// after ','
				_, err = fmt.Fprint(out, " ")
				checkOutputFailure(err)
			}

		}
	}
	_, err = fmt.Fprint(out, "\n}\n")
	checkOutputFailure(err)
}

func writeOutput(filename string, data []byte) {
	// prepare output file
	if *out == "" {
		*out = filename + ".go"
	}
	file, err := os.Create(*out)
	checkOutputFailure(err)
	defer file.Close()

	output := bufio.NewWriter(file)

	// write package clause if any
	if *pkg == "" {
		path, err := filepath.Abs(*out)
		checkOutputFailure(err)
		*pkg = filepath.Base(filepath.Dir(path))
	}
	_, err = fmt.Fprintf(output, "package %s\n\n", *pkg)
	checkOutputFailure(err)

	// write data
	writeData(filename, data, output)

	// flush
	err = output.Flush()
	checkOutputFailure(err)
}

func main() {
	flag.Usage = printUsage
	flag.Parse()

	if flag.NArg() == 0 || (flag.NArg() > 1 && (*out != "" || *name != "")) {
		printUsageAndExit()
	}

	for _, filename := range flag.Args() {
		data := readInput(filename)
		writeOutput(filename, data)
	}
}
