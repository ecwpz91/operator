package meshconfigs

// CatalogService is the schema for a Catalog API service entry
// It is added here since it isn't defined in greymatter-cue
#CatalogService: {
	mesh_id:                   mesh.metadata.name
	service_id:                workload.metadata.name
	name:                      *workload.metadata.name | string
	version?:                  string
	api_endpoint?:             string
	api_spec_endpoint?:        string
	description?:              string
	business_impact?:          string
	enable_instance_metrics:   true
	enable_historical_metrics: *false | bool
}
