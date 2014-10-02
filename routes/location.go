package routes

import "github.com/go-martini/martini"

type LocationRouter struct {
}

func NewLocationRouter() *LocationRouter {
	return &LocationRouter{}
}

func (lr *LocationRouter) Register(r martini.Router) {

}
