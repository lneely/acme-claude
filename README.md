# claude-acme

A Go wrapper program around the Claude CLI that integrates with the Plan 9 acme editor for non-interactive Claude usage with conversation history and permission management.

## Overview

claude-acme provides three main components that work together to create a seamless Claude experience within acme:

- **Claude** - Creates prompt and chat history windows
- **Prompt** - Processes user input and streams Claude responses  
- **Claude-Reset** - Clears conversation context
- **Claude-Permissions** - Manages tool permissions per directory

## Features

- **Directory-based conversations** - Each working directory maintains its own chat history and context
- **Real-time streaming** - Claude responses stream directly to the chat window as they're generated
- **Persistent context** - Full conversation history is maintained and sent with each new prompt
- **Fine-grained permissions** - Control which tools Claude can use on a per-directory basis
- **Native acme integration** - All interaction happens through acme windows with clickable commands

## Installation

### Prerequisites

- Go 1.21 or later
- Plan 9 from User Space (for acme editor)
- Claude CLI installed and authenticated

### Build and Install

```bash
mk install
```

This will build and install all four binaries to `$HOME/bin`.

## Usage

### Basic Workflow

1. **Start a conversation**:
   ```bash
   Claude
   ```
   This creates two windows in your current directory:
   - `$pwd/+Prompt` - Where you type your prompts
   - `$pwd/+Claude` - Chat history with full conversation

2. **Send a prompt**:
   - Type your question/request in the `+Prompt` window
   - Click "Prompt" in the window's tag bar
   - Your input is immediately moved to `+Claude` and Claude's response streams in real-time

3. **Continue the conversation**:
   - Type follow-up questions in `+Prompt`
   - Click "Prompt" again
   - Full conversation context is maintained automatically

4. **Reset conversation** (optional):
   ```bash
   Claude-Reset
   ```

### Permission Management

Claude-Permissions provides fine-grained control over what tools Claude can use in each directory.

#### Basic Permission Commands

- **View current permissions**:
  ```bash
  Claude-Permissions
  ```

- **See available tools to grant**:
  ```bash
  Claude-Permissions ?
  ```

#### Interactive Permission Editing

1. Run `Claude-Permissions` to see current permissions in the `+Claude-Permissions` window
2. Edit the list by adding prefixes:
   - `+` = Allow tool
   - `-` = Deny tool
   - `~` = Remove explicit permission (back to default)
3. Select the modified text and run:
   ```bash
   Claude-Permissions 'selected_text'
   ```

#### Example Permission Session

```bash
# View current permissions
Claude-Permissions
```

In the `+Claude-Permissions` window:
```
# Active permissions for: /home/user/myproject
# Permission Mode: acceptEdits

+ Bash
+ Read
+ Write
+ Edit
```

To grant web access, edit to:
```
+ Bash
+ Read  
+ Write
+ Edit
+ WebSearch
+ WebFetch
```

Select the text and run `Claude-Permissions 'selected_text'`.

### Available Tools

#### Core Tools
- **Read** - Read files from filesystem
- **Write** - Write new files
- **Edit** - Edit existing files
- **MultiEdit** - Make multiple edits to a file
- **NotebookEdit** - Edit Jupyter notebooks
- **Glob** - Find files by pattern
- **Grep** - Search file contents
- **Bash** - Execute shell commands
- **WebSearch** - Search the web
- **WebFetch** - Fetch web pages
- **Task** - Launch specialized agents
- **TodoWrite** - Manage task lists

#### Granular Bash Permissions
Control specific shell commands:
- `Bash(git:*)` - Only git commands
- `Bash(go:*)` - Only go commands (build, test, etc.)
- `Bash(npm:*)` - Only npm commands
- `Bash(make:*)` - Only make commands
- `Bash(ls:*)`, `Bash(cp:*)`, etc.

#### Path-Restricted Tools
Limit file operations to specific directories:
- `Edit(/path/to/dir/*)` - Edit only in specific directory
- `Read(/path/to/dir/*)` - Read only from specific directory
- `Write(/path/to/dir/*)` - Write only to specific directory

### Permission Modes

- **acceptEdits** - Allow file edits with confirmation
- **bypassPermissions** - Skip all permission checks
- **default** - Use Claude's default behavior
- **plan** - Planning mode

## Architecture

### Directory Structure
```
claude-acme/
├── cmd/
│   ├── Claude/           # Window creation command
│   ├── Prompt/           # Main processing command  
│   ├── Claude-Reset/     # Context reset command
│   └── Claude-Permissions/ # Permission management
├── internal/
│   ├── acme/            # Acme 9p protocol wrapper
│   └── context/         # Context and settings management
└── mkfile               # Plan 9 makefile
```

### Context Management

Each directory gets its own context stored in `~/.claude-acme/$directory_hash/`:
- `context.json` - Conversation history
- `settings.json` - Tool permissions and configuration

### Window Management

- `$pwd/+Prompt` - Input window (cleared after each use)
- `$pwd/+Claude` - Persistent chat history
- `$pwd/+Claude-Permissions` - Permission management interface

## Examples

### Development Workflow
```bash
# Start in your project directory
cd ~/myproject

# Create conversation windows
Claude

# Allow development tools
Claude-Permissions
# Edit to allow: +Bash(go:*) +Bash(git:*) +Read +Write +Edit

# Now Claude can run tests, build, and commit changes
```

### Research Workflow  
```bash
# In research directory
cd ~/research

# Allow web access but restrict file operations
Claude-Permissions
# Edit to allow: +WebSearch +WebFetch +Read +Edit(/home/user/research/*)
```

### Safe Exploration
```bash
# New directory
cd ~/experiments

# Very restricted permissions
Claude-Permissions
# Edit to allow: +Read +Bash(ls:*) +Bash(cat:*)

# Claude can only read files and do basic listing
```

## Troubleshooting

### Claude not respecting permissions
- Check that permissions show correctly with `Claude-Permissions`
- Verify that both allowed and denied tools are being set
- Try `Claude-Reset` to clear any cached context

### Windows not appearing
- Ensure acme is running
- Check that you're in the correct working directory
- Try running `Claude` again to recreate windows

### Permission changes not taking effect
- Make sure to select the entire modified text when running `Claude-Permissions 'text'`
- Check the `+Claude-Permissions` window for error messages
- Verify the permission format (correct prefixes: `+`, `-`, `~`)

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.