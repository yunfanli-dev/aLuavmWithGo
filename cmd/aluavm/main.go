package main

import (
	"fmt"
	"os"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/api"
)

func main() {
	vm := api.NewVM()
	vm.SetOutput(os.Stdout)

	if err := run(vm, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "aluavm bootstrap failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("aluavm bootstrap ready")
}

// run dispatches CLI execution between bootstrap file loading and empty-script validation.
func run(vm *api.VM, args []string) error {
	if len(args) == 0 {
		return vm.ExecString("")
	}

	// TODO: Extend the CLI to support inline source execution and richer runtime flags.
	return vm.ExecFile(args[0])
}
