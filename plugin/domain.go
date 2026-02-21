package plugin

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// ResolvedDomainInfo holds resolved domain metadata from a DomainPlugin.
type ResolvedDomainInfo struct {
	DomainID    uuid.UUID `json:"domainId"`
	TypeCode    string    `json:"typeCode"`
	Key         string    `json:"key"`
	DisplayName string    `json:"displayName"`
}

// Subject represents the actor requesting domain access.
type Subject struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"`
}

// DomainPlugin is an optional capability for plugins that register a domain type.
type DomainPlugin interface {
	TypeCode() string
	ResolveDomain(ctx context.Context, r *http.Request) (*ResolvedDomainInfo, bool, error)
	ValidateMembership(ctx context.Context, domainID uuid.UUID, subject Subject) (bool, error)
}
