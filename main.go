package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bokwoon95/erro"
	"github.com/bokwoon95/pagemanager-data/pagemanager"
	"github.com/bokwoon95/pagemanager-data/renderly2"
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
	render, err := renderly2.New(os.DirFS(renderly2.AbsDir(".")))
	if err != nil {
		log.Fatalln(erro.Sdump(err))
	}
	mux.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		// render.
	})
	pm.ListenAndServe(":80", mux)
}
