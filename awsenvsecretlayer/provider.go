package awsenvsecretlayer

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_REGION", nil),
			},
			"profile": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_PROFILE", nil),
				Description: "The profile name as set in the shared credentials file for the provider.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"awsenvsecretlayer_lambda": resourceLambdaLayer(),
		},
		DataSourcesMap: map[string]*schema.Resource{},
		ConfigureContextFunc: func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
			region := d.Get("region").(string)
			profile := d.Get("profile").(string)

			opts := session.Options{
				Config: aws.Config{
					Region: aws.String(region),
				},
				SharedConfigState: session.SharedConfigEnable,
			}

			if profile != "" {
				opts.Profile = profile
			}

			sess, err := session.NewSessionWithOptions(opts)
			if err != nil {
				return nil, diag.FromErr(err)
			}

			return sess, nil
		},
	}
}
