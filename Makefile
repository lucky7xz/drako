.PHONY: install clean fmt test

# ===================================================================
# This new install target provides a simple, one-command installation for drako.
# The main Go program handles creating a default config if one doesn't exist.
# ===================================================================
install:
	@echo "Installing drako..."
	go install ./drako
	@echo "Installation complete."
	@echo "Run 'drako' to start the application."
	@echo "A default config will be created at ~/.config/drako/config.toml on first run if it doesn't exist."
# ===================================================================



clean:
	@echo "Cleaning build artifacts (none required for go install)..."
	@echo "Clean"

fmt:
	go fmt ./...

test:
	go test ./...
