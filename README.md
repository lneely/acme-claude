# acme-claude: Claude in Acme!

The `Claude` program provides a user interface for Acme users to interact with Claude without having to open an interactive terminal.

## Features

**Chat**

- Interactive graphical chat session with Claude
- `Look` and `Execute` (e.g., plumbing, commands) directly from the chat window (right-click)

**Continuity**

- Uses Claude's built-in session management (per-directory)
- Load previous sessions from Acme
- Continues the previous conversation by default (`claude --continue`)

**Permissions**

- Manage permissions per-directory from Acme
- Simple `+` (allow), `-` (deny) , and `~` (remove explicit) permission management
- `Show` to see examples and add new permissions
- "Secure by default" (`Read` only)

## Usage

Start the program by typing `Claude` in the tag line of any window, and middle-clicking it. The working directory for Claude is "inherited" from that window.

![Starting Claude](./img/demo01.png)

To chat with Claude, simply type in the area below the `USER: [Send]` text. To send your message to Claude, middle-click the `[Send]` "button" next to `USER:`, or on `Send` in the `+Claude` window tag line.

![Alt text](./img/demo02.png)

Claude's response will be streamed directly into the `+Claude` chat window.

![Alt text](./img/demo03.png)

You can disable trace logs by closing the `+ClaudeTrace` window (but I don't recommend it).

![Alt text](./img/demo04.png)

Manage permissions by middle-clicking on `Permissions` in the `+Claude` window tag line. This opens the `+Claude-Permissions` window, which lists the *active* permissions for Claude in this directory. If none exist, it defaults to `Read`.

![Alt text](./img/demo05.png)

Click `Edit` to add new permissions. Type a `+` next to all the permissions you want to add, and middle-click `Save`.

![Alt text](./img/demo06.png)

Now the permissions are updated. From here you may `+ <PermissionName>` to add another, or replace the `+` with a `-` to explicitly deny the tool, or `~` to revoke the permission for the active tool.

![Alt text](./img/demo07.png)

By default Claude continues the most recent chat session, however it is possible to load a different session. Middle-click `Sessions` in the `+Claude` window tag line to show the `+Claude-Sessions` window. This shows a brief message in the `+ClaudeTrace` window that the session loaded, and closes the session list.

![Alt text](./img/demo08.png)

This program uses Claude's built-in session management, and is therefore capable of continuing previous discussions. The images below demonstrate this in practice.

![Alt text](./img/demo09.png)
![Alt text](./img/demo10.png)
![Alt text](./img/demo11.png)

## Enjoy!

I built this program for my own use, so I don't promise to maintain this long-term. But, maybe you find it useful too, and if so, enjoy and feel free to contribute.
