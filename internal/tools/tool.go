package tools

import (
	"context"
	"encoding/json"

	"github.com/omnitrix-sh/core.sh/pkg/models"
)

// Tool is the interface that all tools must implement
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// ToModelTool converts a Tool to the models.Tool format
func ToModelTool(t Tool) models.Tool {
	return models.Tool{
		Type: "function",
		Function: models.ToolFunction{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		},
	}
}

// ParseArguments parses JSON arguments into a map
func ParseArguments(argsJSON string) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// GetStringArg safely gets a string argument
func GetStringArg(args map[string]interface{}, key string, defaultVal string) string {
	if val, ok := args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}

// GetIntArg safely gets an int argument
func GetIntArg(args map[string]interface{}, key string, defaultVal int) int {
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return defaultVal
}

// GetBoolArg safely gets a bool argument
func GetBoolArg(args map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := args[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultVal
}
