# acme-claude

Go wrapper program around the Claude CLI that integrates with the Plan 9 acme editor for quasi-interactive Claude usage with conversation history and permission management.

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

This will build and install the Claude binary to `$HOME/bin`.

## Usage

### Basic Workflow

1. **Start Claude**:
   Type `Claude` in the tag of some window (e.g., a project working directory).
   Middle-click on Claude to run the program.

   This creates the +Claude window for that working directory. This is a quasi-interactive
   chat interface.

2. **Send a prompt**:
   - Type your question/request directly in the `+Claude` window
   - Middle-click the "Send" tag to send your message
   - The conversation flows like a terminal chat session with clear USER:/CLAUDE: markers
   - Debug output appears in real-time in the `+ClaudeTrace` window

3. **Continue the conversation**:
   - Type your next message below the conversation in the `+Claude` window
   - Middle-click "Send" again
   - Full conversation context is maintained automatically

4. **Reset conversation**:
   - Middle-click "Reset" in the `+Claude` window's tag bar to clear the context
     for the current working directory.

5. **Manage permissions**:
   - Middle-click "Permissions" in the `+Claude` window's tag bar

### Permission Management

The Claude program provides fine-grained control over what tools Claude can use in each directory.

#### Accessing Permission Management

- **Open permissions interface**:
  - Middle-click "Permissions" in the `+Claude` window's tag bar
  - This opens the `+Claude-Permissions` window

#### Interactive Permission Editing

1. Middle-click "Permissions" to open the `+Claude-Permissions` window
2. Use the commands in the permission window's tag bar:
   - Middle-click "Show" to display current permissions
   - Middle-click "Edit" to see all available tools for editing
   - Edit the list by adding prefixes:
     - `+` = Allow tool
     - `-` = Deny tool
     - `~` = Remove explicit permission (back to default)
   - Middle-click "Save" to apply your changes

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
acme-claude/
├── main.go             # Unified Claude program entry point
├── internal/
│   ├── acme/          # Acme interactions
│   └── context/       # Context and settings management
└── mkfile             # Plan 9 makefile
```

### Context Management

Each directory gets its own context stored in `~/.claude-acme/$directory_hash/`:
- `context.json` - Conversation history
- `settings.json` - Tool permissions and configuration

### Windows

- `$pwd/+Claude` - Interactive chat interface (input and conversation history combined)
- `$pwd/+ClaudeTrace` - Debug output window showing Claude's internal operations
- `$pwd/+Claude-Permissions` - Permission management interface

## Examples

### Development Workflow
```bash
# Start in your project directory
cd ~/myproject

# Create conversation windows
Claude

# Allow development tools
# - Middle-click "Permissions" in +Claude window
# - Middle-click "Edit", then modify list to: +Bash(go:*) +Bash(git:*) +Read +Write +Edit
# - Middle-click "Save" to apply

# Now Claude can run tests, build, and commit changes
```

### Research Workflow
```bash
# In research directory
cd ~/research
Claude

# Allow web access but restrict file operations
# - Middle-click "Permissions" in +Claude window
# - Middle-click "Edit", then modify list to: +WebSearch +WebFetch +Read +Edit(/home/user/research/*)
# - Middle-click "Save" to apply
```

### Safe Exploration
```bash
# New directory
cd ~/experiments
Claude

# Very restricted permissions
# - Middle-click "Permissions" in +Claude window
# - Middle-click "Edit", then modify list to: +Read +Bash(ls:*) +Bash(cat:*)
# - Middle-click "Save" to apply

# Claude can only read files and do basic listing
```

## Troubleshooting

### Claude not respecting permissions
- Check that permissions show correctly by middle-clicking "Permissions"
- Verify that both allowed and denied tools are being set
- Try middle-clicking "Reset" to clear any cached context

### Windows not appearing
- Ensure acme is running
- Check that you're in the correct working directory
- Try running `Claude` again to recreate windows

### Permission changes not taking effect
- Make sure to middle-click "Save" after editing the permission list
- Check the `+Claude-Permissions` window for error messages after clicking "Save"
- Verify the permission format (correct prefixes: `+`, `-`, `~`)

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.