package auth

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mockPrivateState is a simple in-memory implementation for testing
type mockPrivateState struct {
	data map[string][]byte
}

func newMockPrivateState() *mockPrivateState {
	return &mockPrivateState{
		data: make(map[string][]byte),
	}
}

func (m *mockPrivateState) GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics) {
	return m.data[key], nil
}

func (m *mockPrivateState) SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics {
	if value == nil {
		delete(m.data, key)
	} else {
		m.data[key] = value
	}
	return nil
}

func TestSaveAndLoadClusterFromPrivateState(t *testing.T) {
	ctx := context.Background()
	privateState := newMockPrivateState()

	// Create a cluster object
	clusterObj := types.ObjectValueMust(
		GetConnectionAttributeTypes(),
		map[string]attr.Value{
			"host":                   types.StringValue("https://test.example.com"),
			"cluster_ca_certificate": types.StringValue("ca-cert-data"),
			"kubeconfig":             types.StringNull(),
			"context":                types.StringNull(),
			"token":                  types.StringValue("secret-token"),
			"client_certificate":     types.StringNull(),
			"client_key":             types.StringNull(),
			"insecure":               types.BoolValue(false),
			"proxy_url":              types.StringNull(),
			"exec":                   types.ObjectNull(GetExecAttributeTypes()),
		},
	)

	// Save to private state
	nullCluster, err := SaveClusterToPrivateState(ctx, privateState, clusterObj)
	if err != nil {
		t.Fatalf("Failed to save cluster: %v", err)
	}

	// Verify that returned cluster is null (for public state)
	if !nullCluster.IsNull() {
		t.Error("SaveClusterToPrivateState should return a null object for public state")
	}

	// Load from private state
	loadedCluster, err := LoadClusterFromPrivateState(ctx, privateState)
	if err != nil {
		t.Fatalf("Failed to load cluster: %v", err)
	}

	// Verify loaded cluster matches original
	if loadedCluster.IsNull() {
		t.Fatal("Loaded cluster should not be null")
	}

	// Check individual fields
	loadedModel, convErr := ObjectToConnectionModel(ctx, loadedCluster)
	if convErr != nil {
		t.Fatalf("Failed to convert loaded cluster: %v", convErr)
	}

	if loadedModel.Host.ValueString() != "https://test.example.com" {
		t.Errorf("Expected host 'https://test.example.com', got '%s'", loadedModel.Host.ValueString())
	}

	if loadedModel.Token.ValueString() != "secret-token" {
		t.Errorf("Expected token 'secret-token', got '%s'", loadedModel.Token.ValueString())
	}

	if loadedModel.ClusterCACertificate.ValueString() != "ca-cert-data" {
		t.Errorf("Expected CA cert 'ca-cert-data', got '%s'", loadedModel.ClusterCACertificate.ValueString())
	}
}

func TestSaveNullCluster(t *testing.T) {
	ctx := context.Background()
	privateState := newMockPrivateState()

	// Save null cluster
	nullCluster := types.ObjectNull(GetConnectionAttributeTypes())
	result, err := SaveClusterToPrivateState(ctx, privateState, nullCluster)
	if err != nil {
		t.Fatalf("Failed to save null cluster: %v", err)
	}

	// Should return null
	if !result.IsNull() {
		t.Error("Saving null cluster should return null")
	}

	// Should clear private state
	if len(privateState.data) > 0 {
		t.Error("Saving null cluster should clear private state")
	}
}

func TestLoadFromEmptyPrivateState(t *testing.T) {
	ctx := context.Background()
	privateState := newMockPrivateState()

	// Load from empty private state
	loadedCluster, err := LoadClusterFromPrivateState(ctx, privateState)
	if err != nil {
		t.Fatalf("Loading from empty private state should not error: %v", err)
	}

	// Should return null
	if !loadedCluster.IsNull() {
		t.Error("Loading from empty private state should return null")
	}
}

func TestSaveAndLoadClusterWithExec(t *testing.T) {
	ctx := context.Background()
	privateState := newMockPrivateState()

	// Create exec config
	execObj := types.ObjectValueMust(
		GetExecAttributeTypes(),
		map[string]attr.Value{
			"api_version": types.StringValue("client.authentication.k8s.io/v1"),
			"command":     types.StringValue("aws"),
			"args": types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("eks"),
				types.StringValue("get-token"),
				types.StringValue("--cluster-name"),
				types.StringValue("my-cluster"),
			}),
			"env": types.MapValueMust(types.StringType, map[string]attr.Value{
				"AWS_PROFILE": types.StringValue("production"),
				"AWS_REGION":  types.StringValue("us-west-2"),
			}),
		},
	)

	// Create cluster with exec
	clusterObj := types.ObjectValueMust(
		GetConnectionAttributeTypes(),
		map[string]attr.Value{
			"host":                   types.StringValue("https://eks.example.com"),
			"cluster_ca_certificate": types.StringValue("eks-ca-cert"),
			"kubeconfig":             types.StringNull(),
			"context":                types.StringNull(),
			"token":                  types.StringNull(),
			"client_certificate":     types.StringNull(),
			"client_key":             types.StringNull(),
			"insecure":               types.BoolValue(false),
			"proxy_url":              types.StringNull(),
			"exec":                   execObj,
		},
	)

	// Save to private state
	_, err := SaveClusterToPrivateState(ctx, privateState, clusterObj)
	if err != nil {
		t.Fatalf("Failed to save cluster with exec: %v", err)
	}

	// Load from private state
	loadedCluster, err := LoadClusterFromPrivateState(ctx, privateState)
	if err != nil {
		t.Fatalf("Failed to load cluster with exec: %v", err)
	}

	// Verify exec was preserved
	loadedModel, convErr := ObjectToConnectionModel(ctx, loadedCluster)
	if convErr != nil {
		t.Fatalf("Failed to convert loaded cluster: %v", convErr)
	}

	if loadedModel.Exec == nil {
		t.Fatal("Exec should not be nil")
	}

	if loadedModel.Exec.Command.ValueString() != "aws" {
		t.Errorf("Expected exec command 'aws', got '%s'", loadedModel.Exec.Command.ValueString())
	}

	if len(loadedModel.Exec.Args) != 4 {
		t.Errorf("Expected 4 exec args, got %d", len(loadedModel.Exec.Args))
	}

	if len(loadedModel.Exec.Env) != 2 {
		t.Errorf("Expected 2 exec env vars, got %d", len(loadedModel.Exec.Env))
	}
}
