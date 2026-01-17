.PHONY: generate lint clean

# Generate Go code from proto files
generate:
	@echo "Generating protobuf code..."
	buf generate
	@echo "Done!"

# Lint proto files
lint:
	@echo "Linting proto files..."
	buf lint
	@echo "Done!"

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf gen/
	@echo "Done!"

# Format proto files
format:
	@echo "Formatting proto files..."
	buf format -w
	@echo "Done!"
