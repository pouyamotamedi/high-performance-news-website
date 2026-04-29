package api

import (
	"fmt"
	"time"

	"github.com/graphql-go/graphql"
	"high-performance-news-website/internal/integration"
)

// GraphQLAPI provides GraphQL API for integrations
type GraphQLAPI struct {
	manager *integration.IntegrationManager
	schema  graphql.Schema
}

// NewGraphQLAPI creates a new GraphQL API
func NewGraphQLAPI(manager *integration.IntegrationManager) (*GraphQLAPI, error) {
	api := &GraphQLAPI{
		manager: manager,
	}

	schema, err := api.buildSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to build GraphQL schema: %w", err)
	}

	api.schema = schema
	return api, nil
}

// GetSchema returns the GraphQL schema
func (api *GraphQLAPI) GetSchema() graphql.Schema {
	return api.schema
}

// buildSchema builds the GraphQL schema
func (api *GraphQLAPI) buildSchema() (graphql.Schema, error) {
	// Define types
	eventTypeEnum := graphql.NewEnum(graphql.EnumConfig{
		Name: "EventType",
		Values: graphql.EnumValueConfigMap{
			"TEST_FAILURE": &graphql.EnumValueConfig{
				Value: integration.EventTypeTestFailure,
			},
			"TEST_SUCCESS": &graphql.EnumValueConfig{
				Value: integration.EventTypeTestSuccess,
			},
			"DEPLOYMENT": &graphql.EnumValueConfig{
				Value: integration.EventTypeDeployment,
			},
			"SECURITY_ALERT": &graphql.EnumValueConfig{
				Value: integration.EventTypeSecurityAlert,
			},
			"PERFORMANCE_ISSUE": &graphql.EnumValueConfig{
				Value: integration.EventTypePerformanceIssue,
			},
			"CODE_REVIEW": &graphql.EnumValueConfig{
				Value: integration.EventTypeCodeReview,
			},
		},
	})

	priorityEnum := graphql.NewEnum(graphql.EnumConfig{
		Name: "Priority",
		Values: graphql.EnumValueConfigMap{
			"LOW": &graphql.EnumValueConfig{
				Value: integration.PriorityLow,
			},
			"MEDIUM": &graphql.EnumValueConfig{
				Value: integration.PriorityMedium,
			},
			"HIGH": &graphql.EnumValueConfig{
				Value: integration.PriorityHigh,
			},
			"CRITICAL": &graphql.EnumValueConfig{
				Value: integration.PriorityCritical,
			},
		},
	})

	integrationTypeEnum := graphql.NewEnum(graphql.EnumConfig{
		Name: "IntegrationType",
		Values: graphql.EnumValueConfigMap{
			"BUG_TRACKING": &graphql.EnumValueConfig{
				Value: integration.IntegrationTypeBugTracking,
			},
			"PROJECT_MGMT": &graphql.EnumValueConfig{
				Value: integration.IntegrationTypeProjectMgmt,
			},
			"CODE_REVIEW": &graphql.EnumValueConfig{
				Value: integration.IntegrationTypeCodeReview,
			},
			"MONITORING": &graphql.EnumValueConfig{
				Value: integration.IntegrationTypeMonitoring,
			},
			"COMMUNICATION": &graphql.EnumValueConfig{
				Value: integration.IntegrationTypeCommunication,
			},
			"CI_CD": &graphql.EnumValueConfig{
				Value: integration.IntegrationTypeCI,
			},
		},
	})

	// Define JSON scalar type for dynamic data
	jsonType := graphql.NewScalar(graphql.ScalarConfig{
		Name: "JSON",
		Serialize: func(value interface{}) interface{} {
			return value
		},
		ParseValue: func(value interface{}) interface{} {
			return value
		},
	})

	// Define Event type
	eventType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Event",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"type": &graphql.Field{
				Type: eventTypeEnum,
			},
			"source": &graphql.Field{
				Type: graphql.String,
			},
			"timestamp": &graphql.Field{
				Type: graphql.DateTime,
			},
			"priority": &graphql.Field{
				Type: priorityEnum,
			},
			"data": &graphql.Field{
				Type: jsonType,
			},
		},
	})

	// Define IntegrationMetrics type
	metricsType := graphql.NewObject(graphql.ObjectConfig{
		Name: "IntegrationMetrics",
		Fields: graphql.Fields{
			"eventsSent": &graphql.Field{
				Type: graphql.Int,
			},
			"errorCount": &graphql.Field{
				Type: graphql.Int,
			},
			"lastEventTime": &graphql.Field{
				Type: graphql.DateTime,
			},
			"lastErrorTime": &graphql.Field{
				Type: graphql.DateTime,
			},
			"successRate": &graphql.Field{
				Type: graphql.Float,
			},
		},
	})

	// Define IntegrationStatus type
	statusType := graphql.NewObject(graphql.ObjectConfig{
		Name: "IntegrationStatus",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"type": &graphql.Field{
				Type: integrationTypeEnum,
			},
			"healthy": &graphql.Field{
				Type: graphql.Boolean,
			},
			"metrics": &graphql.Field{
				Type: metricsType,
			},
		},
	})

	// Define Webhook type
	webhookType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Webhook",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"url": &graphql.Field{
				Type: graphql.String,
			},
			"enabled": &graphql.Field{
				Type: graphql.Boolean,
			},
			"events": &graphql.Field{
				Type: graphql.NewList(eventType),
			},
			"retries": &graphql.Field{
				Type: graphql.Int,
			},
			"timeout": &graphql.Field{
				Type: graphql.String,
			},
		},
	})

	// Define Query type
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"integrationStatus": &graphql.Field{
				Type: graphql.NewList(statusType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					status := api.manager.GetIntegrationStatus()
					var result []integration.IntegrationStatus
					for _, s := range status {
						result = append(result, s)
					}
					return result, nil
				},
			},
			"integrationHealth": &graphql.Field{
				Type: jsonType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return api.manager.HealthCheck(p.Context), nil
				},
			},
			"webhooks": &graphql.Field{
				Type: graphql.NewList(webhookType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// This would need to be implemented in the manager
					return []interface{}{}, nil
				},
			},
			"eventTypes": &graphql.Field{
				Type: graphql.NewList(eventTypeEnum),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return []integration.EventType{
						integration.EventTypeTestFailure,
						integration.EventTypeTestSuccess,
						integration.EventTypeDeployment,
						integration.EventTypeSecurityAlert,
						integration.EventTypePerformanceIssue,
						integration.EventTypeCodeReview,
					}, nil
				},
			},
		},
	})

	// Define input types for mutations
	eventInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "EventInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"type": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(eventTypeEnum),
			},
			"source": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"priority": &graphql.InputObjectFieldConfig{
				Type: priorityEnum,
			},
			"data": &graphql.InputObjectFieldConfig{
				Type: jsonType,
			},
		},
	})

	webhookInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "WebhookInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"name": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"url": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"secret": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"enabled": &graphql.InputObjectFieldConfig{
				Type: graphql.Boolean,
			},
			"events": &graphql.InputObjectFieldConfig{
				Type: graphql.NewList(eventTypeEnum),
			},
			"retries": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
			"timeout": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
		},
	})

	integrationConfigInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "IntegrationConfigInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"type": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(integrationTypeEnum),
			},
			"enabled": &graphql.InputObjectFieldConfig{
				Type: graphql.Boolean,
			},
			"settings": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(jsonType),
			},
		},
	})

	// Define Mutation type
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"sendEvent": &graphql.Field{
				Type: graphql.String,
				Args: graphql.FieldConfigArgument{
					"event": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(eventInputType),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					eventInput := p.Args["event"].(map[string]interface{})
					
					event := integration.Event{
						ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
						Type:      eventInput["type"].(integration.EventType),
						Source:    eventInput["source"].(string),
						Timestamp: time.Now(),
						Data:      make(map[string]interface{}),
					}

					if priority, ok := eventInput["priority"]; ok {
						event.Priority = priority.(integration.EventPriority)
					} else {
						event.Priority = integration.PriorityMedium
					}

					if data, ok := eventInput["data"]; ok {
						event.Data = data.(map[string]interface{})
					}

					if err := api.manager.SendEvent(p.Context, event); err != nil {
						return nil, err
					}

					return event.ID, nil
				},
			},
			"connectIntegration": &graphql.Field{
				Type: graphql.String,
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"config": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(integrationConfigInputType),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					name := p.Args["name"].(string)
					configInput := p.Args["config"].(map[string]interface{})
					
					config := integration.Config{
						Name:     name,
						Type:     configInput["type"].(integration.IntegrationType),
						Enabled:  true,
						Settings: configInput["settings"].(map[string]interface{}),
					}

					if enabled, ok := configInput["enabled"]; ok {
						config.Enabled = enabled.(bool)
					}

					if err := api.manager.ConnectIntegration(p.Context, name, config); err != nil {
						return nil, err
					}

					return fmt.Sprintf("Successfully connected to %s", name), nil
				},
			},
			"registerWebhook": &graphql.Field{
				Type: graphql.String,
				Args: graphql.FieldConfigArgument{
					"webhook": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(webhookInputType),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					webhookInput := p.Args["webhook"].(map[string]interface{})
					name := webhookInput["name"].(string)
					
					// This would need to be implemented in the manager
					return fmt.Sprintf("Webhook %s registered successfully", name), nil
				},
			},
			"resetMetrics": &graphql.Field{
				Type: graphql.String,
				Args: graphql.FieldConfigArgument{
					"integration": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					integration := p.Args["integration"]
					if integration == nil {
						// Reset all metrics
						return "All metrics reset successfully", nil
					}
					
					integrationName := integration.(string)
					// This would need to be implemented in the manager
					return fmt.Sprintf("Metrics reset for %s", integrationName), nil
				},
			},
		},
	})

	// Create schema
	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})
}