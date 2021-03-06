package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/pflag"
)

type parseGenData struct {
	PackageName    string
	InterpFunc     string
	InterpFuncName string
	OpCodeT        string
	InstructT      string
	FuncName       string
	NumCaptures    int
	Instructions   []Inst
}

var packageName = pflag.String("package", "main", "package name for generated code")
var regexPattern = pflag.String("pattern", "", "regex to generate code for")
var funcName = pflag.String("func", "RegexMatch",
	"function name for regex matching function")
var outFile = pflag.String("out", "", "output file for generated code")

const instruct = `
type Inst struct {
	Op     OpCode
	Char   rune
	Label1 int64
	Label2 int64
}`

const opcode = `
type OpCode uint
const (
	Char OpCode = iota
	Match
	Jump
	Split
	Save
	Nop
)`

var genTemplate = template.Must(template.New("matcher").Parse(`
package {{.PackageName}}
// Code autogenerated, don't touch
{{.OpCodeT}}

{{.InstructT}}

const NumCaptures = {{.NumCaptures}}
// {{.InterpFunc}}

func {{.FuncName}}(input string) (bool, []string) {
	in := []Inst{ {{range .Instructions}}
	    Inst{ {{.Op}}, {{.Char}}, {{.Label1}}, {{.Label2}}},
	{{end}} }

	return {{.InterpFuncName}}(in, input)
}`))

func genMatcher(regex string, w io.Writer) error {
	parser := new(regexParser)
	if inst, err := parser.Parse(regex); err != nil {
		return err
	} else {
		var thompson []byte
		thompson, err = Asset("run_test.go")
		if err != nil {
			return err
		}

		tmp := strings.Split(string(thompson), "\n")
		var vmSrc string
		for i, str := range tmp {
			if strings.Contains(str, "go:generate") {
				vmSrc = strings.Join(tmp[i+1:], "\n")
				break
			}
		}

		captures := strings.Count(regex, "(") * 2
		return genTemplate.Execute(w, parseGenData{
			PackageName:    *packageName,
			InterpFunc:     vmSrc,
			InterpFuncName: "ThompsonVM",
			OpCodeT:        opcode,
			InstructT:      instruct,
			FuncName:       *funcName,
			NumCaptures:    captures,
			Instructions:   finalizeInst(inst.compile()),
		})
	}
}

func main() {
	pflag.Parse()
	if *outFile == "" {
		*outFile = fmt.Sprintf("%s_regex", *funcName)
	}
	file, err := os.OpenFile(*outFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		file, err = os.Create(*outFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	genMatcher(*regexPattern, file)
}
