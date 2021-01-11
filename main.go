package main

import (
	"log"
	"net/http"

	"github.com/bokwoon95/erro"
	"github.com/bokwoon95/pagemanager-data/pagemanager"
	"github.com/go-chi/chi"
)

func main() {
	pm, err := pagemanager.New()
	if err != nil {
		log.Fatalln(erro.Sdump(err))
	}
	mux := chi.NewRouter()
	mux.Use(pm.Middleware)
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})
	pm.ListenAndServe(":80", mux)
}
