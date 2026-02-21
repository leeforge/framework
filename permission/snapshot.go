package permission

import (
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Scope defines permission scope.
type Scope string

const (
	ScopeAPI  Scope = "api"
	ScopeUI   Scope = "ui"
	ScopeData Scope = "data"
)

// Status defines permission status.
type Status string

const (
	StatusActive     Status = "active"
	StatusDeprecated Status = "deprecated"
)

// Permission represents a permission code definition.
type Permission struct {
	Code        string
	Name        string
	Description string
	Scope       Scope
	Status      Status
}

// RouteInfo represents API route metadata.
type RouteInfo struct {
	Method      string
	Path        string
	Description string
	IsPublic    bool
	Permissions []string
}

// Mapping represents API to permission code mapping.
type Mapping struct {
	Method         string
	Path           string
	PermissionCode string
}

// Snapshot contains all permissions, routes, and mappings for syncing.
type Snapshot struct {
	Permissions []Permission
	Routes      []RouteInfo
	Mappings    []Mapping
}

// SnapshotFromRouter walks a chi router and builds a permission snapshot.
func SnapshotFromRouter(r chi.Routes) (Snapshot, error) {
	var routes []RouteInfo
	if err := chi.Walk(r, func(method string, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		meta, ok := ExtractMeta(handler)
		if !ok {
			meta = Meta{}
		}
		routes = append(routes, RouteInfo{
			Method:      strings.ToUpper(method),
			Path:        route,
			Description: meta.Description,
			IsPublic:    meta.IsPublic,
			Permissions: append([]string(nil), meta.Permissions...),
		})
		return nil
	}); err != nil {
		return Snapshot{}, err
	}

	return BuildSnapshot(routes), nil
}

// BuildSnapshot builds a deduplicated snapshot from routes.
func BuildSnapshot(routes []RouteInfo) Snapshot {
	permissionMap := make(map[string]Permission)
	routeMap := make(map[string]RouteInfo)
	mappingMap := make(map[string]Mapping)

	for _, route := range routes {
		if route.Method == "" || route.Path == "" {
			continue
		}
		normalizedPerms := uniqueNormalized(route.Permissions)
		route.Permissions = normalizedPerms

		routeKey := routeKey(route.Method, route.Path)
		if _, exists := routeMap[routeKey]; !exists {
			routeMap[routeKey] = route
		}

		for _, code := range normalizedPerms {
			if code == "" {
				continue
			}
			if _, exists := permissionMap[code]; !exists {
				permissionMap[code] = Permission{
					Code:   code,
					Name:   code,
					Scope:  ScopeAPI,
					Status: StatusActive,
				}
			}

			mappingKey := mappingKey(route.Method, route.Path, code)
			if _, exists := mappingMap[mappingKey]; !exists {
				mappingMap[mappingKey] = Mapping{
					Method:         route.Method,
					Path:           route.Path,
					PermissionCode: code,
				}
			}
		}
	}

	return Snapshot{
		Permissions: permissionsFromMap(permissionMap),
		Routes:      routesFromMap(routeMap),
		Mappings:    mappingsFromMap(mappingMap),
	}
}

func routeKey(method, path string) string {
	return method + ":" + path
}

func mappingKey(method, path, code string) string {
	return method + ":" + path + ":" + code
}

func normalizeCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func uniqueNormalized(codes []string) []string {
	if len(codes) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(codes))
	var result []string
	for _, code := range codes {
		normalized := normalizeCode(code)
		if normalized == "" {
			continue
		}
		if _, exists := set[normalized]; exists {
			continue
		}
		set[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func permissionsFromMap(m map[string]Permission) []Permission {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for code := range m {
		keys = append(keys, code)
	}
	sort.Strings(keys)
	result := make([]Permission, 0, len(keys))
	for _, code := range keys {
		result = append(result, m[code])
	}
	return result
}

func routesFromMap(m map[string]RouteInfo) []RouteInfo {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]RouteInfo, 0, len(keys))
	for _, key := range keys {
		result = append(result, m[key])
	}
	return result
}

func mappingsFromMap(m map[string]Mapping) []Mapping {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]Mapping, 0, len(keys))
	for _, key := range keys {
		result = append(result, m[key])
	}
	return result
}
