package main

import (
	"net/http"
	"os"

	"github.com/asecurityteam/vpcflow-diffd/pkg"
	"github.com/asecurityteam/vpcflow-diffd/pkg/domain"
	"github.com/asecurityteam/vpcflow-diffd/pkg/plugins"
	"github.com/go-chi/chi"
)

func main() {
	router := chi.NewRouter()
	middleware := []func(http.Handler) http.Handler{
		plugins.DefaultLogMiddleware(),
		plugins.DefaultStatMiddleware(),
	}
	service := &diffd.Service{
		Middleware: middleware,
	}
	if err := service.BindRoutes(router); err != nil {
		panic(err.Error())
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	r := &diffd.Runtime{
		Server: server,
		ExitSignals: []domain.ExitSignal{
			plugins.OS,
		},
	}

	if err := r.Run(); err != nil {
		panic(err.Error())
	}
}
