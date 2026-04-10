# Default target executed when you just type 'make'
all: build

# Install Node.js dependencies using npm
# This reads the package.json file and downloads Vite, React, etc.
install:
	npm install

# Build the single HTML file using Vite
# The 'build' target depends on 'install', so 'make' will automatically run 'npm install' first
build: install
	npm run build

# Optional: Clean up generated files and downloaded dependencies
# You can run this by typing 'make clean'
clean:
	rm -rf dist node_modules
