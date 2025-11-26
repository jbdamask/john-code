package tools

import (
	"context"
	"fmt"
)

// BashOutputTool
type BashOutputTool struct{}

func (t *BashOutputTool) Definition() ToolDefinition {
    return ToolDefinition{
        Name: "BashOutput",
        Description: "Retrieve output from a background bash process.",
        Schema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "shell_id": map[string]interface{}{
                    "type": "string",
                },
            },
            "required": []string{"shell_id"},
        },
    }
}

func (t *BashOutputTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    id, ok := args["shell_id"].(string)
    if !ok {
        return "", fmt.Errorf("shell_id required")
    }

    output, done, err := GlobalShellManager.GetOutput(id)
    
    status := "running"
    if done {
        status = "finished"
    }
    if err != nil {
        status = fmt.Sprintf("error: %v", err)
    }

    return fmt.Sprintf("Shell ID: %s\nStatus: %s\nOutput:\n%s", id, status, output), nil
}

// KillShellTool
type KillShellTool struct{}

func (t *KillShellTool) Definition() ToolDefinition {
    return ToolDefinition{
        Name: "KillShell",
        Description: "Kill a background bash process.",
        Schema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "shell_id": map[string]interface{}{
                    "type": "string",
                },
            },
            "required": []string{"shell_id"},
        },
    }
}

func (t *KillShellTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    id, ok := args["shell_id"].(string)
    if !ok {
        return "", fmt.Errorf("shell_id required")
    }

    err := GlobalShellManager.Kill(id)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("Successfully killed shell %s", id), nil
}
