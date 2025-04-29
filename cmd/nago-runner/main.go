// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}

func realMain() error {
	if len(os.Args[1:]) == 0 {
		return runService()
	}

	if runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	switch os.Args[len(os.Args)-1] {
	case "configure":
		return configure()
	case "install":
		return install()
	default:
		return fmt.Errorf("unknown command %s", flag.Args()[0])
	}
}
