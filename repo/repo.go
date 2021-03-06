// Package repo represents a repository of qri information
// Analogous to a git repository, repo expects a rigid structure
// filled with rich types specific to qri.
// Lots of things in here take inspiration from the ipfs datastore interface:
// github.com/ipfs/go-datastore
package repo

import (
	"fmt"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo/profile"
)

var (
	// ErrNotFound is the err implementers should return when stuff isn't found
	ErrNotFound = fmt.Errorf("repo: not found")
	// ErrNameRequired is for when a name is missing-but-expected
	ErrNameRequired = fmt.Errorf("repo: name is required")
	// ErrNameTaken is for when a Namestore name is already taken
	ErrNameTaken = fmt.Errorf("repo: name already in use")
	// ErrRepoEmpty is for when the repo has no datasets
	ErrRepoEmpty = fmt.Errorf("repo: this repo contains no datasets")
)

// Repo is the interface for working with a qri repository
// conceptually, it's a more-specific version of a datastore.
type Repo interface {
	// All repositories wrapp a content-addressed filestore as the cannonical
	// record of this repository's data. Store gives direct access to the
	// cafs.Filestore instance any given repo is using.
	Store() cafs.Filestore
	// Graph returns a graph of this repositories data resources
	Graph() (map[string]*dsgraph.Node, error)
	// At the heart of all repositories is a namestore, which maps user-defined
	// aliases for datasets to their underlying content-addressed hash
	// as an example:
	// 		my_dataset : /ipfs/Qmeiuzejjs....
	// these aliases are then used in qri SQL statements. Names are *not*
	// universally unique, but must be unique within the namestore
	Namestore
	// Repos also serve as a store of dataset information.
	// It's important that this store maintain sync with any underlying filestore.
	// (which is why we might want to kill this in favor of just having a cache?)
	// The behaviour of the embedded DatasetStore will typically differ from the cache,
	// by only returning saved/pinned/permanent datasets.
	Datasets
	// QueryLog keeps a log of queries that have been run
	QueryLog
	// ChangeRequets gives this repo's change request store
	ChangeRequestStore
	// A repository must maintain profile information about the owner of this dataset.
	// The value returned by Profile() should represent the user.
	Profile() (*profile.Profile, error)
	// It must be possible to alter profile information.
	SaveProfile(*profile.Profile) error
	// A repository must maintain profile information about encountered peers.
	// Decsisions regarding retentaion of peers is left to the the implementation
	// TODO - should rename this to "profiles" to separate from the networking
	// concept of a peer
	Peers() Peers
	// Cache keeps an ephemeral store of dataset information
	// that may be purged at any moment. Results of searching for datasets,
	// dataset references other users have, etc, should all be stored here.
	Cache() Datasets
	// All repositories provide their own analytics information.
	// Our analytics implementation is under super-active development.
	Analytics() analytics.Analytics
}

// Namestore is an in-progress solution for aliasing
// datasets locally, it's an interface for storing & retrieving
// datasets by local names
type Namestore interface {
	PutName(name string, path datastore.Key) error
	GetPath(name string) (datastore.Key, error)
	GetName(path datastore.Key) (string, error)
	DeleteName(name string) error
	Namespace(limit, offset int) ([]*DatasetRef, error)
	NameCount() (int, error)
}

// Datasets is the minimum interface to act as a store of datasets.
// It's intended to look a *lot* like the ipfs datastore interface, but
// scoped only to datasets to make for easier consumption.
// Datasets stored here should be reasonably dereferenced to avoid
// additional lookups.
// All fields here work only with paths (which are datastore.Key's)
// to dereference a name, you'll need a Namestore interface
// oh golang, can we haz generics plz?
type Datasets interface {
	// Put a dataset in the store
	PutDataset(path datastore.Key, ds *dataset.Dataset) error
	// Put multiple datasets in the store
	PutDatasets([]*DatasetRef) error
	// Get a dataset from the store
	GetDataset(path datastore.Key) (*dataset.Dataset, error)
	// Remove a dataset from the store
	DeleteDataset(path datastore.Key) error
	// Query is extracted from the ipfs datastore interface:
	Query(query.Query) (query.Results, error)
}

// QueryLogItem is a list of details for logging a query
type QueryLogItem struct {
	Query       string
	Name        string
	Key         datastore.Key
	DatasetPath datastore.Key
	Time        time.Time
}

// QueryLog keeps logs
type QueryLog interface {
	LogQuery(*QueryLogItem) error
	ListQueryLogs(limit, offset int) ([]*QueryLogItem, error)
	QueryLogItem(q *QueryLogItem) (*QueryLogItem, error)
}

// SearchParams encapsulates parameters provided to Searchable.Search
type SearchParams struct {
	Q             string
	Limit, Offset int
}

// Searchable is an opt-in interface for supporting repository search
type Searchable interface {
	Search(p SearchParams) ([]*DatasetRef, error)
}

// DatasetsQuery is a convenience function to read all query results & parse into a
// map[string]*dataset.Dataset.
func DatasetsQuery(dss Datasets, q query.Query) (map[string]*dataset.Dataset, error) {
	ds := map[string]*dataset.Dataset{}
	results, err := dss.Query(q)
	if err != nil {
		return nil, err
	}

	for res := range results.Next() {
		d, ok := res.Value.(*dataset.Dataset)
		if !ok {
			return nil, fmt.Errorf("query returned the wrong type, expected a profile pointer")
		}
		ds[res.Key] = d
	}

	return ds, nil
}
