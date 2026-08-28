package main

import (
	"fmt"
	"erp-azhan/api/internal/identity"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	acc, _ := identity.GenerateAccessToken(1, nil)
	fmt.Println(acc)
}
