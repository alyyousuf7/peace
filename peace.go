package peace

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	Pass  = "Pass"
	Fail  = "Fail"
	Panic = "Panic"
)

type Test struct {
	Name   string
	Status string
}

func (t Test) String() string {
	return fmt.Sprintf("[%s] %s", strings.ToLower(t.Status), t.Name)
}

type Result struct {
	Package string
	Tests   []Test
}

func (r Result) String() string {
	var buf bytes.Buffer
	for _, test := range r.Tests {
		buf.WriteString(fmt.Sprintf("%s \n", test))
	}

	return fmt.Sprintf("\n%s: [%d]\n%s", r.Package, len(r.Tests), buf.String())
}

func Do(pkg string, tags string, logging bool) (*Result, error) {
	if logging {
		log.Printf("Package: %s\n", pkg)
	}
	path := fmt.Sprintf("%s/src/%s", os.Getenv("GOPATH"), pkg)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		if logging {
			log.Println("Error while reading directory")
		}
		return nil, err
	}

	args := []string{"test"}
	if tags != "" {
		args = append(args, fmt.Sprintf("-tags %s", tags))
	}

	cmds := [][]string{}
	for _, file := range files {
		if !file.IsDir() && strings.Contains(file.Name(), "_test.go") {
			if logging {
				log.Printf("File: %s\n", file.Name())
			}

			raw, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", path, file.Name()))
			if err != nil {
				if logging {
					log.Println("Error while reading file")
				}
				return nil, err
			}

			// default
			r0 := regexp.MustCompile(`func\s(Test[A-Za-z0-9_]+)`)
			for _, match := range r0.FindAllStringSubmatch(string(raw), -1) {
				cmds = append(cmds, append(args, pkg, "-run", match[1]))
			}

			// gocheck
			r1 := regexp.MustCompile(`func\s\((?:.*\*([A-Za-z0-9_]+))\)\s(Test[A-Za-z0-9_]+)`)
			for _, match := range r1.FindAllStringSubmatch(string(raw), -1) {
				cmds = append(cmds, append(args, pkg, "-check.f", match[1]+"."+match[2]))
			}
		}
	}

	result := new(Result)
	result.Package = pkg

	for index, args := range cmds {
		if logging {
			log.Printf("Executing [%d/%d]: `go %s`\n", index+1, len(cmds), strings.Join(args, " "))
		}

		name := args[len(args)-1]
		status := Pass

		var out bytes.Buffer

		cmd := exec.Command("go", args...)
		cmd.Stdout = &out
		cmd.Env = os.Environ()

		if cmd.Run() != nil {
			r := regexp.MustCompile(`(?i)panic:.*`)
			if r.MatchString(out.String()) {
				status = Panic
				goto APPEND
			}
			status = Fail
		}
	APPEND:
		if logging && status != Pass {
			log.Println(out.String())
		}
		result.Tests = append(result.Tests, Test{name, status})
	}

	return result, nil
}
