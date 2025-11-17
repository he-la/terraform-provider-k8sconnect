package auth

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GetClusterSchemaForResource returns the complete cluster schema attribute for resources
// with write-only behavior (not stored in state).
//
// The cluster configuration is:
// - Required in Terraform configuration (.tf files)
// - Available during all operations (Create/Update/Delete/Read)
// - Stored in private state (not visible to users)
// - NOT stored in public state (to avoid persisting credentials)
//
// Usage: Resources must call SaveClusterToPrivateState() before saving state,
// and LoadClusterFromPrivateState() during Read operations.
func GetClusterSchemaForResource() resourceschema.SingleNestedAttribute {
	return resourceschema.SingleNestedAttribute{
		Required: true,
		Description: "Kubernetes cluster connection for this specific resource. Can be different per-resource, enabling multi-cluster " +
			"deployments without provider aliases. Supports inline credentials (token, exec, client certs) or kubeconfig. " +
			"**NOTE: Connection details are write-only and not stored in state to protect credentials. " +
			"You must always provide the cluster configuration in your .tf files.**",
		Attributes: GetConnectionSchemaForResource(),
		PlanModifiers: []planmodifier.Object{
			WriteOnly(),
		},
	}
}

// GetConnectionSchemaForResource returns the cluster connection schema attributes for resources.
// This is the single source of truth for the connection schema.
func GetConnectionSchemaForResource() map[string]resourceschema.Attribute {
	return map[string]resourceschema.Attribute{
		"host": resourceschema.StringAttribute{
			Optional:    true,
			Description: "The hostname (in form of URI) of the Kubernetes API server.",
			Validators: []validator.String{
				urlValidator{},
			},
		},
		"cluster_ca_certificate": resourceschema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "Root certificate bundle for TLS authentication. Accepts PEM format or base64-encoded PEM - automatically detected.",
		},
		"kubeconfig": resourceschema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "Raw kubeconfig file content.",
			Validators: []validator.String{
				kubeconfigValidator{},
			},
		},
		"context": resourceschema.StringAttribute{
			Optional:    true,
			Description: "Context to use from the kubeconfig. Optional when kubeconfig contains exactly one context (that context will be used automatically). Required when kubeconfig contains multiple contexts to prevent accidental connection to the wrong cluster. Error will list available contexts if not specified when required.",
		},
		"token": resourceschema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "Token to authenticate to the Kubernetes API server.",
		},
		"client_certificate": resourceschema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "Client certificate for TLS authentication. Accepts PEM format or base64-encoded PEM - automatically detected.",
		},
		"client_key": resourceschema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "Client certificate key for TLS authentication. Accepts PEM format or base64-encoded PEM - automatically detected.",
		},
		"insecure": resourceschema.BoolAttribute{
			Optional:    true,
			Description: "Whether server should be accessed without verifying the TLS certificate.",
		},
		"proxy_url": resourceschema.StringAttribute{
			Optional:    true,
			Description: "URL of the proxy to use for requests.",
			Validators: []validator.String{
				urlValidator{},
			},
		},
		"exec": resourceschema.SingleNestedAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "Configuration for exec-based authentication.",
			Attributes: map[string]resourceschema.Attribute{
				"api_version": resourceschema.StringAttribute{
					Required:    true,
					Description: "API version to use when encoding the ExecCredentials resource.",
				},
				"command": resourceschema.StringAttribute{
					Required:    true,
					Description: "Command to execute.",
				},
				"args": resourceschema.ListAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "Arguments to pass when executing the plugin.",
				},
				"env": resourceschema.MapAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "Environment variables to set when executing the plugin.",
				},
			},
		},
	}
}

// GetClusterSchemaForDataSource returns the complete cluster schema attribute for data sources.
// Data sources mark cluster as sensitive to protect credentials in output.
func GetClusterSchemaForDataSource() datasourceschema.SingleNestedAttribute {
	return datasourceschema.SingleNestedAttribute{
		Required: true,
		Sensitive: true,
		Description: "Kubernetes cluster connection configuration. " +
			"Connection details are marked as sensitive to protect credentials.",
		Attributes: GetConnectionSchemaForDataSource(),
	}
}

// GetConnectionSchemaForDataSource returns the cluster connection schema attributes for datasources.
// This converts the resource schema to datasource schema types.
func GetConnectionSchemaForDataSource() map[string]datasourceschema.Attribute {
	return ConvertResourceAttributesToDatasource(GetConnectionSchemaForResource())
}
