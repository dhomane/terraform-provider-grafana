package slo_test

import (
	"fmt"
	"regexp"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := acctest.RandomWithPrefix("SLO Terraform Testing")

	var slo gapi.Slo
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests destroy
		CheckDestroy: testAccSloCheckDestroy(&slo),
		Steps: []resource.TestStep{
			{
				// Tests Create
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_slo/resource.tf", map[string]string{
					"Terraform Testing": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.test", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.test", "id"),
					resource.TestCheckResourceAttr("grafana_slo.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_slo.test", "description", "Terraform Description"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.type", "freeform"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.freeform.0.query", "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.value", "0.995"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.window", "30d"),
				),
			},
			{
				// Tests Update
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_slo/resource_update.tf", map[string]string{
					"Terraform Testing": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.update", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.update", "id"),
					resource.TestCheckResourceAttr("grafana_slo.update", "name", "Updated - "+randomName),
					resource.TestCheckResourceAttr("grafana_slo.update", "description", "Updated - Terraform Description"),
					resource.TestCheckResourceAttr("grafana_slo.update", "query.0.type", "freeform"),
					resource.TestCheckResourceAttr("grafana_slo.update", "query.0.freeform.0.query", "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"),
					resource.TestCheckResourceAttr("grafana_slo.update", "objectives.0.value", "0.9995"),
					resource.TestCheckResourceAttr("grafana_slo.update", "objectives.0.window", "7d"),
				),
			},
			{
				// Tests that No Alerting Rules are Generated when No Alerting Field is defined on the Terraform State File
				Config: noAlert(randomName + " - No Alerting Check"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.no_alert", &slo),
					testAlertingExists(false, "grafana_slo.no_alert", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.no_alert", "id"),
					resource.TestCheckResourceAttr("grafana_slo.no_alert", "name", randomName+" - No Alerting Check"),
				),
			},
			{
				// Tests that Alerting Rules are Generated when an Empty Alerting Field is defined on the Terraform State File
				Config: emptyAlert(randomName + " - Empty Alerting Check"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.empty_alert", &slo),
					testAlertingExists(true, "grafana_slo.empty_alert", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.empty_alert", "id"),
					resource.TestCheckResourceAttr("grafana_slo.empty_alert", "name", randomName+" - Empty Alerting Check"),
				),
			},
			{
				// Tests Create
				Config: testutils.TestAccExample(t, "resources/grafana_slo/resource_ratio.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.ratio", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.ratio", "id"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "name", "Terraform Testing - Ratio Query"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "description", "Terraform Description - Ratio Query"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.type", "ratio"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.ratio.0.success_metric", "kubelet_http_requests_total{status!~\"5..\"}"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.ratio.0.total_metric", "kubelet_http_requests_total"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.ratio.0.group_by_labels.0", "job"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.ratio.0.group_by_labels.1", "instance"),
				),
			},
		},
	})
}

func testAccSloCheckExists(rn string, slo *gapi.Slo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		gotSlo, err := client.GetSlo(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("error getting SLO: %s", err)
		}

		*slo = gotSlo

		return nil
	}
}

func testAlertingExists(expectation bool, rn string, slo *gapi.Slo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[rn]
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		gotSlo, _ := client.GetSlo(rs.Primary.ID)
		*slo = gotSlo

		if slo.Alerting == nil && expectation == false {
			return nil
		}

		if slo.Alerting != nil && expectation == true {
			return nil
		}

		return fmt.Errorf("SLO Alerting expectation mismatch")
	}
}

func testAccSloCheckDestroy(slo *gapi.Slo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		client.DeleteSlo(slo.UUID)

		return nil
	}
}

const sloObjectivesInvalid = `
resource  "grafana_slo" "invalid" {
  name            = "Test SLO"
  description     = "Description Test SLO"
  query {
	freeform {
		query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
	}
    type = "freeform"
  }
  destination_datasource {
	type = "mimir"
	uid = "grafanacloud-prom"
  }
  objectives {
	value  = 1.5
    window = "1m"
  }
}
`

func emptyAlert(name string) string {
	return fmt.Sprintf(`
	resource "grafana_slo" "empty_alert" {
	  description = "%[1]s"
	  name        = "%[1]s"
	  objectives {
		value  = 0.995
		window = "28d"
	  }
	  destination_datasource {
		type = "mimir"
		uid = "grafanacloud-prom"
	  }
	  query {
		type = "freeform"
		freeform {
		  query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
		}
	  }
	  alerting {}
	}
	`, name)
}

func noAlert(name string) string {
	return fmt.Sprintf(`
resource "grafana_slo" "no_alert" {
	description = "%[1]s"
	name        = "%[1]s"
  objectives {
    value  = 0.995
    window = "28d"
  }
  destination_datasource {
	type = "mimir"
	uid = "grafanacloud-prom"
  }
  query {
    type = "freeform"
    freeform {
      query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    }
  }
}
`, name)
}

func TestAccResourceInvalidSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      sloObjectivesInvalid,
				ExpectError: regexp.MustCompile("Error:"),
			},
		},
	})
}
