package awsenvsecretlayer

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func processYamlConfig(yamlConfig string) (map[string]string, error) {
	result := make(map[string]string)

	if yamlConfig == "" {
		return result, nil
	}

	var yamlData map[string]interface{}
	err := yaml.Unmarshal([]byte(yamlConfig), &yamlData)
	if err != nil {
		return nil, err
	}

	for k, v := range yamlData {
		flattenedMap := flatten("", k, v)
		for fk, fv := range flattenedMap {
			result[fk] = fv
		}
	}

	return result, nil
}

func flatten(prefix string, key string, value interface{}) map[string]string {
	result := make(map[string]string)

	switch v := value.(type) {
	case map[string]interface{}:
		for k, subv := range v {
			newKey := key + "_" + k
			if prefix != "" {
				newKey = prefix + "_" + newKey
			}
			subMap := flatten(key, newKey, subv)
			for k, v := range subMap {
				result[k] = v
			}
		}
	case string:
		if prefix != "" {
			key = prefix + "_" + key
		}
		result[key] = v
	}

	return result
}

func CreateZipFile(fileName string, content []byte, licenseFiles []string) (string, error) {
	tmpZipFile, err := os.CreateTemp("", "zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp zip file: %s", err)
	}
	defer tmpZipFile.Close()

	zipWriter := zip.NewWriter(tmpZipFile)
	defer zipWriter.Close()

	zipFile, err := zipWriter.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file entry: %s", err)
	}

	_, err = zipFile.Write(content)
	if err != nil {
		return "", fmt.Errorf("failed to write content to zip file entry: %s", err)
	}

	for _, licenseFile := range licenseFiles {
		licenseContent, err := os.ReadFile(licenseFile)
		if err != nil {
			return "", fmt.Errorf("failed to read license file: %s", err)
		}

		zipLicenseFile, err := zipWriter.Create(filepath.Base(licenseFile))
		if err != nil {
			return "", fmt.Errorf("failed to create license file entry in zip: %s", err)
		}

		_, err = zipLicenseFile.Write(licenseContent)
		if err != nil {
			return "", fmt.Errorf("failed to write license content to zip file entry: %s", err)
		}
	}

	err = zipWriter.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close zip writer: %s", err)
	}

	return tmpZipFile.Name(), nil
}

func ReadZipFile(zipFilePath string) ([]byte, error) {
	file, err := os.Open(zipFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %s", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat zip file: %s", err)
	}

	zipFileBytes := make([]byte, fileInfo.Size())
	_, err = file.Read(zipFileBytes)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read zip file: %s", err)
	}

	return zipFileBytes, nil
}

func jsonEncode(data map[string]string) string {
	encoded, _ := json.Marshal(data)
	return string(encoded)
}

func computeSecretsHash(secrets map[string]string) string {
	sortedKeys := make([]string, 0, len(secrets))
	for k := range secrets {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	h := sha256.New()
	for _, k := range sortedKeys {
		h.Write([]byte(k))
		h.Write([]byte(secrets[k]))
	}

	return hex.EncodeToString(h.Sum(nil))
}
