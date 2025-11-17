package auth

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestWriteOnlyModifier(t *testing.T) {
	tests := []struct {
		name           string
		configValue    types.Object
		stateValue     types.Object
		expectedResult types.Object
		description    string
	}{
		{
			name: "config has value - plan uses config",
			configValue: types.ObjectValueMust(
				GetConnectionAttributeTypes(),
				map[string]attr.Value{
					"host":                   types.StringValue("https://test.example.com"),
					"cluster_ca_certificate": types.StringValue("ca-cert"),
					"kubeconfig":             types.StringNull(),
					"context":                types.StringNull(),
					"token":                  types.StringValue("token123"),
					"client_certificate":     types.StringNull(),
					"client_key":             types.StringNull(),
					"insecure":               types.BoolValue(false),
					"proxy_url":              types.StringNull(),
					"exec":                   types.ObjectNull(GetExecAttributeTypes()),
				},
			),
			stateValue: types.ObjectNull(GetConnectionAttributeTypes()),
			expectedResult: types.ObjectValueMust(
				GetConnectionAttributeTypes(),
				map[string]attr.Value{
					"host":                   types.StringValue("https://test.example.com"),
					"cluster_ca_certificate": types.StringValue("ca-cert"),
					"kubeconfig":             types.StringNull(),
					"context":                types.StringNull(),
					"token":                  types.StringValue("token123"),
					"client_certificate":     types.StringNull(),
					"client_key":             types.StringNull(),
					"insecure":               types.BoolValue(false),
					"proxy_url":              types.StringNull(),
					"exec":                   types.ObjectNull(GetExecAttributeTypes()),
				},
			),
			description: "When config has a value, plan should use that value",
		},
		{
			name:           "config is null - plan is null",
			configValue:    types.ObjectNull(GetConnectionAttributeTypes()),
			stateValue:     types.ObjectNull(GetConnectionAttributeTypes()),
			expectedResult: types.ObjectNull(GetConnectionAttributeTypes()),
			description:    "When config is null, plan should be null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifier := WriteOnly()
			ctx := context.Background()

			// Create request
			// For this test, we're simulating a non-destroy operation
			// So Plan.Raw should have a value (not null)
			req := planmodifier.ObjectRequest{
				ConfigValue: tt.configValue,
				StateValue:  tt.stateValue,
				Plan: tfsdk.Plan{
					Raw: tftypes.NewValue(tftypes.Object{
						AttributeTypes: map[string]tftypes.Type{
							"cluster": tftypes.Object{},
						},
					}, map[string]tftypes.Value{
						"cluster": tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{}),
					}),
				},
			}

			// Create response
			resp := &planmodifier.ObjectResponse{
				PlanValue: tt.stateValue, // Start with state value
			}

			// Call modifier
			modifier.PlanModifyObject(ctx, req, resp)

			// Check result
			if !resp.PlanValue.Equal(tt.expectedResult) {
				t.Errorf("%s: expected plan value %v, got %v", tt.description, tt.expectedResult, resp.PlanValue)
			}
		})
	}
}

func TestWriteOnlyModifierDescription(t *testing.T) {
	modifier := WriteOnly()
	ctx := context.Background()

	description := modifier.Description(ctx)
	if description == "" {
		t.Error("Description should not be empty")
	}

	markdownDesc := modifier.MarkdownDescription(ctx)
	if markdownDesc == "" {
		t.Error("MarkdownDescription should not be empty")
	}
}
