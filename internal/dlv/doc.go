// Copyright 2024 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// Package dlv allows to run any binaries with latest version of delve.
//
// To make it works:
//  1. compile your code with
//     -gcflags "all=-N -l"
//  2. Use [NewDlvRunner] function to encapsulate your main function.
//
// # Tips
//
// Don't bring delve in production: use compile tags
//
// Refactor your code in such way you can easily call the main workload as a one-liner in `main()` function.
// Then use compile tag to ensure that the actual main function will wrap the workload in delve only if the debug flag
// is used at compile time:
//
//	go build -gcflags "all=-N -l" -tags debug path/to/my/package
//
// It can be done through having two main files in package main, one named `main.go` and one name `main_debug.go`. The
// former will be compiled only if there is no debug tag, the latter only if there is a debug tag. Making mistake with
// tags would end up with two main functions, which will cause a compile error (better for avoiding shipment of debug
// binaries in production).
//
// # Example
//
// main_debug.go
//
//	//go:build debug
//
//	package main
//
//	 import (
//	 "github.com/juju/juju/internal/dlv"
//	 "github.com/juju/juju/internal/dlv/config"
//	 )
//
//	func main() {
//	   os.Exit(dlv.NewDlvRunner(dlv.NewDlvRunner(
//	   dlv.Headless(),
//	   dlv.WithApiVersion(2),
//	   dlv.WithPort(1122),
//	   dlv.WaitDebugger()))(mainArgs)(os.Args))
//	}
//
// main.go
//
//	//go:build !debug
//
//	package main
//
//	import (
//	   "os"
//	)
//
//
//	func main() {
//	   os.Exit(mainArgs(os.Args))
//	}
//
//	func mainArgs(args []string) int { /* ... */ }
package dlv
