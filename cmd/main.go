package main

import "commentservice/internal/app"

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
