package main

import (
	"github.com/ajuda-dev/backend/src/config"
)

// @title AjudaDev Backend API
// @version 1.0
// @description Sessão por cookie HttpOnly (ajudadev_session): login, registro e callback OAuth gravam o cookie e o navegador o envia sozinho — o JavaScript nunca lê o token. Apps nativos usam Authorization: Bearer com o token obtido via header X-Client-Type: native no login/registro.
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	config.InitApp()
}
