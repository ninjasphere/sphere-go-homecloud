package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-martini/martini"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/sphere-go-homecloud/models"
	"github.com/ninjasphere/sphere-go-homecloud/state"
)

type SiteRouter struct {
}

func NewSiteRouter() *SiteRouter {
	return &SiteRouter{}
}

func (lr *SiteRouter) Register(r martini.Router) {

	r.Get("", lr.GetAll)
	r.Get("/:id", lr.GetSite)
	r.Put("/:id", lr.PutSite)
	r.Delete("/:id", lr.DeleteSite)

}

// GetAll retrieves a list of all sites
func (lr *SiteRouter) GetAll(r *http.Request, w http.ResponseWriter, siteModel *models.SiteModel, stateManager state.StateManager) {
	sites, err := siteModel.FetchAll()

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve sites", http.StatusInternalServerError, w)
		return
	}

	WriteServerResponse(sites, http.StatusOK, w)
}

// GetSite retrieves a site using it's identifier
func (lr *SiteRouter) GetSite(params martini.Params, w http.ResponseWriter, siteModel *models.SiteModel, stateManager state.StateManager) {

	site, err := siteModel.Fetch(params["id"])

	log.Infof(spew.Sprintf("site: %v", site))

	if err == models.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown site id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve site", http.StatusInternalServerError, w)
		return
	}

	WriteServerResponse(site, http.StatusOK, w)
}

// GetAll updates a site using it's identifier, with the JSON payload containing name and type
func (lr *SiteRouter) PutSite(params martini.Params, r *http.Request, w http.ResponseWriter, siteModel *models.SiteModel) {

	var site *model.Site

	err := json.NewDecoder(r.Body).Decode(&site)

	if err != nil {
		WriteServerErrorResponse("Unable to parse body", http.StatusInternalServerError, w)
		return
	}

	err = siteModel.Update(params["id"], site)

	if err != nil {
		WriteServerErrorResponse("Unable to update site", http.StatusInternalServerError, w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteSite removes a site using it's identifier
func (lr *SiteRouter) DeleteSite(params martini.Params, w http.ResponseWriter, siteModel *models.SiteModel) {

	err := siteModel.Delete(params["id"])

	if err == models.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown site id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to delete site", http.StatusInternalServerError, w)
		return
	}

	w.WriteHeader(http.StatusOK) // TODO: talk to theo about this response.
}
