# Running Claude in a FreeBSD Jail

This document explains how to set up `acme-claude` to run the Claude CLI inside a FreeBSD jail.

## Configuration Guide

Do it how you want, below are just possibly-helpful hints.

### jail.conf

1. **Path Mapping**: The `exec.prestart` line mounts your host directories into the jail
2. **User IDs**: It's convenient if the jail user and host user share the same UID.

```
claude {
  path = "/jails/claude";
  host.hostname = "jail.claude";
  exec.prestart = "/sbin/mount_nullfs /path/to/src /path/to/jails/claude/home/youruser/src";
  exec.start = "/bin/sh /etc/rc";
  exec.stop = "/bin/sh /etc/rc.shutdown";
  exec.clean;
  mount.devfs;

  # do what you want otherwise (e.g., networking)
}
```

### Wrapper Script

Use the provided `claude.jailed` script to execute Claude commands in the jail. Configure these variables at the top of the script to match your configuration.

- `JAIL_USER`: Username inside the jail (default: "claudeuser")
- `JAIL_NAME`: Jail name from jail.conf (default: "claude")
- `SHARED_DIRS`: Space-separated list of shared directories (default: "src prj")

You can symlink or copy `claude.jailed` to something like $HOME/bin/claude, assuming $HOME/bin is in your path.


## Why?

- Nifty!
- Sandboxed execution
- Network and filesystem access control
- YOLO! if you want
