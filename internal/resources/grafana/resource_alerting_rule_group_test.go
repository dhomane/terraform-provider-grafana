package grafana_test

import (
	"fmt"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAlertRule_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group gapi.RuleGroup

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testAlertRuleCheckDestroy(&group),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.my_alert_rule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test update content.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule_group/resource.tf", map[string]string{
					"My Alert Rule 1": "A Different Rule",
				}),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.name", "A Different Rule"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.for", "2m"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
			// Test rename group.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule_group/resource.tf", map[string]string{
					"My Rule Group": "A Different Rule Group",
				}),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "A Different Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.for", "2m"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
			// Test change interval.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule_group/resource.tf", map[string]string{
					"240": "360",
				}),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "360"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
				),
			},
			// Test re-parent folder.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_reparent_folder.tf"),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "folder_uid", "test-uid"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.for", "2m"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
		},
	})
}

func TestAccAlertRule_model(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group gapi.RuleGroup

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testAlertRuleCheckDestroy(&group),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_model_normalization.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Model normalization means that default values for fields in the model JSON are not
					// included in the state, to prevent permadiffs, but non-default values must be included.
					resource.TestCheckResourceAttr("grafana_rule_group.rg_model_params_defaults",
						"rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
					resource.TestCheckResourceAttr("grafana_rule_group.rg_model_params_omitted",
						"rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
					resource.TestCheckResourceAttr("grafana_rule_group.rg_model_params_non_default",
						"rule.0.data.0.model", "{\"hide\":false,\"intervalMs\":1001,\"maxDataPoints\":43201,\"refId\":\"A\"}"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.rg_model_params_defaults",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_rule_group.rg_model_params_omitted",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_rule_group.rg_model_params_non_default",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAlertRule_compound(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group gapi.RuleGroup

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testAlertRuleCheckDestroy(&group),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_multi_rule_group.tf"),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.my_multi_alert_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "name", "My Multi-Alert Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.#", "2"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.my_multi_alert_group",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test update.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule_group/_acc_multi_rule_group.tf", map[string]string{
					"Rule 1": "asdf",
				}),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.my_multi_alert_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "name", "My Multi-Alert Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.#", "2"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.0.name", "My Alert asdf"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.1.name", "My Alert Rule 2"),
				),
			},
			// Test addition of a rule to an existing group.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_multi_rule_group_added.tf"),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.my_multi_alert_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "name", "My Multi-Alert Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.#", "3"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.1.name", "My Alert Rule 2"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.2.name", "My Alert Rule 3"),
				),
			},
			// Test removal of rules from an existing group.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_multi_rule_group_subtracted.tf"),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.my_multi_alert_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "name", "My Multi-Alert Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.#", "2"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.1.name", "My Alert Rule 2"),
				),
			},
		},
	})
}

func TestAccAlertRule_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group gapi.RuleGroup
	var org gapi.Org
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testAlertRuleCheckDestroy(&group),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccAlertRuleGroupInOrgConfig(name, 240),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.test", &group),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule_group.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
			// Test update content.
			{
				Config: testAccAlertRuleGroupInOrgConfig(name, 360),
				Check: resource.ComposeTestCheckFunc(
					testRuleGroupCheckExists("grafana_rule_group.test", &group),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule_group.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "interval_seconds", "360"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testRuleGroupCheckExists(rname string, g *gapi.RuleGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rname]
		if !ok {
			return fmt.Errorf("resource not found: %s, resources: %#v", rname, s.RootModule().Resources)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		orgID, idStr := grafana.SplitOrgResourceID(resource.Primary.ID)
		key := grafana.UnpackGroupID(idStr)
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)
		grp, err := client.AlertRuleGroup(key.FolderUID, key.Name)
		if err != nil {
			return fmt.Errorf("error getting resource: %s", err)
		}

		*g = grp
		return nil
	}
}

func testAlertRuleCheckDestroy(group *gapi.RuleGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		if len(group.Rules) > 0 {
			client = client.WithOrgID(group.Rules[0].OrgID)
		}
		_, err := client.AlertRuleGroup(group.FolderUID, group.Title)
		if err == nil && strings.HasPrefix(err.Error(), "status: 404") {
			return fmt.Errorf("rule group still exists on the server")
		}
		return nil
	}
}

func testAccAlertRuleGroupInOrgConfig(name string, interval int) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_folder" "test" {
	org_id = grafana_organization.test.id
	title = "%[1]s"
}

resource "grafana_rule_group" "test" {
	org_id = grafana_organization.test.id
	name             = "%[1]s"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = %[2]d
	rule {
		name           = "My Alert Rule 1"
		for            = "2m"
		condition      = "B"
		no_data_state  = "NoData"
		exec_err_state = "Alerting"
		is_paused = false
		data {
			ref_id     = "A"
			query_type = ""
			relative_time_range {
				from = 600
				to   = 0
			}
			datasource_uid = "PD8C576611E62080A"
			model = jsonencode({
				hide          = false
				intervalMs    = 1000
				maxDataPoints = 43200
				refId         = "A"
			})
		}
	}
}
`, name, interval)
}
