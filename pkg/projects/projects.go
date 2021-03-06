package projects

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/curious-kitten/scratch-post/internal/decoder"
	"github.com/curious-kitten/scratch-post/pkg/metadata"
)

//go:generate mockgen -source ./projects.go -destination mocks/projects.go

// Project represents a umbrella for tests that refer to the same product
type Project struct {
	Identity    *metadata.Identity `json:"identity"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
}

// AddIdentity sets the identity of the project
func (p *Project) AddIdentity(identity *metadata.Identity) {
	p.Identity = identity
}

// GetIdentity retruns the identity of the project
func (p *Project) GetIdentity() *metadata.Identity {
	return p.Identity
}

// Validate check whether the constraints on Project have been met
func (p *Project) Validate() error {
	if p.Name == "" {
		return metadata.NewValidationError("name is a mandatory parameter")
	}
	return nil
}

type Adder interface {
	AddOne(ctx context.Context, item interface{}) error
}

// Getter is used to retrieve items from the store
type Getter interface {
	Get(ctx context.Context, id string, item interface{}) error
	GetAll(ctx context.Context, items interface{}) error
}

// Deleter deletes an entry from the collection
type Deleter interface {
	Delete(ctx context.Context, id string) error
}

type Updater interface {
	Update(ctx context.Context, id string, item interface{}) error
}

type ReaderUpdater interface {
	Getter
	Updater
}

type IdentityGenerator interface {
	AddMeta(author string, objType string, identifiable metadata.Identifiable) error
}

// New creates a new project
func New(ig IdentityGenerator, store Adder) func(ctx context.Context, author string, data io.Reader) (interface{}, error) {
	return func(ctx context.Context, author string, data io.Reader) (interface{}, error) {
		project := &Project{}
		if err := decoder.Decode(project, data); err != nil {
			return nil, err
		}
		if err := ig.AddMeta(author, "project", project); err != nil {
			return nil, err
		}
		if err := store.AddOne(ctx, project); err != nil {
			return nil, err
		}
		return project, nil
	}
}

// List returns a function used to return the projects
func List(collection Getter) func(ctx context.Context) ([]interface{}, error) {
	return func(ctx context.Context) ([]interface{}, error) {
		projects := []Project{}
		err := collection.GetAll(ctx, &projects)
		if err != nil {
			return nil, err
		}
		items := make([]interface{}, len(projects))
		for i, v := range projects {
			items[i] = v
		}
		return items, nil
	}
}

// Get returns a scenario based on the passed ID
func Get(collectiom Getter) func(ctx context.Context, id string) (interface{}, error) {
	return func(ctx context.Context, id string) (interface{}, error) {
		project := &Project{}
		if err := collectiom.Get(ctx, id, project); err != nil {
			return nil, err
		}
		return project, nil
	}
}

// Delete returns a function to delete a scenario based on the passed ID
func Delete(collection Deleter) func(ctx context.Context, id string) error {
	return func(ctx context.Context, id string) error {
		if err := collection.Delete(ctx, id); err != nil {
			return err
		}
		return nil
	}
}

// Update is used to replace a scenario with the provided scenario
func Update(collection ReaderUpdater) func(ctx context.Context, user string, id string, data io.Reader) (interface{}, error) {
	return func(ctx context.Context, user string, id string, data io.Reader) (interface{}, error) {
		project := &Project{}
		if err := decoder.Decode(project, data); err != nil {
			return nil, err
		}
		foundProject, err := Get(collection)(ctx, id)
		if err != nil {
			return nil, err
		}
		var s *Project
		var ok bool
		if s, ok = foundProject.(*Project); !ok {
			return nil, fmt.Errorf("invalid data structure in DB")
		}
		project.Identity = s.Identity
		project.Identity.UpdateTime = time.Now()
		project.Identity.UpdatedBy = user
		if err := collection.Update(ctx, id, project); err != nil {
			return nil, err
		}
		return project, nil
	}
}
