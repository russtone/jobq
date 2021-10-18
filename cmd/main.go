package main

import (
	_ "embed"
	"flag"
	"log"
	"os"
	"text/template"
	"time"
)

//go:embed ../jobq.go
var jobqTemplate string

type Options struct {
	Package   string
	Task      string
	Result    string
	CreatedAt string
}

func main() {
	var options Options

	flag.StringVar(&options.Package, "package", "", "package name")
	flag.StringVar(&options.Task, "task", "", "task type name")
	flag.StringVar(&options.Result, "result", "", "result type name")

	flag.Parse()

	if options.Package == "" ||
		options.Task == "" ||
		options.Result == "" {
		log.Fatal("error: flags required")
	}

	tpl, err := template.New("").Parse(jobqTemplate)
	if err != nil {
		log.Fatal(err)
	}

	options.CreatedAt = time.Now().Format(time.RFC1123)

	out, err := os.OpenFile("jobq.go", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}

	if err := tpl.Execute(out, options); err != nil {
		log.Fatalf("template execution failed: %v", err)
	}
}
