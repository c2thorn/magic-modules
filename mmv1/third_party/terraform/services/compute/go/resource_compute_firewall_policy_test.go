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

	org := fmt.Sprintf("organizations/%s", envvar.GetTestOrgFromEnv(t))
	folderName := fmt.Sprintf("tf-test-folder-%s", acctest.RandString(t, 10))
	policyName := fmt.Sprintf("tf-test-firewall-policy-%s", acctest.RandString(t, 10))

	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories(t),
		CheckDestroy:             testAccCheckComputeFirewallDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccComputeFirewallPolicy_basic(folderName, org, policyName),
			},
			{
				ResourceName:      "google_compute_firewall_policy.default",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccComputeFirewallPolicy_update(folderName, org, policyName),
			},
			{
				ResourceName:      "google_compute_firewall_policy.default",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccComputeFirewallPolicy_update(folderName, org, policyName),
			},
			{
				ResourceName:      "google_compute_firewall_policy.default",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccComputeFirewallPolicy_basic(folderName, org, policyName string) string {
  return fmt.Sprintf(`
resource "google_folder" "folder" {
  display_name = "%s"
  parent       = "%s"
  deletion_protection = false
}

resource "google_compute_firewall_policy" "default" {
  provider = google-beta
  
  parent      = "%s"
  short_name  = "%s"
  description = "Resource created for Terraform acceptance testing"
}
`, folderName, org, org, policyName)
}

func testAccComputeFirewallPolicy_update(folderName, org, policyName string) string {
  return fmt.Sprintf(`
resource "google_folder" "folder" {
  display_name = "%s"
  parent       = "%s"
  deletion_protection = false
}

resource "google_compute_firewall_policy" "default" {
  provider = google-beta
  
  parent      = "%s"
  short_name  = "%s"
  description = "An updated description"
}
`, folderName, org, org, policyName)
}
