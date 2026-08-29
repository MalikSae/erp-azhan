package main

import (
	"erp-azhan/api/internal/identity"
	"fmt"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	brandID := int64(2)
	acc, _ := identity.GenerateAccessToken(8, &brandID, "admin")
	fmt.Println(acc)
}
