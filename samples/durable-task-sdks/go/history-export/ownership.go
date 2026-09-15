package main

import (
	"context"
	"errors"
	"strings"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/exporthistory"
)

// The stock export filter has no instance-ID predicate. Refuse a whole page
// containing an unowned ID before the SDK can schedule any history reads.
// An immutable allow-list also guards direct metadata/history activity requests.
type ownedHistorySource struct {
	inner   exporthistory.HistorySource
	allowed map[api.InstanceID]struct{}
}

var errUnownedHistory = errors.New("refusing to read/export an instance outside this run; use an isolated task hub")

func (s *ownedHistorySource) ListInstanceIDs(ctx context.Context, query api.InstanceIDQuery) (*api.InstanceIDQueryResult, error) {
	page, err := s.inner.ListInstanceIDs(ctx, query)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, errors.New("instance listing returned no page")
	}
	for _, id := range page.InstanceIDs {
		if _, owned := s.allowed[id]; !owned {
			return nil, errUnownedHistory
		}
	}
	return page, nil
}

func (s *ownedHistorySource) FetchOrchestrationMetadata(
	ctx context.Context, id api.InstanceID, options ...api.FetchOrchestrationMetadataOptions,
) (*api.OrchestrationMetadata, error) {
	if _, owned := s.allowed[id]; !owned {
		return nil, errUnownedHistory
	}
	return s.inner.FetchOrchestrationMetadata(ctx, id, options...)
}

func (s *ownedHistorySource) StreamOrchestrationHistory(
	ctx context.Context, id api.InstanceID, query api.HistoryQuery, handler api.HistoryEventHandler,
) error {
	if _, owned := s.allowed[id]; !owned {
		return errUnownedHistory
	}
	return s.inner.StreamOrchestrationHistory(ctx, id, query, handler)
}

type ownedHistoryStore struct {
	inner       exporthistory.Store
	allowed     map[api.InstanceID]struct{}
	container   string
	prefix      string
	beforeWrite func(context.Context) error
}

func (s *ownedHistoryStore) Write(ctx context.Context, object exporthistory.ExportObject) error {
	if _, owned := s.allowed[api.InstanceID(object.Metadata["instanceId"])]; !owned {
		return errUnownedHistory
	}
	if object.Container != s.container || !strings.HasPrefix(object.Name, s.prefix) {
		return errors.New("refusing a history write outside this run's container/prefix")
	}
	if s.beforeWrite != nil {
		if err := s.beforeWrite(ctx); err != nil {
			return err
		}
	}
	return s.inner.Write(ctx, object)
}
