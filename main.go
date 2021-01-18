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
	render, err := renderly2.New(
		os.DirFS(renderly2.AbsDir(".")),
		renderly2.GlobalHTMLEnvFuncs(pagemanager.EnvFunc),
		renderly2.GlobalJSEnvFuncs(pagemanager.EnvFunc),
		renderly2.TemplateFuncs(pm.FuncMap()),
	)
	if err != nil {
		log.Fatalln(erro.Sdump(err))
	}
	mux := chi.NewRouter()
	mux.Use(pm.Middleware)
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})
	mux.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		err := render.Page(w, r, "templates/plainsimple/post-index.html", []string{
			"pagemanager/tachyons.min.css",
			"templates/plainsimple/header.html",
			"templates/plainsimple/footer.html",
			"templates/plainsimple/style.css",
			"templates/plainsimple/post-index.js",
		}, nil)
		if err != nil {
			http.Error(w, erro.Sdump(err), http.StatusInternalServerError)
		}
	})
	pm.ListenAndServe(":80", mux)
}
