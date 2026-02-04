// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package main

import (
	"os"

	"github.com/mtreilly/arc-ai/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
