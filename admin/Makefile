.PHONY: build clean install

# Build ralphy-admin binary and place it in bin/
build:
	go build -o ../bin/ralphy-admin .

# Build and install to dotfiles (where the symlink points)
install: build
	@if [ -d "$(HOME)/.dotfiles/ralphy/bin" ]; then \
		cp ../bin/ralphy-admin "$(HOME)/.dotfiles/ralphy/bin/ralphy-admin"; \
		echo "Installed to ~/.dotfiles/ralphy/bin/ralphy-admin"; \
	fi

# Remove the built binary
clean:
	rm -f ../bin/ralphy-admin
