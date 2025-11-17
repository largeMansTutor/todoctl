package web

import (
	"github.com/thetrollfarmercodes/todoctl/todo/pkg/httpadapter"
)

func (h *App) Mount(router httpadapter.Router) {
	router.Group(func(protected httpadapter.Router) {
		protected.Use(httpadapter.APIKeyAuth(h.cfg.APIKey, h.logger))
		if h.rateLimiter != nil {
			protected.Use(h.rateLimiter.Middleware)
		}
		protected.Get("/todos/{id}", h.GetTodo)
		protected.Post("/todos", h.CreateTodos)
		protected.Patch("/todos", h.UpdateTodos)
		protected.Get("/todos", h.ListTodos)
	})
}
