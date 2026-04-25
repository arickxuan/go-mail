package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"mail0/handlers"
	"mail0/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := flag.String("addr", ":"+port, "listen address")
	flag.Parse()

	storePath := filepath.Join(".", "accounts.json")
	s, err := store.New(storePath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	if err := os.MkdirAll("static", 0755); err != nil {
		log.Printf("Warning: could not create static dir: %v", err)
	}

	r := gin.Default()

	h := handlers.New(s)
	api := r.Group("/api")
	h.RegisterAPIRoutes(api)

	fs := http.FileServer(http.Dir("./web"))

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		fs.ServeHTTP(c.Writer, c.Request)
	})

	log.Printf("Mail0 starting on %s", *addr)
	if err := r.Run(*addr); err != nil {
		log.Fatal(err)
	}
}
