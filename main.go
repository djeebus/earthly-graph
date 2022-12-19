package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/earthly/earthly/ast"
	"github.com/earthly/earthly/ast/spec"
)

type Graph map[string]map[string]struct{}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s FILENAME", os.Args[0])
		os.Exit(1)
		return
	}

	filename := os.Args[1]
	earthfile, err := ast.Parse(context.TODO(), filename, true)
	if err != nil {
		fmt.Printf("failed to parse file:\n%v", err)
		os.Exit(1)
		return
	}

	graph := make(Graph)

	for _, target := range earthfile.Targets {
		graph[target.Name] = make(map[string]struct{})

		processBlock(target, target.Recipe, graph)
	}

	println("graph TD")
	for name, targets := range graph {
		for target := range targets {
			fmt.Printf("\t%s --> %s\n", name, target)
		}
	}
}

func processBlock(target spec.Target, block spec.Block, graph Graph) {
	for _, item := range block {
		if item.Command != nil {
			processCommand(target, graph, *item.Command)
		}

		if item.Wait != nil {
			processBlock(target, item.Wait.Body, graph)
		}

		if item.For != nil {
			processBlock(target, item.For.Body, graph)
		}

		if item.If != nil {
			processBlock(target, item.If.IfBody, graph)
		}
	}
}

func processCommand(target spec.Target, graph Graph, command spec.Command) {
	switch command.Name {

	// these commands can't have dependencies
	case "ARG", "ENTRYPOINT", "ENV", "FROM DOCKERFILE", "RUN", "SAVE ARTIFACT", "SAVE IMAGE", "WORKDIR":
		return

	// these commands might have dependencies
	case "BUILD", "COPY", "FROM":
		findAndAddDependencies(command.Args, graph, target)
	default:
		panic(command.Name)
	}
}

func findAndAddDependencies(args []string, graph Graph, target spec.Target) {
	for _, arg := range args {
		if len(arg) < 1 || arg[0] != '+' {
			continue
		}

		dep := arg[1:]
		dep = strings.Split(dep, "/")[0]
		graph[target.Name][dep] = struct{}{}
	}
}
