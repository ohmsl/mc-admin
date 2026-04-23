package main

import (
	"log"

	"github.com/ohmsl/mc-admin/apps/pb/internal/routes"
	"github.com/pocketbase/pocketbase"
)

func main() {
	app := pocketbase.New()

	if err := routes.Register(app); err != nil {
		log.Fatal(err)
	}

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
