package main

import (
	_ "embed"
	"html/template"
	"log"
	"os"
)

//go:embed _template/preview.html
var tpl string

func main() {
	data := struct {
		Title   string
		Content string
	}{
		Title:   "My page",
		Content: "This is a test",
	}

	check := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	t, err := template.New("preview").Parse(tpl)
	check(err)
	err = t.Execute(os.Stdout, data)
	check(err)
}
