package postman

import (
	"github.com/apirout/apirout/internal/console/executor"
	"github.com/apirout/apirout/internal/console/registry"
)

// Service generates Postman collections
type Service struct {
	registry *registry.Registry
	executor *executor.Service
}

// NewService creates a new Postman service
func NewService(reg *registry.Registry, exec *executor.Service) *Service {
	return &Service{
		registry: reg,
		executor: exec,
	}
}

// GenerateCollection generates a Postman collection
func (s *Service) GenerateCollection(groupIDs []string, endpointIDs []string, baseURL string) map[string]interface{} {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	collection := map[string]interface{}{
		"info": map[string]interface{}{
			"name":        "Pusdatin Apirout",
			"description": "Auto-generated Postman collection",
			"schema":      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		"item": []interface{}{},
		"auth": map[string]interface{}{
			"type": "bearer",
			"bearer": []map[string]interface{}{
				{
					"key":   "token",
					"value": "{{api_token}}",
					"type":  "string",
				},
			},
		},
		"variable": []map[string]interface{}{
			{
				"key":   "base_url",
				"value": baseURL,
			},
			{
				"key":   "api_token",
				"value": "",
			},
		},
	}

	endpoints, err := s.registry.ListEndpoints()
	if err != nil {
		return collection
	}

	items := []interface{}{}

	// Add auth folder
	authFolder := map[string]interface{}{
		"name": "Authentication",
		"item": []interface{}{
			map[string]interface{}{
				"name": "API Client Login",
				"request": map[string]interface{}{
					"method": "POST",
					"header": []map[string]interface{}{
						{"key": "Content-Type", "value": "application/json"},
					},
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  `{"username": "client1", "password": "secret"}`,
					},
					"url": map[string]interface{}{
						"raw":  "{{base_url}}/api/login",
						"host": []string{"{{base_url}}"},
						"path": []string{"api", "login"},
					},
				},
			},
		},
	}
	items = append(items, authFolder)

	// Add endpoints folder
	endpointsFolder := map[string]interface{}{
		"name": "Endpoints",
		"item": []interface{}{},
	}

	endpointItems := []interface{}{}
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

		item := map[string]interface{}{
			"name": endpoint.Path,
			"request": map[string]interface{}{
				"method": endpoint.Method,
				"url": map[string]interface{}{
					"raw":  "{{base_url}}" + endpoint.Path,
					"host": []string{"{{base_url}}"},
					"path": pathToArray(endpoint.Path),
				},
			},
		}

		// Always add pagination query params (disabled by default)
		queryParams := []map[string]interface{}{
			{
				"key":         "page",
				"value":       "1",
				"description": "Page number",
				"disabled":    true,
			},
			{
				"key":         "limit",
				"value":       "10",
				"description": "Items per page",
				"disabled":    true,
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
				queryParams = append(queryParams, map[string]interface{}{
					"key":         col,
					"value":       "value",
					"description": "Dynamic filter for " + col,
					"disabled":    true,
				})
				count++
			}
		}
		
		item["request"].(map[string]interface{})["url"].(map[string]interface{})["query"] = queryParams

		if !endpoint.IsPublic {
			item["request"].(map[string]interface{})["auth"] = map[string]interface{}{
				"type": "bearer",
				"bearer": []map[string]interface{}{
					{
						"key":   "token",
						"value": "{{api_token}}",
						"type":  "string",
					},
				},
			}
		}

		endpointItems = append(endpointItems, item)
	}

	endpointsFolder["item"] = endpointItems
	items = append(items, endpointsFolder)

	collection["item"] = items
	return collection
}

func pathToArray(path string) []string {
	// Remove leading slash and split
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	result := []string{}
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
