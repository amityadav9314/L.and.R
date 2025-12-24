package tools

import (
	"context"
	"fmt"
)

// Simple implements tool.Tool interface and provides a Call method
type Simple struct {
	NameVal string
	DescVal string
	Fn      func(args map[string]interface{}) (string, error)
}

func (t *Simple) Name() string {
	return t.NameVal
}

func (t *Simple) Description() string {
	return t.DescVal
}

func (t *Simple) IsLongRunning() bool {
	return false
}

// Call executes the tool. Used by llmagent via reflection or interface check (assumed)
func (t *Simple) Call(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	if t.Fn == nil {
		return nil, fmt.Errorf("tool function not implemented")
	}
	// Execute custom function
	result, err := t.Fn(args)
	if err != nil {
		return nil, err
	}
	// Return as map for ADK compatibility
	return map[string]interface{}{"result": result}, nil
}
