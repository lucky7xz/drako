# Shell Integration (CD on Exit)

**Important Note for `go install` Users:**
If you installed Drako via `go install`, the binary is likely located at `~/go/bin/drako` (or `$GOPATH/bin/drako`). Ensure `~/go/bin` is in your shell's `PATH`. If your shell cannot find `drako`, you may need to add this line to your config:
```bash
export PATH=$PATH:~/go/bin
```
You can also try to replace the line below

```bash
drako --cwd-file="$tmp"
```
with

```bash
~/go/bin/drako --cwd-file="$tmp"
```

in the function below. 

## Bash / Zsh

Add the following function to your `.bashrc` or `.zshrc`:

```bash
function x() {
    tmp="$(mktemp -t drako-cwd.XXXXXX)"
    drako --cwd-file="$tmp"
    cwd="$(cat "$tmp")"
    if [ -n "$cwd" ] && [ "$cwd" != "$PWD" ]; then
        cd "$cwd"
    fi
    rm -f "$tmp"
}
```

Drako can automatically change your shell's current working directory when you exit the application. This behaves similarly to tools like `ranger` or `yazi`. This feature works by writing the final directory to a temporary file, which your shell then reads and `cd`'s into.

Now you can use `x` to launch Drako. When you navigate to a directory inside Drako and exit (using `q` or `Ctrl+C`), your shell will be in that directory.

**Note:** This guide is for **Bash**. Zsh, fish windows etc. and require their own custom scripting styles not covered here.
