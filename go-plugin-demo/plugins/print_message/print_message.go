package main

import (
	"fmt"
)

type PrintMessagePlugin struct{}

func (p *PrintMessagePlugin) Run() error {
	fmt.Println("Hello from the PrintMessage plugin!")
	return nil
}

var Plugin PrintMessagePlugin
