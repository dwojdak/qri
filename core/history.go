package core

import (
	"fmt"
	"net/rpc"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

// HistoryRequests encapsulates business logic for the log
// of changes to datasets, think "git log"
type HistoryRequests struct {
	repo repo.Repo
	cli  *rpc.Client
}

// CoreRequestsName implements the Requets interface
func (d HistoryRequests) CoreRequestsName() string { return "history" }

// NewHistoryRequests creates a HistoryRequests pointer from either a repo
// or an rpc.Client
func NewHistoryRequests(r repo.Repo, cli *rpc.Client) *HistoryRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewHistoryRequests"))
	}
	return &HistoryRequests{
		repo: r,
	}
}

// LogParams defines parameters for the Log method
type LogParams struct {
	ListParams
	// Path to the dataset to fetch history for
	Path datastore.Key
}

// Log returns the history of changes for a given dataset
func (d *HistoryRequests) Log(params *LogParams, res *[]*repo.DatasetRef) (err error) {
	if d.cli != nil {
		return d.cli.Call("HistoryRequests.Log", params, res)
	}

	log := []*repo.DatasetRef{}
	limit := params.Limit
	ref := &repo.DatasetRef{Path: params.Path}

	if params.Path.String() == "" {
		return fmt.Errorf("path is required")
	}

	for {
		ref.Dataset, err = dsfs.LoadDataset(d.repo.Store(), ref.Path)
		if err != nil {
			return err
		}
		log = append(log, ref)

		limit--
		if limit == 0 || ref.Dataset.Previous.String() == "" {
			break
		}
		// TODO - clean this up
		_, cleaned := dsfs.RefType(ref.Dataset.Previous.String())
		ref = &repo.DatasetRef{Path: datastore.NewKey(cleaned)}
	}

	*res = log
	return nil
}
