// Package tfutil holds small helpers that the resource and data-source
// packages share.
package tfutil

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

// ClientFromProviderData converts a provider's ProviderData into a
// *clerkapi.Client. It returns a nil client with no diagnostics when
// providerData is nil: the framework passes nil on the first Configure pass,
// before the provider itself is configured, and callers must treat that as
// "nothing to do yet", not as an error.
func ClientFromProviderData(providerData any) (*clerkapi.Client, diag.Diagnostics) {
	var diags diag.Diagnostics
	if providerData == nil {
		return nil, diags
	}
	c, ok := providerData.(*clerkapi.Client)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("expected *clerkapi.Client, got %T", providerData))
		return nil, diags
	}
	return c, diags
}
