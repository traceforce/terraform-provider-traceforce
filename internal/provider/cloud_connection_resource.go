// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	traceforce "github.com/traceforce/traceforce-go-sdk"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &cloudConnectionResource{}
	_ resource.ResourceWithConfigure      = &cloudConnectionResource{}
	_ resource.ResourceWithImportState    = &cloudConnectionResource{}
	_ resource.ResourceWithValidateConfig = &cloudConnectionResource{}
)

// cloudProviders are the clouds with a customer-hosted collector.
var cloudProviders = []string{"aws", "m365"}

// NewCloudConnectionResource registers a customer cloud account (AWS account or
// Microsoft 365 tenant) with Traceforce at the start of terraform apply, and
// disconnects it on destroy.
func NewCloudConnectionResource() resource.Resource {
	return &cloudConnectionResource{}
}

type cloudConnectionResource struct {
	client *traceforce.Client
}

type cloudConnectionResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ProviderName  types.String `tfsdk:"provider_name"`
	Account       types.String `tfsdk:"account"`
	Name          types.String `tfsdk:"name"`
	ModuleVersion types.String `tfsdk:"module_version"`
	ModuleCommit  types.String `tfsdk:"module_commit"`
	Status        types.String `tfsdk:"status"`
}

func (r *cloudConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_connection"
}

func (r *cloudConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *cloudConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "A customer cloud account Traceforce collects from. Registered at the start of apply; disconnected on destroy. " +
			"If this organization already has the account (e.g. from an earlier collector, or a lost state), apply takes that " +
			"connection over and warns. Import an existing connection with its ID.",
		Attributes: map[string]schema.Attribute{
			// "provider" is reserved in Terraform resource blocks.
			"provider_name": schema.StringAttribute{
				Description:   "Cloud: aws or m365.",
				Required:      true,
				PlanModifiers: replace,
			},
			"account": schema.StringAttribute{
				Description:   "AWS account ID or Microsoft 365 tenant ID (stored in lowercase).",
				Required:      true,
				PlanModifiers: replace,
			},
			"name": schema.StringAttribute{
				Description: "Name shown on the connection card until the collector's first upload, which may replace it. " +
					"Removing it keeps the current name.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"module_version": schema.StringAttribute{
				Description: "Collector bundle version, e.g. 1.2.0. Apply fails when it is older than the minimum Traceforce accepts.",
				Required:    true,
			},
			"module_commit": schema.StringAttribute{
				Description: "Git commit of that bundle.",
				Optional:    true,
			},
			// The following attributes are computed.
			"id": schema.StringAttribute{
				Description:   "System generated ID of the connection.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Description: "pending (no upload yet), connected, or failed (the last upload carried an error).",
				Computed:    true,
			},
		},
	}
}

// ValidateConfig refuses a cloud without a collector before anything is sent.
func (r *cloudConnectionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cloud types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("provider_name"), &cloud)...)
	if cloud.IsNull() || cloud.IsUnknown() {
		return
	}
	for _, p := range cloudProviders {
		if cloud.ValueString() == p {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(path.Root("provider_name"), "Unknown cloud",
		fmt.Sprintf("provider_name must be one of %s, got %q.", strings.Join(cloudProviders, ", "), cloud.ValueString()))
}

// statusName is the status as a word.
func statusName(s traceforce.CloudConnectionStatus) string {
	switch s {
	case traceforce.CloudConnectionStatusPending:
		return "pending"
	case traceforce.CloudConnectionStatusConnected:
		return "connected"
	case traceforce.CloudConnectionStatusFailed:
		return "failed"
	}
	return "unknown"
}

// fromAPI copies what Traceforce stores into the model. The account is kept as
// written when it differs only in case (Traceforce stores it in lowercase), so
// it does not plan a replacement. The name is only filled when the model has
// none (import): the collector's uploads may replace it.
func fromAPI(m *cloudConnectionResourceModel, conn *traceforce.CloudConnection) {
	m.ID = types.StringValue(conn.ID)
	m.ProviderName = types.StringValue(conn.Provider)
	if !strings.EqualFold(m.Account.ValueString(), conn.Account) {
		m.Account = types.StringValue(conn.Account)
	}
	if (m.Name.IsNull() || m.Name.IsUnknown()) && conn.Name != nil {
		m.Name = types.StringValue(*conn.Name)
	} else if m.Name.IsUnknown() {
		m.Name = types.StringNull()
	}
	m.ModuleVersion = types.StringPointerValue(conn.ModuleVersion)
	m.ModuleCommit = types.StringPointerValue(conn.ModuleCommit)
	m.Status = types.StringValue(statusName(conn.Status))
}

// register sends the plan to Traceforce, for create and update alike: the API
// keeps one row per org + cloud + account, so registering again returns the same connection.
func (r *cloudConnectionResource) register(plan *cloudConnectionResourceModel) (*traceforce.CloudConnection, error) {
	conn, err := r.client.CreateCloudConnection(traceforce.CreateCloudConnectionRequest{
		Provider:      plan.ProviderName.ValueString(),
		Account:       plan.Account.ValueString(),
		Name:          plan.Name.ValueString(),
		ModuleVersion: plan.ModuleVersion.ValueString(),
		ModuleCommit:  plan.ModuleCommit.ValueString(),
	})
	if err != nil {
		return nil, err
	}
	fromAPI(plan, conn)
	return conn, nil
}

func (r *cloudConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A bundle older than the minimum version fails here (HTTP 426), before
	// anything that depends on this resource is created.
	conn, err := r.register(&plan)
	var apiErr *traceforce.APIError
	if errors.As(err, &apiErr) && apiErr.Code == traceforce.ErrCollectorTooOld {
		resp.Diagnostics.AddAttributeError(path.Root("module_version"), "Collector bundle too old",
			fmt.Sprintf("Collector bundle %s is older than the minimum Traceforce accepts. Download the latest collector "+
				"from Traceforce and apply again. Nothing was created.\n\n%s", plan.ModuleVersion.ValueString(), err))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error registering cloud connection", err.Error())
		return
	}
	if !conn.Created {
		resp.Diagnostics.AddWarning("Cloud connection already registered",
			fmt.Sprintf("This organization already had %s account %s (connection %s), so this configuration now manages it. "+
				"If another Terraform state also manages it, remove it from one of them (terraform state rm): "+
				"a destroy in either one disconnects it.", conn.Provider, conn.Account, conn.ID))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *cloudConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := r.client.GetCloudConnection(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading cloud connection", err.Error())
		return
	}
	// Disconnected or deleted in Traceforce: the next apply registers it again.
	if conn == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	fromAPI(&state, conn)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cloudConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state cloudConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.register(&plan); err != nil {
		resp.Diagnostics.AddError("Error updating cloud connection", err.Error())
		return
	}
	// The connection was removed in Traceforce after the last refresh, so the
	// API registered a new one: refresh and apply again.
	if plan.ID.ValueString() != state.ID.ValueString() {
		resp.Diagnostics.AddError("Cloud connection was replaced",
			fmt.Sprintf("Connection %s no longer exists in Traceforce; the account was registered again as %s. "+
				"Run terraform apply again to record the new connection.", state.ID.ValueString(), plan.ID.ValueString()))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete disconnects the connection (terraform destroy). Already gone counts as done.
func (r *cloudConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCloudConnection(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error disconnecting cloud connection", err.Error())
	}
}

// ImportState takes an existing connection by its ID: terraform import traceforce_cloud_connection.this <id>.
func (r *cloudConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
