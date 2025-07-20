/*
Copyright © 2025 chickiexd
*/
package main

import (
	"github.com/chickiexd/ez_copy/cmd"
	"github.com/chickiexd/ez_copy/logger"
)

func main() {
	logger.Init()
	defer logger.Sync()
	cmd.Execute()
}
