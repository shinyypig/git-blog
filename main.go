package main

import (
	"fmt"
	"log"
	"net/http"

	"shinyypig/gitblog/gitkit"
	"shinyypig/gitblog/router"
)

func main() {
	// Configure git hooks
	hooks := &gitkit.HookScripts{
		PreReceive: `echo "Hello World!"`,
	}

	// Configure git service
	service := gitkit.New(gitkit.Config{
		Dir:        "data",
		AutoCreate: true,
		AutoHooks:  true,
		Hooks:      hooks,
	})

	// Configure git server. Will create git repos path if it does not exist.
	// If hooks are set, it will also update all repos with new version of hook scripts.
	if err := service.Setup(); err != nil {
		log.Fatal(err)
	}

	blogServer := router.BlogServer()
	http.Handle("/", service)

	// Watch data directory for changes
	watch := router.NewNotifyFile()
	watch.WatchDir("data")

	// Start HTTP server
	fmt.Println("Starting server on http://")
	if err := http.ListenAndServe("127.0.0.1:8080", blogServer); err != nil {
		log.Fatal(err)
	}
}
