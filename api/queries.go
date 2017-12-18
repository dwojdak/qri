package api

import (
	"encoding/json"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/repo"
)

// QueryHandlers wraps a requests struct to interface with http.HandlerFunc
type QueryHandlers struct {
	core.QueryRequests
	log logging.Logger
}

// NewQueryHandlers allocates a new QueryHandlers pointer
func NewQueryHandlers(log logging.Logger, r repo.Repo) *QueryHandlers {
	req := core.NewQueryRequests(r, nil)
	return &QueryHandlers{*req, log}
}

// ListHandler is the endpoint for listing this repo's log of queries
func (h *QueryHandlers) ListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.listQueriesHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// func (d *QueryHandlers) GetHandler(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	case "OPTIONS":
// 		util.EmptyOkHandler(w, r)
// 	case "GET":
// 		d.getDatasetHandler(w, r)
// 	default:
// 		util.NotFoundHandler(w, r)
// 	}
// }

// func (d *QueryHandlers) getDatasetHandler(w http.ResponseWriter, r *http.Request) {
// 	res := &dataset.Dataset{}
// 	args := &GetParams{
// 		Path: r.URL.Path[len("/queries/"):],
// 		Hash: r.FormValue("hash"),
// 	}
// 	err := d.Get(args, res)
// 	if err != nil {
// 		util.WriteErrResponse(w, http.StatusInternalServerError, err)
// 		return
// 	}
// 	util.WriteResponse(w, res)
// }

func (h *QueryHandlers) listQueriesHandler(w http.ResponseWriter, r *http.Request) {
	args := core.ListParamsFromRequest(r)
	args.OrderBy = "created"
	res := []*repo.DatasetRef{}
	err := h.List(&args, &res)
	if err != nil {
		h.log.Infof("error listing datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WritePageResponse(w, res, r, args.Page())
}

// RunHandler is the endpoint for executing a query
func (h *QueryHandlers) RunHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
		h.runHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *QueryHandlers) runHandler(w http.ResponseWriter, r *http.Request) {
	ds := &dataset.Dataset{}
	if err := json.NewDecoder(r.Body).Decode(ds); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	format := r.FormValue("format")
	if format == "" {
		format = "csv"
	}
	df, err := dataset.ParseDataFormatString(format)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	p := &core.RunParams{
		SaveName: r.FormValue("name"),
		Dataset:  ds,
	}
	p.Format = df

	res := &repo.DatasetRef{}
	if err := h.Run(p, res); err != nil {
		h.log.Infof("error running query: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

// DatasetQueriesHandler is the endpoint for getting the queries that reference a dataset
func (h *QueryHandlers) DatasetQueriesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.datasetQueriesHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *QueryHandlers) datasetQueriesHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.DatasetQueriesParams{
		Path: r.URL.Path[len("/queries"):],
	}

	res := []*repo.DatasetRef{}
	if err := h.DatasetQueries(p, &res); err != nil {
		h.log.Info("error listing dataset queries: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}
