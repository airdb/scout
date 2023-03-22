package bootstrap

import (
	"context"
	"net/http"

	wecommod "github.com/airdb/scout/modules/wecom"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"go.uber.org/fx"
)

type wecomDeps struct {
	fx.In

	Handler *wecommod.Handler
}

type Wecom struct {
	deps   *wecomDeps
	mux    *chi.Mux
	server *http.Server
}

func NewWecom(deps wecomDeps) *Wecom {
	mux := chi.NewRouter()
	mux.Use(middleware.Logger)
	mux.Use(render.SetContentType(render.ContentTypeHTML))

	return &Wecom{deps: &deps, mux: mux}
}

func (w *Wecom) Start() error {
	w.mux.Route("/wecom/kf", func(r chi.Router) {
		r.HandleFunc("/callback", w.deps.Handler.GetCallback)
		r.Get("/accounts", w.deps.Handler.GetAccounts)
	})

	w.server = &http.Server{Addr: "0.0.0.0:30120", Handler: w.mux}

	// Run the server
	return w.server.ListenAndServe()
}

func (w *Wecom) Stop() error {
	return w.server.Shutdown(context.TODO())
}
