package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type BashTool struct {
	cwd string
}

func NewBashTool() *BashTool {
	cwd, _ := os.Getwd()
	return &BashTool{
		cwd: cwd,
	}
}

func (t *BashTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "Bash",
		Description: "Executes a given bash command in a persistent shell session with optional timeout.",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The bash command to execute.",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Timeout in milliseconds (default 120000).",
				},
                "run_in_background": map[string]interface{}{
                    "type": "boolean",
                    "description": "Run the command in the background.",
                },
			},
			"required": []string{"command"},
		},
	}
}

func (t *BashTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	cmdStr, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("command argument is required and must be a string")
	}
    
    runInBackground, _ := args["run_in_background"].(bool)

    // Handle explicit CD commands to update internal state
    // This is a heuristic to simulate persistent CWD
    if strings.HasPrefix(strings.TrimSpace(cmdStr), "cd ") {
        path := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(cmdStr), "cd "))
        // clean up quotes
        path = strings.Trim(path, "\"'")
        
        // actually, checking if directory exists
        err := os.Chdir(path)
        if err == nil {
            t.cwd, _ = os.Getwd()
            return fmt.Sprintf("Changed directory to %s", t.cwd), nil
        }
    }

	// Create command
	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	cmd.Dir = t.cwd
    
    if runInBackground {
        id := GlobalShellManager.Start(cmd)
        return fmt.Sprintf("Started background process with ID %s. Use BashOutput tool to monitor.", id), nil
    }

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		return fmt.Sprintf("Error: %v\nOutput:\n%s", err, output), nil
	}

	if len(output) > 30000 {
		output = output[:30000] + "\n...[Output Truncated]..."
	}

	return output, nil
}
