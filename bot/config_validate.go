package bot

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// validate_yaml validates the YAML data against the appropriate configuration struct.
// - filePath: The path to the YAML file being validated.
// - yamldata: The YAML data to validate.
func validate_yaml(filePath string, yamldata []byte) error {
	// Determine the file type based on the directory structure.
	fileType := getFileType(filePath)

	// First, check if the YAML is valid by unmarshalling into a yaml.Node.
	var rootNode yaml.Node
	err := yaml.Unmarshal(yamldata, &rootNode)
	if err != nil {
		return fmt.Errorf("invalid YAML syntax in '%s': %v", filePath, err)
	}

	// Next, perform fix-ups of "Append" prefixes and remove free-form sections.
	if err := processNode(fileType, &rootNode); err != nil {
		return fmt.Errorf("error processing YAML nodes in '%s': %v", filePath, err)
	}

	// Marshal the modified YAML node back into bytes.
	modifiedYAML, err := yaml.Marshal(&rootNode)
	if err != nil {
		return fmt.Errorf("failed to marshal modified YAML for '%s': %v", filePath, err)
	}

	// Unmarshal the modified YAML into the appropriate struct with KnownFields enabled.
	var targetStruct interface{}
	switch fileType {
	case "robot":
		targetStruct = &ConfigLoader{}
	case "plugin":
		targetStruct = &Plugin{}
	case "job":
		targetStruct = &Job{}
	default:
		return fmt.Errorf("unknown file type for '%s'", filePath)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(modifiedYAML))
	decoder.KnownFields(true)
	if err := decoder.Decode(targetStruct); err != nil {
		return fmt.Errorf("validation error in '%s': %v", filePath, err)
	}

	// Validation successful.
	return nil
}

// getFileType determines the configuration type based on the file path.
func getFileType(filePath string) string {
	dir := filepath.Dir(filePath)
	lastDir := filepath.Base(dir)

	switch lastDir {
	case "plugins":
		return "plugin"
	case "jobs":
		return "job"
	default:
		return "robot"
	}
}

// processNode recursively processes the YAML node to fix "Append" prefixes and remove free-form sections.
func processNode(fileType string, node *yaml.Node) error {
	// Define free-form sections to exclude based on file type.
	freeFormSections := map[string][]string{
		"robot":  {"ProtocolConfig", "BrainConfig", "HistoryConfig"},
		"plugin": {"Config"},
		"job":    {"Config"},
	}

	freeFormKeys := freeFormSections[fileType]

	if node.Kind == yaml.MappingNode {
		// Process key-value pairs in mapping nodes.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Fix-up "Append" prefixes.
			if strings.HasPrefix(keyNode.Value, "Append") {
				originalKey := keyNode.Value
				keyNode.Value = strings.TrimPrefix(keyNode.Value, "Append")
				keyNode.Value = strings.TrimSpace(keyNode.Value)
				// Add a comment indicating the original key.
				if keyNode.HeadComment == "" {
					keyNode.HeadComment = fmt.Sprintf("(was: %s)", originalKey)
				}
			}

			// Remove free-form sections.
			removeKey := false
			for _, freeKey := range freeFormKeys {
				if keyNode.Value == freeKey {
					// Mark this key-value pair for removal.
					removeKey = true
					break
				}
			}

			if removeKey {
				// Remove this key-value pair from the Content slice.
				node.Content = append(node.Content[:i], node.Content[i+2:]...)
				// Adjust index to account for removed elements.
				i -= 2
				continue
			}

			// Recursively process the value node.
			if err := processNode(fileType, valueNode); err != nil {
				return err
			}
		}
	} else if node.Kind == yaml.SequenceNode || node.Kind == yaml.DocumentNode {
		// Process child nodes.
		for _, childNode := range node.Content {
			if err := processNode(fileType, childNode); err != nil {
				return err
			}
		}
	}
	// For scalar nodes and others, no processing is needed.
	return nil
}
