package main

import (
	"fmt"
	"log"
	"os"
	"plugin"
)

type Plugin interface {
    Run() error
}

func main() {
    if len(os.Args) != 2 {
        log.Fatalf("Usage: %s [plugin_file]", os.Args[0])
    }

    pluginFile := os.Args[1]
    plg, err := plugin.Open(pluginFile)
    if err != nil {
        log.Fatalf("Failed to open plugin: %s", err)
    }

    symbol, err := plg.Lookup("Plugin")
    if err != nil {
        log.Fatalf("Failed to find plugin symbol 'Plugin': %s", err)
    }

    pluginInstance, ok := symbol.(Plugin)
    if !ok {
        log.Fatal("Symbol 'Plugin' does not implement the Plugin interface")
    }

    if err := pluginInstance.Run(); err != nil {
        log.Fatalf("Plugin execution failed: %s", err)
    }

    fmt.Println("Plugin executed successfully")
}