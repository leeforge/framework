package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	chi "github.com/go-chi/chi/v5"
)

// PrintJson prints the json string of the given value.
func PrintJson(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// PrintRoutes prints all the registered routes in a given chi.Router to the console.
func PrintRoutes(r chi.Router) {
	fmt.Println("\n=== Registered Routes ===")
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("%-6s %s\n", method, strings.Replace(route, "/*/", "/", -1))
		return nil
	}
	if err := chi.Walk(r, walkFunc); err != nil {
		fmt.Printf("Error walking routes: %v\n", err)
	}
	fmt.Println("========================")
}
