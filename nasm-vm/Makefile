# DTRules NASM VM - Makefile
# Copyright 2024 Paul Snow
# Licensed under the Apache License, Version 2.0

# Assembler and compiler
NASM := nasm
CC := gcc

# Flags
NASM_FLAGS := -f elf64 -g -F dwarf -I include/
CFLAGS := -Wall -Wextra -g -O2

# Directories
SRC_DIR := src
TEST_DIR := tests
BUILD_DIR := build
OBJ_DIR := $(BUILD_DIR)/obj

# Source files
ASM_SOURCES := $(wildcard $(SRC_DIR)/*.asm)
ASM_OBJECTS := $(patsubst $(SRC_DIR)/%.asm,$(OBJ_DIR)/%.o,$(ASM_SOURCES))

# Test sources
TEST_SOURCES := $(wildcard $(TEST_DIR)/*.c)
TEST_OBJECTS := $(patsubst $(TEST_DIR)/%.c,$(OBJ_DIR)/%.o,$(TEST_SOURCES))

# Output
VM_LIB := $(BUILD_DIR)/libdtrules_vm.a
TEST_BIN := $(BUILD_DIR)/test_vm

# Default target
.PHONY: all
all: $(VM_LIB) $(TEST_BIN)

# Create directories
$(OBJ_DIR):
	mkdir -p $(OBJ_DIR)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Assemble NASM sources
$(OBJ_DIR)/%.o: $(SRC_DIR)/%.asm | $(OBJ_DIR)
	$(NASM) $(NASM_FLAGS) -o $@ $<

# Compile C test sources
$(OBJ_DIR)/%.o: $(TEST_DIR)/%.c | $(OBJ_DIR)
	$(CC) $(CFLAGS) -c -o $@ $<

# Create static library
$(VM_LIB): $(ASM_OBJECTS) | $(BUILD_DIR)
	ar rcs $@ $^

# Build test binary
$(TEST_BIN): $(TEST_OBJECTS) $(VM_LIB) | $(BUILD_DIR)
	$(CC) $(CFLAGS) -o $@ $(TEST_OBJECTS) $(VM_LIB)

# Run tests
.PHONY: test
test: $(TEST_BIN)
	./$(TEST_BIN)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# Show object files (for debugging)
.PHONY: show
show:
	@echo "ASM sources: $(ASM_SOURCES)"
	@echo "ASM objects: $(ASM_OBJECTS)"
	@echo "Test sources: $(TEST_SOURCES)"
	@echo "Test objects: $(TEST_OBJECTS)"

# Help
.PHONY: help
help:
	@echo "DTRules NASM VM Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build library and tests (default)"
	@echo "  test    - Build and run tests"
	@echo "  clean   - Remove build artifacts"
	@echo "  show    - Show source/object files"
	@echo "  help    - Show this help"
