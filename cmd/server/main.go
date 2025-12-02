package main

import (
	"log"

	"github.com/gabkaclassic/GitQuest/internal/app"
)

func main() {
	a, err := app.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
