package main

import (
	"log"
	"net/http"

	"github.com/bokwoon95/erro"
	"github.com/bokwoon95/pagemanager-data/pagemanager"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	pm, err := pagemanager.New()
	if err != nil {
		log.Fatalln(erro.Sdump(err))
	}
	mux := chi.NewRouter()
	mux.Use(middleware.Compress(5))
	mux.Use(pm.Middleware)
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<h1>hello world</h1>"))
	})
	// render, err := renderly.New(
	// 	os.DirFS(renderly.AbsDir(".")),
	// 	renderly.AltFS("builtin", os.DirFS(renderly.AbsDir("pagemanager"))),
	// 	renderly.GlobalHTMLEnvFuncs(pagemanager.EnvFunc),
	// 	renderly.GlobalJSEnvFuncs(pagemanager.EnvFunc),
	// 	renderly.TemplateFuncs(pm.FuncMap()),
	// )
	// if err != nil {
	// 	log.Fatalln(erro.Sdump(err))
	// }
	// mux.Get("/test", func(w http.ResponseWriter, r *http.Request) {
	// 	err := render.Page(w, r, "templates/plainsimple/post-index.html", []string{
	// 		"builtin::tachyons.min.css",
	// 		"templates/plainsimple/header.html",
	// 		"templates/plainsimple/footer.html",
	// 		"templates/plainsimple/style.css",
	// 		"templates/plainsimple/post-index.js",
	// 	}, nil)
	// 	if err != nil {
	// 		http.Error(w, erro.Sdump(err), http.StatusInternalServerError)
	// 	}
	// })
	pm.ListenAndServe(":80", mux)
}
