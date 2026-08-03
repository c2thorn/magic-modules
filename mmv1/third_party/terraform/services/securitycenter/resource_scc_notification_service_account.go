package securitycenter

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-google/google/registry"
	rmClient "github.com/hashicorp/terraform-provider-google/google/services/resourcemanager/client"
	"github.com/hashicorp/terraform-provider-google/google/services/serviceusage"
	"github.com/hashicorp/terraform-provider-google/google/tpgresource"
	transport_tpg "github.com/hashicorp/terraform-provider-google/google/transport"
)

func ResourceSecurityCenterNotificationServiceAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityCenterNotificationServiceAccountCreate,
		Read:   resourceSecurityCenterNotificationServiceAccountRead,
		Delete: resourceSecurityCenterNotificationServiceAccountDelete,

		Importer: &schema.ResourceImporter{
			StateContext: ResourceSecurityCenterNotificationServiceAccountImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		CustomizeDiff: customdiff.All(
			func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
				if _, ok := diff.GetOk("organization"); ok {
					return nil
				}
				if _, ok := diff.GetOk("folder"); ok {
					return nil
				}
				return tpgresource.DefaultProviderProject(ctx, diff, meta)
			},
		),

		Schema: map[string]*schema.Schema{
			"organization": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"organization", "folder", "project"},
			},
			"folder": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"organization", "folder", "project"},
			},
			"project": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"organization", "folder", "project"},
			},
			"email": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: `The email address of the Cloud Security Command Center Notification service account.`,
			},
			"member": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: `The Identity of the Cloud Security Command Center Notification service account in the form 'serviceAccount:{email}'. This value is often used to refer to the service account in order to grant IAM permissions.`,
			},
		},
		UseJSONNumber: true,
	}
}

func getSccNotificationServiceAccountProjectNumber(d *schema.ResourceData, config *transport_tpg.Config, project, userAgent string) (string, error) {
	if _, err := strconv.ParseInt(project, 10, 64); err == nil {
		return project, nil
	}

	log.Printf("[DEBUG] Retrieving project number for SCC notification service account by doing a GET with the project id %q", project)
	billingProject := project
	if bp, err := tpgresource.GetBillingProject(d, config); err == nil {
		billingProject = bp
	}

	getProjectCall := rmClient.NewClient(config, userAgent).Projects.Get(project)
	if config.UserProjectOverride {
		getProjectCall.Header().Add("X-Goog-User-Project", billingProject)
	}
	projectCall, err := getProjectCall.Do()
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve project %s: %w", project, err)
	}

	return strconv.FormatInt(projectCall.ProjectNumber, 10), nil
}

func getSccNotificationServiceAccountDetails(d *schema.ResourceData, config *transport_tpg.Config, userAgent string) (string, string, string, error) {
	if v, ok := d.GetOk("organization"); ok {
		org := v.(string)
		url, err := tpgresource.ReplaceVars(d, config, "{{ServiceUsageBasePath}}organizations/{{organization}}/services/securitycenter.googleapis.com:generateServiceIdentity")
		if err != nil {
			return "", "", "", err
		}
		email := fmt.Sprintf("service-org-%s@gcp-sa-scc-notification.iam.gserviceaccount.com", org)
		parentId := fmt.Sprintf("organizations/%s", org)
		return url, email, parentId, nil
	}

	if v, ok := d.GetOk("folder"); ok {
		folder := v.(string)
		url, err := tpgresource.ReplaceVars(d, config, "{{ServiceUsageBasePath}}folders/{{folder}}/services/securitycenter.googleapis.com:generateServiceIdentity")
		if err != nil {
			return "", "", "", err
		}
		email := fmt.Sprintf("service-folder-%s@gcp-sa-scc-notification.iam.gserviceaccount.com", folder)
		parentId := fmt.Sprintf("folders/%s", folder)
		return url, email, parentId, nil
	}

	if v, ok := d.GetOk("project"); ok {
		project := v.(string)
		projectNumber, err := getSccNotificationServiceAccountProjectNumber(d, config, project, userAgent)
		if err != nil {
			return "", "", "", err
		}
		url, err := tpgresource.ReplaceVars(d, config, "{{ServiceUsageBasePath}}projects/{{project}}/services/securitycenter.googleapis.com:generateServiceIdentity")
		if err != nil {
			return "", "", "", err
		}
		email := fmt.Sprintf("service-%s@gcp-sa-scc-notification.iam.gserviceaccount.com", projectNumber)
		parentId := fmt.Sprintf("projects/%s", project)
		return url, email, parentId, nil
	}

	return "", "", "", fmt.Errorf("one of organization, folder, or project must be specified")
}

func resourceSecurityCenterNotificationServiceAccountCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*transport_tpg.Config)
	userAgent, err := tpgresource.GenerateUserAgentString(d, config.UserAgent)
	if err != nil {
		return err
	}

	url, email, parentId, err := getSccNotificationServiceAccountDetails(d, config, userAgent)
	if err != nil {
		return err
	}

	billingProject := ""
	if bp, err := tpgresource.GetBillingProject(d, config); err == nil {
		billingProject = bp
	}

	res, err := transport_tpg.SendRequest(transport_tpg.SendRequestOptions{
		Config:    config,
		Method:    "POST",
		Project:   billingProject,
		RawURL:    url,
		UserAgent: userAgent,
		Timeout:   d.Timeout(schema.TimeoutCreate),
	})
	if err != nil {
		return fmt.Errorf("Error creating Cloud SCC Notification Service Account: %s", err)
	}

	var opRes map[string]interface{}
	err = serviceusage.ServiceUsageOperationWaitTimeWithResponse(
		config, res, &opRes, billingProject, "Creating Cloud SCC Notification Service Account", userAgent,
		d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return err
	}

	d.SetId(email)
	if err := d.Set("email", email); err != nil {
		return fmt.Errorf("Error setting email: %s", err)
	}
	if err := d.Set("member", "serviceAccount:"+email); err != nil {
		return fmt.Errorf("Error setting member: %s", err)
	}

	log.Printf("[DEBUG] Created Cloud SCC Notification Service Account %q for parent %q", email, parentId)
	return nil
}

func resourceSecurityCenterNotificationServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*transport_tpg.Config)
	userAgent, err := tpgresource.GenerateUserAgentString(d, config.UserAgent)
	if err != nil {
		return err
	}

	_, email, _, err := getSccNotificationServiceAccountDetails(d, config, userAgent)
	if err != nil {
		return err
	}

	d.SetId(email)
	if err := d.Set("email", email); err != nil {
		return fmt.Errorf("Error setting email: %s", err)
	}
	if err := d.Set("member", "serviceAccount:"+email); err != nil {
		return fmt.Errorf("Error setting member: %s", err)
	}
	return nil
}

func resourceSecurityCenterNotificationServiceAccountDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func ResourceSecurityCenterNotificationServiceAccountImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	if strings.HasPrefix(id, "organizations/") {
		if err := d.Set("organization", strings.TrimPrefix(id, "organizations/")); err != nil {
			return nil, fmt.Errorf("Error setting organization: %s", err)
		}
	} else if strings.HasPrefix(id, "folders/") {
		if err := d.Set("folder", strings.TrimPrefix(id, "folders/")); err != nil {
			return nil, fmt.Errorf("Error setting folder: %s", err)
		}
	} else if strings.HasPrefix(id, "projects/") {
		if err := d.Set("project", strings.TrimPrefix(id, "projects/")); err != nil {
			return nil, fmt.Errorf("Error setting project: %s", err)
		}
	} else if strings.Contains(id, "@gcp-sa-scc-notification.iam.gserviceaccount.com") {
		parts := strings.Split(id, "@")
		accountPart := parts[0]
		if strings.HasPrefix(accountPart, "service-org-") {
			if err := d.Set("organization", strings.TrimPrefix(accountPart, "service-org-")); err != nil {
				return nil, fmt.Errorf("Error setting organization: %s", err)
			}
		} else if strings.HasPrefix(accountPart, "service-folder-") {
			if err := d.Set("folder", strings.TrimPrefix(accountPart, "service-folder-")); err != nil {
				return nil, fmt.Errorf("Error setting folder: %s", err)
			}
		} else if strings.HasPrefix(accountPart, "service-") {
			if err := d.Set("project", strings.TrimPrefix(accountPart, "service-")); err != nil {
				return nil, fmt.Errorf("Error setting project: %s", err)
			}
		} else {
			return nil, fmt.Errorf("Unsupported import format %q for google_scc_notification_service_account", id)
		}
	} else {
		return nil, fmt.Errorf("Unsupported import format %q for google_scc_notification_service_account: expected organizations/{org_id}, folders/{folder_id}, projects/{project_id}, or email address", id)
	}
	return []*schema.ResourceData{d}, nil
}

func init() {
	registry.Schema{
		Name:        "google_scc_notification_service_account",
		ProductName: "securitycenter",
		Type:        registry.SchemaTypeResource,
		Schema:      ResourceSecurityCenterNotificationServiceAccount(),
	}.Register()
}
