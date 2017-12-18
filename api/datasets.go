package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/repo"
)

// DatasetHandlers wraps a requests struct to interface with http.HandlerFunc
type DatasetHandlers struct {
	core.DatasetRequests
	log  logging.Logger
	repo repo.Repo
}

// NewDatasetHandlers allocates a DatasetHandlers pointer
func NewDatasetHandlers(log logging.Logger, r repo.Repo) *DatasetHandlers {
	req := core.NewDatasetRequests(r, nil)
	h := DatasetHandlers{*req, log, r}
	return &h
}

// DatasetsHandler is a dataset list endpoint
func (h *DatasetHandlers) DatasetsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.listDatasetsHandler(w, r)
	case "PUT":
		h.updateDatasetHandler(w, r)
	case "POST":
		h.initDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// DatasetHandler is a dataset single endpoint
func (h *DatasetHandlers) DatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getDatasetHandler(w, r)
	case "PUT":
		h.updateDatasetHandler(w, r)
	case "DELETE":
		h.deleteDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// InitDatasetHandler is an endpoint for creating new datasets
func (h *DatasetHandlers) InitDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
		h.initDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// StructuredDataHandler is the data endpoint for a dataset
func (h *DatasetHandlers) StructuredDataHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getStructuredDataHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// AddDatasetHandler is the endpoint for adding an existing dataset to this repo
func (h *DatasetHandlers) AddDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
		h.addDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// RenameDatasetHandler is the endpoint for renaming datasets
func (h *DatasetHandlers) RenameDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST", "PUT":
		h.renameDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ZipDatasetHandler is the endpoint for getting a zip archive of a dataset
func (h *DatasetHandlers) ZipDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &repo.DatasetRef{}
	args := &core.GetDatasetParams{
		Path: datastore.NewKey(r.URL.Path[len("/download/"):]),
		Hash: r.FormValue("hash"),
	}
	err := h.Get(args, res)
	if err != nil {
		h.log.Infof("error getting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("filename=\"%s.zip\"", "dataset"))
	dsutil.WriteZipArchive(h.repo.Store(), res.Dataset, w)
}

func (h *DatasetHandlers) listDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	args := core.ListParamsFromRequest(r)
	args.OrderBy = "created"
	res := []*repo.DatasetRef{}
	if err := h.List(&args, &res); err != nil {
		h.log.Infof("error listing datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, args.Page()); err != nil {
		h.log.Infof("error list datasests response: %s", err.Error())
	}
}

func (h *DatasetHandlers) getDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &repo.DatasetRef{}
	args := &core.GetDatasetParams{
		Path: datastore.NewKey(r.URL.Path[len("/datasets/"):]),
		Hash: r.FormValue("hash"),
	}
	err := h.Get(args, res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *DatasetHandlers) initDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.InitDatasetParams{}
	switch r.Header.Get("Content-Type") {
	case "application/json":
		json.NewDecoder(r.Body).Decode(p)
	default:
		var f cafs.File
		infile, header, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		f = memfs.NewMemfileReader(header.Filename, infile)
		p = &core.InitDatasetParams{
			URL:          r.FormValue("url"),
			Name:         r.FormValue("name"),
			DataFilename: header.Filename,
			Data:         f,
		}
	}

	res := &repo.DatasetRef{}
	if err := h.InitDataset(p, res); err != nil {
		h.log.Infof("error initializing dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res.Dataset)
}

func (h *DatasetHandlers) updateDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("Content-Type") {
	case "application/json":
		h.updateMetadataHandler(w, r)
		// default:
		//  h.initDatasetFileHandler(w, r)
	}
}

func (h *DatasetHandlers) updateMetadataHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.UpdateParams{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res := &repo.DatasetRef{}
	if err := h.Update(p, res); err != nil {
		h.log.Infof("error updating dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *DatasetHandlers) deleteDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.DeleteParams{
		Name: r.FormValue("name"),
		Path: datastore.NewKey(r.URL.Path[len("/datasets"):]),
	}

	ref := &repo.DatasetRef{}
	if err := h.Get(&core.GetDatasetParams{Name: p.Name, Path: p.Path}, ref); err != nil {
		return
	}

	res := false
	if err := h.Delete(p, &res); err != nil {
		h.log.Infof("error deleting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, ref.Dataset)
}

func (h *DatasetHandlers) getStructuredDataHandler(w http.ResponseWriter, r *http.Request) {
	listParams := core.ListParamsFromRequest(r)
	all, err := util.ReqParamBool("all", r)
	if err != nil {
		all = false
	}

	objectRows, err := util.ReqParamBool("object_rows", r)
	if err != nil {
		objectRows = true
	}

	p := &core.StructuredDataParams{
		Format: dataset.JSONDataFormat,
		FormatConfig: &dataset.JSONOptions{
			ArrayEntries: !objectRows,
		},
		Path:   datastore.NewKey(r.URL.Path[len("/data"):]),
		Limit:  listParams.Limit,
		Offset: listParams.Offset,
		All:    all,
	}
	data := &core.StructuredData{}
	if err := h.StructuredData(p, data); err != nil {
		h.log.Infof("error reading structured data: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, data)
}

func (h *DatasetHandlers) addDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.AddParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		// TODO - clean this up
		p.Hash = r.URL.Path[len("/add/"):]
		if p.Name == "" && r.FormValue("name") != "" {
			p.Name = r.FormValue("name")
		}
	} else {
		p = &core.AddParams{
			Name: r.URL.Query().Get("name"),
			Hash: r.URL.Path[len("/add/"):],
		}
	}

	res := &repo.DatasetRef{}
	if err := h.AddDataset(p, res); err != nil {
		h.log.Infof("error adding dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h DatasetHandlers) renameDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.RenameParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	} else {
		p = &core.RenameParams{
			Current: r.URL.Query().Get("current"),
			New:     r.URL.Query().Get("new"),
		}
	}

	res := &repo.DatasetRef{}
	if err := h.Rename(p, res); err != nil {
		h.log.Infof("error renaming dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}
