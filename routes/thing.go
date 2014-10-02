package routes

import "github.com/go-martini/martini"

type ThingRouter struct {
}

func NewThingRouter() *ThingRouter {
	return &ThingRouter{}
}

func (lr *ThingRouter) Register(r martini.Router) {

}
