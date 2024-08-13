package compute_test

import (
	"fmt"
	"testing"
	"github.com/hashicorp/terraform-provider-google/google/acctest"
	"github.com/hashicorp/terraform-provider-google/google/envvar"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeFirewallPolicy_update(t *testing.T) {
	t.Parallel()

	org := envvar.GetTestOrgFromEnv(t)
	policyName := fmt.Sprintf("tf-test-firewall-policy-%s", acctest.RandString(t, 10))

	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories(t),
		CheckDestroy:             testAccCheckComputeFirewallDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccComputeFirewallPolicy_basic(org, policyName),
			},
			{
				ResourceName:      "google_compute_firewall_policy.default",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccComputeFirewallPolicy_update(org, policyName),
			},
			{
				ResourceName:      "google_compute_firewall_policy.default",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccComputeFirewallPolicy_update(org, policyName),
			},
			{
				ResourceName:      "google_compute_firewall_policy.default",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccComputeFirewallPolicy_basic(org, policyName string) string {
  return fmt.Sprintf(`
resource "google_compute_firewall_policy" "default" {
  parent      = "%s"
  short_name  = "%s"
  description = "Resource created for Terraform acceptance testing"
}
`, "organizations/"+org, policyName)
}

func testAccComputeFirewallPolicy_update(org, policyName string) string {
  return fmt.Sprintf(`
resource "google_compute_firewall_policy" "default" {
  parent      = "%s"
  short_name  = "%s"
  description = "An updated description"
}
`, "organizations/"+org, policyName)
}
