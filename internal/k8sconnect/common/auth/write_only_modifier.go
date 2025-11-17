package auth

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// writeOnlyModifier is a plan modifier that implements write-only behavior for the cluster attribute.
// It ensures that:
// - Configuration values are available during Create/Update/Delete operations
// - The attribute is set to null in state (never persisted)
// - No drift detection occurs on this attribute
type writeOnlyModifier struct{}

// WriteOnly returns a plan modifier for write-only object attributes
func WriteOnly() planmodifier.Object {
	return writeOnlyModifier{}
}

func (m writeOnlyModifier) Description(ctx context.Context) string {
	return "Makes the cluster attribute write-only (used from config, not stored in state)"
}

func (m writeOnlyModifier) MarkdownDescription(ctx context.Context) string {
	return "Makes the cluster attribute write-only (used from config, not stored in state)"
}

func (m writeOnlyModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	// If destroying the resource, no modification needed
	if req.Plan.Raw.IsNull() {
		return
	}

	// If config has a value, use it in the plan
	// (it will be saved to private state and set to null in public state by the resource implementation)
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		resp.PlanValue = req.ConfigValue
		return
	}

	// Otherwise set to null (state will be null, and that's expected)
	resp.PlanValue = types.ObjectNull(req.StateValue.AttributeTypes(ctx))
}
