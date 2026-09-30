package openapi

import (
	"strings"

	"github.com/apirout/apirout/internal/console/executor"
	"github.com/apirout/apirout/internal/console/registry"
)

// Service generates OpenAPI specifications
type Service struct {
	registry *registry.Registry
	executor *executor.Service
}

// NewService creates a new OpenAPI service
func NewService(reg *registry.Registry, exec *executor.Service) *Service {
	return &Service{
		registry: reg,
		executor: exec,
	}
}

// GenerateSpec generates an OpenAPI specification
func (s *Service) GenerateSpec(groupIDs []string, endpointIDs []string) map[string]interface{} {
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "Pusdatin Apirout",
			"description": "Auto-generated API from database queries",
			"version":     "1.0.0",
		},
		"paths": make(map[string]interface{}),
	}

	endpoints, err := s.registry.ListEndpoints()
	if err != nil {
		return spec
	}

	paths := spec["paths"].(map[string]interface{})

	for _, endpoint := range endpoints {
		if !endpoint.Enabled {
			continue
		}

		// Filter by group if specified
		if groupIDs != nil && endpoint.GroupID != nil {
			found := false
			for _, gid := range groupIDs {
				if gid == *endpoint.GroupID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by endpoint IDs if specified
		if endpointIDs != nil {
			found := false
			for _, eid := range endpointIDs {
				if eid == endpoint.ID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		methodLower := strings.ToLower(endpoint.Method)
		if methodLower == "" {
			methodLower = "get"
		}

		pathSpec := map[string]interface{}{
			"summary":     "Execute " + endpoint.Path,
			"operationId": endpoint.ID,
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Successful response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"data": map[string]interface{}{
										"type": "array",
									},
								},
							},
						},
					},
				},
				"401": map[string]interface{}{
					"description": "Unauthorized",
				},
				"500": map[string]interface{}{
					"description": "Internal server error",
				},
			},
		}

		// Always add pagination params
		parameters := []map[string]interface{}{
			{
				"name":        "page",
				"in":          "query",
				"description": "Page number for pagination",
				"schema": map[string]interface{}{
					"type": "integer",
				},
			},
			{
				"name":        "limit",
				"in":          "query",
				"description": "Number of items per page",
				"schema": map[string]interface{}{
					"type": "integer",
				},
			},
		}

		if endpoint.AllowDynamicFilters {
			// Fetch columns
			columns, _ := s.executor.GetQueryColumns(endpoint.ID)
			
			// Show up to 4 columns as examples
			count := 0
			for _, col := range columns {
				if count >= 4 {
					break
				}
				parameters = append(parameters, map[string]interface{}{
					"name":        col,
					"in":          "query",
					"description": "Dynamic filter for " + col,
					"schema": map[string]interface{}{
						"type": "string",
					},
					"required": false,
				})
				count++
			}
		}
		
		pathSpec["parameters"] = parameters

		if !endpoint.IsPublic {
			pathSpec["security"] = []map[string]interface{}{
				{"bearerAuth": []string{}},
			}
		}

		if _, exists := paths[endpoint.Path]; !exists {
			paths[endpoint.Path] = make(map[string]interface{})
		}
		paths[endpoint.Path].(map[string]interface{})[methodLower] = pathSpec
	}

	// Add security scheme
	spec["components"] = map[string]interface{}{
		"securitySchemes": map[string]interface{}{
			"bearerAuth": map[string]interface{}{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
			},
		},
	}

	return spec
}
