package main

import (
	"github.com/ajuda-dev/backend/src/config"
)

// @title AjudaDev Backend API
// @version 1.0
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	config.InitApp()
}