package plugin

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

type testDomainPlugin struct{}

func (testDomainPlugin) TypeCode() string { return "tenant" }

func (testDomainPlugin) ResolveDomain(_ context.Context, _ *http.Request) (*ResolvedDomainInfo, bool, error) {
	return &ResolvedDomainInfo{DomainID: uuid.New(), TypeCode: "tenant", Key: "acme"}, true, nil
}

func (testDomainPlugin) ValidateMembership(_ context.Context, _ uuid.UUID, _ Subject) (bool, error) {
	return true, nil
}

func TestDomainPluginContractCompiles(t *testing.T) {
	var _ DomainPlugin = testDomainPlugin{}
}
