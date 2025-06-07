# Directory Installation Examples

This document shows how pyhub-installer handles different installation scenarios with the new directory-based approach.

## Installation Directory Structure

### Windows
```
C:\Users\{username}\AppData\Local\Programs\
├── pyhub-mcptools\           # Program directory
│   ├── pyhub-mcptools.exe    # Main executable
│   ├── config.json           # Configuration
│   └── resources\            # Additional resources
└── another-tool\
    └── another-tool.exe
```

**PATH**: Each program directory is added to PATH individually:
- `C:\Users\{username}\AppData\Local\Programs\pyhub-mcptools`
- `C:\Users\{username}\AppData\Local\Programs\another-tool`

### macOS/Linux
```
~/.local/
├── share/                    # Program installations
│   ├── pyhub-mcptools/      # Program directory
│   │   ├── pyhub-mcptools   # Main executable
│   │   ├── config.json      # Configuration
│   │   └── resources/       # Additional resources
│   └── another-tool/
│       └── another-tool
└── bin/                     # Symbolic links
    ├── pyhub-mcptools -> ~/.local/share/pyhub-mcptools/pyhub-mcptools
    └── another-tool -> ~/.local/share/another-tool/another-tool
```

**PATH**: Only `~/.local/bin` needs to be in PATH

## Example 1: Single Executable File

**Command:**
```bash
pyhub-installer install github:pyhub-kr/simple-cli
```

**Windows Result:**
```
C:\Users\alice\AppData\Local\Programs\simple-cli\
└── simple-cli.exe

PATH += C:\Users\alice\AppData\Local\Programs\simple-cli
```

**macOS/Linux Result:**
```
~/.local/share/simple-cli/
└── simple-cli

~/.local/bin/
└── simple-cli -> ~/.local/share/simple-cli/simple-cli
```

## Example 2: Complex Application with Resources

**Command:**
```bash
pyhub-installer install github:pyhub-kr/pyhub-mcptools
```

**Archive Contents:**
```
pyhub-mcptools-v1.0.0/
├── pyhub-mcptools      # Main executable
├── pyhub-mcp-helper    # Helper executable
├── config.json
├── templates/
│   ├── default.tmpl
│   └── custom.tmpl
└── docs/
    └── README.md
```

**Windows Result:**
```
C:\Users\alice\AppData\Local\Programs\pyhub-mcptools\
├── pyhub-mcptools.exe
├── pyhub-mcp-helper.exe
├── config.json
├── templates\
│   ├── default.tmpl
│   └── custom.tmpl
└── docs\
    └── README.md

PATH += C:\Users\alice\AppData\Local\Programs\pyhub-mcptools
```

**macOS/Linux Result:**
```
~/.local/share/pyhub-mcptools/
├── pyhub-mcptools
├── pyhub-mcp-helper
├── config.json
├── templates/
│   ├── default.tmpl
│   └── custom.tmpl
└── docs/
    └── README.md

~/.local/bin/
├── pyhub-mcptools -> ~/.local/share/pyhub-mcptools/pyhub-mcptools
└── pyhub-mcp-helper -> ~/.local/share/pyhub-mcptools/pyhub-mcp-helper
```

## Example 3: Multiple Executables in Subdirectories

**Archive Contents:**
```
my-toolkit-v2.0/
├── bin/
│   ├── tool1
│   └── tool2
├── lib/
│   └── shared.so
└── README.md
```

**Windows Result:**
```
C:\Users\alice\AppData\Local\Programs\my-toolkit\
├── bin\
│   ├── tool1.exe
│   └── tool2.exe
├── lib\
│   └── shared.dll
└── README.md

PATH += C:\Users\alice\AppData\Local\Programs\my-toolkit\bin
```

**macOS/Linux Result:**
```
~/.local/share/my-toolkit/
├── bin/
│   ├── tool1
│   └── tool2
├── lib/
│   └── shared.so
└── README.md

~/.local/bin/
├── tool1 -> ~/.local/share/my-toolkit/bin/tool1
└── tool2 -> ~/.local/share/my-toolkit/bin/tool2
```

## PATH Management

### First Installation
```
$ pyhub-installer install github:pyhub-kr/pyhub-mcptools

Installing to: C:\Users\alice\AppData\Local\Programs\pyhub-mcptools
✓ Installation complete

⚠️  C:\Users\alice\AppData\Local\Programs\pyhub-mcptools is not in your PATH.
Would you like to add it automatically? [Y/n]: Y

✓ PATH updated successfully.
Please restart your terminal or run:
  refreshenv  (Windows Command Prompt)
  $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")  (PowerShell)
```

### Subsequent Installations
```
$ pyhub-installer install github:pyhub-kr/another-tool

Installing to: ~/.local/share/another-tool
✓ Created symlink: ~/.local/bin/another-tool -> ~/.local/share/another-tool/another-tool
✓ Installation complete

~/.local/bin is already in your PATH ✓
```

## Avoiding Problematic Paths

The installer will NOT use these paths:
- ❌ `C:\Users\alice\AppData\Local\Programs\cursor\resources\app\bin`
- ❌ `C:\Python312\Scripts`
- ❌ `~/.vscode/extensions/something/bin`
- ❌ `/usr/local/opt/node@18/bin`

Instead, it uses standard OS locations:
- ✅ `C:\Users\alice\AppData\Local\Programs\{program-name}`
- ✅ `~/.local/share/{program-name}`
- ✅ `~/.local/bin` (for symlinks)