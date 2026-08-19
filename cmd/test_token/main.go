package main

import (
	"fmt"
	"erp-azhan/api/internal/identity"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	brandID := int64(2)
	acc, _ := identity.GenerateAccessToken(8, &brandID)
	fmt.Println(acc)
}
