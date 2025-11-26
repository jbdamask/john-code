# John Code

John Code is a CLI tool that helps you with software engineering tasks, inspired by Claude Code.

## Prerequisites

-   Go 1.20+ installed.
-   An Anthropic API Key (`ANTHROPIC_API_KEY`).
-   `ripgrep` installed (for the `Grep` tool).

## Installation

Clone the repository:

```bash
git clone https://github.com/jbdamask/john-code.git
cd john-code
```

## Build and Run

1.  **Build the binary:**

    ```bash
    go build -o john ./cmd/john
    ```

2.  **Run the tool:**

    You must set your Anthropic API key in the environment variable `ANTHROPIC_API_KEY`.

    ```bash
    export ANTHROPIC_API_KEY="your-api-key-here"
    ./john
    ```

## Usage

Once running, type your request at the prompt.

-   Type `exit` or `quit` to stop.
-   John Code can execute bash commands, read/write/edit files, and manage todo lists to help you complete tasks.
