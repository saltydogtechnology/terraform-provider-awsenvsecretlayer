package awsenvsecretlayer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	hclog "github.com/hashicorp/go-hclog"
)

var logger = hclog.New(&hclog.LoggerOptions{
	Level:      hclog.Debug,
	Output:     os.Stderr,
	JSONFormat: false,
})

func resourceLambdaLayer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceLambdaLayerCreate,
		ReadContext:   resourceLambdaLayerRead,
		DeleteContext: resourceLambdaLayerDelete,
		UpdateContext: resourceLambdaLayerUpdate,
		CustomizeDiff: resourceLambdaLayerCustomizeDiff,
		Schema: map[string]*schema.Schema{
			"yaml_config": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"secrets_arns": {
				Type:      schema.TypeList,
				Optional:  true,
				Sensitive: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"envs_map": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"stored_secrets_hash": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"layer_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"file_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"license_files": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"compatible_runtimes": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"skip_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"track_actual_secrets": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"need_update": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"layer_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceLambdaLayerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sess := m.(*session.Session)

	content, err, secretHash := createEnvFileContent(d, sess)
	if err != nil {
		return diag.FromErr(err)
	}

	licenseFilesRaw := d.Get("license_files").([]interface{})
	licenseFiles := make([]string, len(licenseFilesRaw))
	for i, lf := range licenseFilesRaw {
		licenseFiles[i] = lf.(string)
	}

	zipFile, err := CreateZipFile(d.Get("file_name").(string), []byte(content), licenseFiles)
	if err != nil {
		return diag.FromErr(err)
	}

	zipFileBytes, err := ReadZipFile(zipFile)
	if err != nil {
		return diag.FromErr(err)
	}

	var compatibleRuntimes []*string
	if v, ok := d.GetOk("compatible_runtimes"); ok {
		compatibleRuntimes = expandStringList(v.([]interface{}))
	}

	input := &lambda.PublishLayerVersionInput{
		LayerName:          aws.String(d.Get("layer_name").(string)),
		CompatibleRuntimes: compatibleRuntimes,
		Content: &lambda.LayerVersionContentInput{
			ZipFile: zipFileBytes,
		},
	}

	lambdaSvc := lambda.New(sess)
	output, err := lambdaSvc.PublishLayerVersion(input)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s", *output.LayerArn))
	d.Set("layer_id", fmt.Sprintf("%s:%d", *output.LayerArn, *output.Version))
	logger.Debug("DEBUG layer id", "value", fmt.Sprintf("%s:%d", *output.LayerArn, *output.Version))
	d.Set("stored_secrets_hash", secretHash)

	return resourceLambdaLayerRead(ctx, d, m)
}

func resourceLambdaLayerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sess := m.(*session.Session)
	secretsArns := d.Get("secrets_arns").([]interface{})
	storedSecretsHash := d.Get("stored_secrets_hash").(string)

	fetchedSecretsHash, err := fetchSecrets(secretsArns, sess, false, d.Get("track_actual_secrets").(bool))
	if err != nil {
		return diag.FromErr(err)
	}

	if fetchedSecretsHash != storedSecretsHash {
		d.Set("need_update", true)
	}

	return nil
}

func resourceLambdaLayerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	logger.Debug("running resourceLambdaLayerUpdate...")

	sess := m.(*session.Session)
	secretsArns := d.Get("secrets_arns").([]interface{})
	storedSecretsHash := d.Get("stored_secrets_hash").(string)
	logger.Debug("resourceLambdaLayerUpdate storedSecretsHash", "value", storedSecretsHash)

	// Fetch secrets using the fetchSecrets function
	fetchedSecretsHash, err := fetchSecrets(secretsArns, sess, d.HasChange("secrets_arns"), d.Get("track_actual_secrets").(bool))
	if err != nil {
		return diag.FromErr(err)
	}

	// Check if storedSecretsHash and fetchedSecrets are equal
	secretsEqual := storedSecretsHash == fetchedSecretsHash

	if d.HasChanges("layer_name", "yaml_config", "secrets_arns", "envs_map", "file_name", "compatible_runtimes", "license_files") || !secretsEqual || d.Get("need_update").(bool) {
		logger.Debug("resourceLambdaLayerUpdate HasChanges", "value", true)
		skipDestroy := d.Get("skip_destroy").(bool)
		logger.Debug("skipDestroy", "value", skipDestroy)

		if !skipDestroy {
			err := resourceLambdaLayerDelete(ctx, d, m)
			if err != nil {
				return err
			}
		}

		return resourceLambdaLayerCreate(ctx, d, m)
	}

	return nil
}

func deleteLambdaLayerVersions(layerARN string, sess *session.Session) error {
    lambdaSvc := lambda.New(sess)
    layerName := extractLayerName(layerARN)
	logger.Debug("deleteLambdaLayerVersions", "layerARN", layerARN)
	logger.Debug("deleteLambdaLayerVersions", "layerName", layerName)

    listLayerVersionsOutput, err := lambdaSvc.ListLayerVersions(&lambda.ListLayerVersionsInput{
        LayerName: aws.String(layerName),
    })
    if err != nil {
        return err
    }

    logger.Debug("deleteLambdaLayerVersions", "listLayerVersionsOutput", fmt.Sprintf("%+v", listLayerVersionsOutput))

    for _, layerVersion := range listLayerVersionsOutput.LayerVersions {
		logger.Debug("deleteLambdaLayerVersions", "layerVersion", layerVersion)
        _, err = lambdaSvc.DeleteLayerVersion(&lambda.DeleteLayerVersionInput{
            LayerName:     aws.String(layerName),
            VersionNumber: layerVersion.Version,
        })
        if err != nil {
            return err
        }
    }

    return nil
}

func extractLayerName(layerARN string) string {
	logger.Debug("extractLayerName", "layerARN", layerARN)
    arnParts := strings.Split(layerARN, ":")
    if len(arnParts) < 7 {
        logger.Error("Invalid Lambda Layer ARN", "ARN", layerARN)
        return ""
    }
    layerName := arnParts[6]
    logger.Debug("extractLayerName", "layerName", layerName)
    return layerName
}

func resourceLambdaLayerCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	sess := meta.(*session.Session)
	storedSecretsHash := diff.Get("stored_secrets_hash").(string)
	secretsArns := diff.Get("secrets_arns").([]interface{})

	// Fetch secrets hash using the fetchSecrets function
	fetchedSecretsHash, err := fetchSecrets(secretsArns, sess, diff.HasChange("secrets_arns"), diff.Get("track_actual_secrets").(bool))
	if err != nil {
		return err
	}

	logger.Debug("resourceLambdaLayerCustomizeDiff fetchedSecretsHash", "value", fetchedSecretsHash)
	logger.Debug("resourceLambdaLayerCustomizeDiff storedSecretsHash", "value", storedSecretsHash)

	// Set new stored_secrets_hash if fetchedSecretsHash is different from storedSecretsHash
	if fetchedSecretsHash != storedSecretsHash {
		if err := diff.SetNew("stored_secrets_hash", fetchedSecretsHash); err != nil {
			return err
		}

		// Mark a field to be recomputed
		if err := diff.SetNewComputed("layer_id"); err != nil {
			return err
		}
		
	}

	if diff.HasChanges("layer_name", "yaml_config", "envs_map", "file_name", "compatible_runtimes", "license_files") {
        if err := diff.SetNewComputed("layer_id"); err != nil {
            return err
        }
    }

	return nil
}

func resourceLambdaLayerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    sess := m.(*session.Session)
    layerARN := d.Id()
	logger.Debug("resourceLambdaLayerDelete", "layerARN", layerARN)

    err := deleteLambdaLayerVersions(layerARN, sess)
    if err != nil {
        return diag.FromErr(err)
    }

    return nil
}

func expandStringList(lst []interface{}) []*string {
	if len(lst) == 0 {
		return nil
	}

	strings := make([]*string, len(lst))
	for i, v := range lst {
		strings[i] = aws.String(v.(string))
	}

	return strings
}

// Function to convert map to .env format
func mapToEnvFormat(envsMap map[string]interface{}) string {
	var envBuilder strings.Builder

	for k, v := range envsMap {
		envBuilder.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}

	return envBuilder.String()
}

func createEnvFileContent(d *schema.ResourceData, sess *session.Session) (string, error, string) {
	yamlConfig := d.Get("yaml_config").(string)
	secretsArns := d.Get("secrets_arns").([]interface{})
	envsMap := d.Get("envs_map").(map[string]interface{})

	mergedVars, err := processYamlConfig(yamlConfig)
	if err != nil {
		return "", err, ""
	}

	// Fetching secrets from AWS Secrets Manager
	secretsMgr := secretsmanager.New(sess)

	for _, secretArn := range secretsArns {
		input := &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretArn.(string)),
		}

		result, err := secretsMgr.GetSecretValue(input)
		if err != nil {
			return "", fmt.Errorf("failed to get secret value: %s", err), ""
		}

		var secretVars map[string]string
		err = json.Unmarshal([]byte(*result.SecretString), &secretVars)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal secret JSON: %s", err), ""
		}

		for k, v := range secretVars {
			mergedVars[k] = v
		}
	}

	envFileContent := ""
	for k, v := range mergedVars {
		envFileContent += fmt.Sprintf("%s=%v\n", k, v)
	}

	envFileContent += mapToEnvFormat(envsMap)

	// Fetch secrets hash using the fetchSecrets function
	fetchedSecretsHash, err := fetchSecrets(secretsArns, sess, false, d.Get("track_actual_secrets").(bool))
	if err != nil {
		return "", fmt.Errorf("failed to get fetchedSecretsHash: %s", err), ""
	}

	logger.Debug("createEnvFileContent fetchedSecretsHash", "value", fetchedSecretsHash)

	return envFileContent, nil, fetchedSecretsHash
}

func isJSON(s string) bool {
    var js map[string]interface{}
    return json.Unmarshal([]byte(s), &js) == nil
}

func fetchSecrets(secretsArns []interface{}, sess *session.Session, arnsChanged bool, trackActualSecrets bool) (string, error) {
	if !trackActualSecrets && !arnsChanged && len(secretsArns) == 0 {
        logger.Debug("secrets_arns changed to empty list, skipping secrets fetching")
        return "", nil
    }

    svc := secretsmanager.New(sess)
    fetchedSecrets := make(map[string]string)

    for _, secretArn := range secretsArns {
        input := &secretsmanager.GetSecretValueInput{
            SecretId: aws.String(secretArn.(string)),
        }

        result, err := svc.GetSecretValue(input)
        if err != nil {
            return "", fmt.Errorf("failed to fetch secret: %s, %s", secretArn, err)
        }

        secretString := aws.StringValue(result.SecretString)
        if isJSON(secretString) {
            var secretVars map[string]string
            err = json.Unmarshal([]byte(secretString), &secretVars)
            if err != nil {
                return "", fmt.Errorf("failed to unmarshal secret JSON: %s", err)
            }
            for k, v := range secretVars {
                fetchedSecrets[k] = v
            }
        } else {
            fetchedSecrets[aws.StringValue(result.Name)] = secretString
        }
    }

    fetchedSecretsHash := computeSecretsHash(fetchedSecrets)
    return fetchedSecretsHash, nil
}
