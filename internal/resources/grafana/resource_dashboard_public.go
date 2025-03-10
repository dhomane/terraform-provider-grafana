package grafana

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourcePublicDashboard() *schema.Resource {
	return &schema.Resource{

		Description: `
Manages Grafana public dashboards.

**Note:** This resource is available only with Grafana 10.2+.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/dashboard-public/)
* [HTTP API](https://grafana.com/docs/grafana/next/developers/http_api/dashboard_public/)
`,

		CreateContext: CreatePublicDashboard,
		ReadContext:   ReadPublicDashboard,
		UpdateContext: UpdatePublicDashboard,
		DeleteContext: DeletePublicDashboard,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				Description: "The unique identifier of a public dashboard. " +
					"It's automatically generated if not provided when creating a public dashboard. ",
			},
			"dashboard_uid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique identifier of the original dashboard.",
			},
			"access_token": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				Description: "A public unique identifier of a public dashboard. This is used to construct its URL. " +
					"It's automatically generated if not provided when creating a public dashboard. ",
			},
			"time_selection_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to `true` to enable the time picker in the public dashboard. The default value is `false`.",
			},
			"is_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to `true` to enable the public dashboard. The default value is `false`.",
			},
			"annotations_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to `true` to show annotations. The default value is `false`.",
			},
			"share": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Set the share mode. The default value is `public`.",
			},
		},
	}
}

func CreatePublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := ClientFromNewOrgResource(meta, d)
	dashboardUID := d.Get("dashboard_uid").(string)

	publicDashboardPayload := makePublicDashboard(d)
	pd, err := client.NewPublicDashboard(dashboardUID, publicDashboardPayload)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%s:%s", orgID, pd.DashboardUID, pd.UID))
	return ReadPublicDashboard(ctx, d, meta)
}
func UpdatePublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	orgID, dashboardUID, publicDashboardUID := SplitPublicDashboardID(d.Id())
	client := meta.(*common.Client).GrafanaAPI.WithOrgID(orgID)

	publicDashboard := makePublicDashboard(d)
	pd, err := client.UpdatePublicDashboard(dashboardUID, publicDashboardUID, publicDashboard)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%s:%s", orgID, pd.DashboardUID, pd.UID))
	return ReadPublicDashboard(ctx, d, meta)
}

func DeletePublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	orgID, dashboardUID, publicDashboardUID := SplitPublicDashboardID(d.Id())
	client := meta.(*common.Client).GrafanaAPI.WithOrgID(orgID)
	return diag.FromErr(client.DeletePublicDashboard(dashboardUID, publicDashboardUID))
}

func makePublicDashboard(d *schema.ResourceData) gapi.PublicDashboardPayload {
	return gapi.PublicDashboardPayload{
		UID:                  d.Get("uid").(string),
		AccessToken:          d.Get("access_token").(string),
		TimeSelectionEnabled: d.Get("time_selection_enabled").(bool),
		IsEnabled:            d.Get("is_enabled").(bool),
		AnnotationsEnabled:   d.Get("annotations_enabled").(bool),
		Share:                d.Get("share").(string),
	}
}

func ReadPublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	orgID, dashboardUID, _ := SplitPublicDashboardID(d.Id())
	client := meta.(*common.Client).GrafanaAPI.WithOrgID(orgID)
	pd, err := client.PublicDashboardbyUID(dashboardUID)
	if err, shouldReturn := common.CheckReadError("dashboard", d, err); shouldReturn {
		return err
	}

	d.Set("org_id", strconv.FormatInt(orgID, 10))

	d.Set("uid", pd.UID)
	d.Set("dashboard_uid", pd.DashboardUID)
	d.Set("access_token", pd.AccessToken)
	d.Set("time_selection_enabled", pd.TimeSelectionEnabled)
	d.Set("is_enabled", pd.IsEnabled)
	d.Set("annotations_enabled", pd.AnnotationsEnabled)
	d.Set("share", pd.Share)

	d.SetId(fmt.Sprintf("%d:%s:%s", orgID, pd.DashboardUID, pd.UID))

	return nil
}

func SplitPublicDashboardID(id string) (int64, string, string) {
	ids := strings.Split(id, ":")
	orgID, _ := strconv.ParseInt(ids[0], 10, 64)
	return orgID, ids[1], ids[2]
}
