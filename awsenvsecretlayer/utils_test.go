package awsenvsecretlayer

import (
	"testing"
	"os"
	"bytes"
	"reflect"
	"archive/zip"
	"io/ioutil"
	"github.com/stretchr/testify/assert"

)

func TestProcessYamlConfig(t *testing.T) {
	yamlConfig := `
name: test
FOO: bar
BAR: baz
`
	expectedResult := map[string]string{
		"name": "test",
		"FOO":  "bar",
		"BAR":  "baz",
	}
	result, err := processYamlConfig(yamlConfig)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("unexpected result: got %v, want %v", result, expectedResult)
	}
}

func TestComputeSecretsHash(t *testing.T) {
	secrets := map[string]string{
		"FOO": "bar",
		"BAR": "baz",
	}
	expectedResult := "b439fdce7a45b7cc4cc21d8db9f07c30a8e41291f48d8fb21031e5711179d3f9"
	result := computeSecretsHash(secrets)
	if result != expectedResult {
		t.Errorf("unexpected result: got %v, want %v", result, expectedResult)
	}
}

func TestCreateZipFile(t *testing.T) {
	content := []byte("test content")
	licenseFiles := []string{"test_license.txt"}
	zipFilePath, err := CreateZipFile("envs.txt", content, licenseFiles)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer os.Remove(zipFilePath)
	_, err = os.Stat(zipFilePath)
	if os.IsNotExist(err) {
		t.Errorf("zip file not created: %s", err)
	}
}

func TestReadZipFile(t *testing.T) {
    content := []byte("test content")
    licenseFile := []byte("test license")
    zipFile, err := CreateZipFile("envs.txt", content, []string{"test_license.txt"})
    if err != nil {
        t.Fatalf("error creating zip file: %s", err)
    }

    defer os.Remove(zipFile)

    zipFileBytes, err := ReadZipFile(zipFile)
    if err != nil {
        t.Fatalf("error reading zip file: %s", err)
    }

    r, err := zip.NewReader(bytes.NewReader(zipFileBytes), int64(len(zipFileBytes)))
    if err != nil {
        t.Fatalf("error creating zip reader: %s", err)
    }

    foundContent := false
    foundLicenseFile := false
    for _, f := range r.File {
        if f.Name == "envs.txt" {
            foundContent = true
            rc, err := f.Open()
            if err != nil {
                t.Fatalf("error opening content file: %s", err)
            }
            defer rc.Close()
            actualContent, err := ioutil.ReadAll(rc)
            if err != nil {
                t.Fatalf("error reading content file: %s", err)
            }
            assert.Equal(t, content, actualContent)
        }
        if f.Name == "test_license.txt" {
            foundLicenseFile = true
            rc, err := f.Open()
            if err != nil {
                t.Fatalf("error opening license file: %s", err)
            }
            defer rc.Close()
            actualLicenseFile, err := ioutil.ReadAll(rc)
            if err != nil {
                t.Fatalf("error reading license file: %s", err)
            }
            assert.Equal(t, licenseFile, actualLicenseFile)
        }
    }
    assert.True(t, foundContent)
    assert.True(t, foundLicenseFile)
}
