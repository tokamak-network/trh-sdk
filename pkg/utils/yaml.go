package utils

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func UpdateYAMLField(filePath string, fieldPath string, newValue interface{}) error {
	// Read the YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal YAML into a generic map
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Update the nested field
	keys := strings.Split(fieldPath, ".")
	if err := setNestedValue(yamlData, keys, newValue); err != nil {
		return fmt.Errorf("failed to update YAML field: %w", err)
	}

	// Marshal the updated YAML data
	updatedData, err := yaml.Marshal(yamlData)
	if err != nil {
		return fmt.Errorf("failed to marshal updated YAML: %w", err)
	}

	// Write back to the file
	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated YAML file: %w", err)
	}

	return nil
}

// setNestedValue sets a value in a nested YAML structure
func setNestedValue(data map[string]interface{}, keys []string, value interface{}) error {
	if len(keys) == 0 {
		return fmt.Errorf("invalid field path")
	}

	// Traverse to the last key
	current := data
	for i, key := range keys {
		if i == len(keys)-1 {
			// Set the value at the last key
			current[key] = value
			return nil
		}

		// Move to the next nested map
		next, exists := current[key]
		if !exists {
			// Create nested map if it doesn't exist
			next = make(map[string]interface{})
			current[key] = next
		}

		// Type assertion to ensure it's a map
		nestedMap, ok := next.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot set value, %s is not a map", key)
		}

		current = nestedMap
	}

	return nil
}
