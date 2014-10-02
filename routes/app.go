package routes

import "github.com/go-martini/martini"

type AppRouter struct {
}

func NewAppRouter() *AppRouter {
	return &AppRouter{}
}

func (lr *AppRouter) Register(r martini.Router) {

}
