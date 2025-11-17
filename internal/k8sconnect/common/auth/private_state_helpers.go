package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const privateStateKeyCluster = "cluster_connection"

// PrivateStateAccessor defines the interface for accessing private state
type PrivateStateAccessor interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// clusterJSON is a JSON-serializable representation of cluster configuration
type clusterJSON struct {
	Host                 string     `json:"host,omitempty"`
	ClusterCACertificate string     `json:"cluster_ca_certificate,omitempty"`
	Kubeconfig           string     `json:"kubeconfig,omitempty"`
	Context              string     `json:"context,omitempty"`
	Token                string     `json:"token,omitempty"`
	ClientCertificate    string     `json:"client_certificate,omitempty"`
	ClientKey            string     `json:"client_key,omitempty"`
	Insecure             bool       `json:"insecure,omitempty"`
	ProxyURL             string     `json:"proxy_url,omitempty"`
	Exec                 *execJSON  `json:"exec,omitempty"`
}

type execJSON struct {
	APIVersion string            `json:"api_version"`
	Command    string            `json:"command"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

// SaveClusterToPrivateState saves the cluster configuration to private state
// and returns a null object for storing in public state
func SaveClusterToPrivateState(ctx context.Context, privateState PrivateStateAccessor, cluster types.Object) (types.Object, error) {
	if cluster.IsNull() || cluster.IsUnknown() {
		tflog.Debug(ctx, "Cluster is null or unknown, clearing private state")
		// Clear private state
		_ = privateState.SetKey(ctx, privateStateKeyCluster, nil)
		return types.ObjectNull(GetConnectionAttributeTypes()), nil
	}

	// Convert to ClusterModel
	clusterModel, err := ObjectToConnectionModel(ctx, cluster)
	if err != nil {
		return types.ObjectNull(GetConnectionAttributeTypes()), fmt.Errorf("failed to convert cluster: %w", err)
	}

	// Convert to JSON-serializable struct
	jsonCluster := clusterJSON{}
	
	if !clusterModel.Host.IsNull() {
		jsonCluster.Host = clusterModel.Host.ValueString()
	}
	if !clusterModel.ClusterCACertificate.IsNull() {
		jsonCluster.ClusterCACertificate = clusterModel.ClusterCACertificate.ValueString()
	}
	if !clusterModel.Kubeconfig.IsNull() {
		jsonCluster.Kubeconfig = clusterModel.Kubeconfig.ValueString()
	}
	if !clusterModel.Context.IsNull() {
		jsonCluster.Context = clusterModel.Context.ValueString()
	}
	if !clusterModel.Token.IsNull() {
		jsonCluster.Token = clusterModel.Token.ValueString()
	}
	if !clusterModel.ClientCertificate.IsNull() {
		jsonCluster.ClientCertificate = clusterModel.ClientCertificate.ValueString()
	}
	if !clusterModel.ClientKey.IsNull() {
		jsonCluster.ClientKey = clusterModel.ClientKey.ValueString()
	}
	if !clusterModel.Insecure.IsNull() {
		jsonCluster.Insecure = clusterModel.Insecure.ValueBool()
	}
	if !clusterModel.ProxyURL.IsNull() {
		jsonCluster.ProxyURL = clusterModel.ProxyURL.ValueString()
	}

	if clusterModel.Exec != nil {
		jsonExec := &execJSON{
			APIVersion: clusterModel.Exec.APIVersion.ValueString(),
			Command:    clusterModel.Exec.Command.ValueString(),
		}
		
		if len(clusterModel.Exec.Args) > 0 {
			jsonExec.Args = make([]string, 0, len(clusterModel.Exec.Args))
			for _, arg := range clusterModel.Exec.Args {
				if !arg.IsNull() {
					jsonExec.Args = append(jsonExec.Args, arg.ValueString())
				}
			}
		}
		
		if len(clusterModel.Exec.Env) > 0 {
			jsonExec.Env = make(map[string]string)
			for k, v := range clusterModel.Exec.Env {
				if !v.IsNull() {
					jsonExec.Env[k] = v.ValueString()
				}
			}
		}
		
		jsonCluster.Exec = jsonExec
	}

	// Serialize to JSON
	data, err := json.Marshal(jsonCluster)
	if err != nil {
		return types.ObjectNull(GetConnectionAttributeTypes()), fmt.Errorf("failed to marshal cluster: %w", err)
	}

	// Save to private state
	diags := privateState.SetKey(ctx, privateStateKeyCluster, data)
	if diags.HasError() {
		return types.ObjectNull(GetConnectionAttributeTypes()), fmt.Errorf("failed to save to private state: %v", diags)
	}

	tflog.Debug(ctx, "Saved cluster to private state (write-only)")
	
	// Return null for public state
	return types.ObjectNull(GetConnectionAttributeTypes()), nil
}

// LoadClusterFromPrivateState loads the cluster configuration from private state
func LoadClusterFromPrivateState(ctx context.Context, privateState PrivateStateAccessor) (types.Object, error) {
	data, diags := privateState.GetKey(ctx, privateStateKeyCluster)
	if diags.HasError() {
		return types.ObjectNull(GetConnectionAttributeTypes()), fmt.Errorf("failed to get from private state: %v", diags)
	}

	if len(data) == 0 {
		tflog.Debug(ctx, "No cluster in private state")
		return types.ObjectNull(GetConnectionAttributeTypes()), nil
	}

	// Deserialize from JSON
	var jsonCluster clusterJSON
	if err := json.Unmarshal(data, &jsonCluster); err != nil {
		return types.ObjectNull(GetConnectionAttributeTypes()), fmt.Errorf("failed to unmarshal cluster: %w", err)
	}

	// Convert to ClusterModel
	clusterModel := ClusterModel{}
	
	clusterModel.Host = stringOrNull(jsonCluster.Host)
	clusterModel.ClusterCACertificate = stringOrNull(jsonCluster.ClusterCACertificate)
	clusterModel.Kubeconfig = stringOrNull(jsonCluster.Kubeconfig)
	clusterModel.Context = stringOrNull(jsonCluster.Context)
	clusterModel.Token = stringOrNull(jsonCluster.Token)
	clusterModel.ClientCertificate = stringOrNull(jsonCluster.ClientCertificate)
	clusterModel.ClientKey = stringOrNull(jsonCluster.ClientKey)
	clusterModel.Insecure = types.BoolValue(jsonCluster.Insecure)
	clusterModel.ProxyURL = stringOrNull(jsonCluster.ProxyURL)

	if jsonCluster.Exec != nil {
		execModel := &ExecAuthModel{
			APIVersion: types.StringValue(jsonCluster.Exec.APIVersion),
			Command:    types.StringValue(jsonCluster.Exec.Command),
		}
		
		if len(jsonCluster.Exec.Args) > 0 {
			execModel.Args = make([]types.String, 0, len(jsonCluster.Exec.Args))
			for _, arg := range jsonCluster.Exec.Args {
				execModel.Args = append(execModel.Args, types.StringValue(arg))
			}
		}
		
		if len(jsonCluster.Exec.Env) > 0 {
			execModel.Env = make(map[string]types.String)
			for k, v := range jsonCluster.Exec.Env {
				execModel.Env[k] = types.StringValue(v)
			}
		}
		
		clusterModel.Exec = execModel
	}

	// Convert to types.Object
	clusterObj, err := ConnectionToObject(ctx, clusterModel)
	if err != nil {
		return types.ObjectNull(GetConnectionAttributeTypes()), fmt.Errorf("failed to convert to object: %w", err)
	}

	tflog.Debug(ctx, "Loaded cluster from private state (write-only)")
	return clusterObj, nil
}

// stringOrNull returns a StringValue if the string is non-empty, otherwise StringNull
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// MergeClusterFromConfigAndPrivateState merges cluster from config (if present) and private state (fallback)
// This is used during Read operations where config might have the cluster value
func MergeClusterFromConfigAndPrivateState(ctx context.Context, configCluster types.Object, privateState PrivateStateAccessor) (types.Object, error) {
	// If config has cluster, use it (and save to private state for next time)
	if !configCluster.IsNull() && !configCluster.IsUnknown() {
		tflog.Debug(ctx, "Using cluster from config")
		// Save to private state for future Read operations
		_, err := SaveClusterToPrivateState(ctx, privateState, configCluster)
		if err != nil {
			tflog.Warn(ctx, "Failed to save cluster to private state", map[string]interface{}{
				"error": err.Error(),
			})
		}
		return configCluster, nil
	}

	// Fallback to private state
	tflog.Debug(ctx, "Config cluster is null, loading from private state")
	return LoadClusterFromPrivateState(ctx, privateState)
}
