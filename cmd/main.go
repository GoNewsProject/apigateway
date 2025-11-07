package main

import (
	"apigateway/internal/app"
)

func main() {
	err := app.Run("configs/dev.yaml")
	if err != nil {
		panic(err)
	}
}
