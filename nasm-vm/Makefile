# DTRules NASM VM - Makefile
# Copyright 2024 Paul Snow
# Licensed under the Apache License, Version 2.0

# Assembler and compiler
NASM := nasm
CC := gcc

# Flags - include both root and src/include directories
NASM_FLAGS := -f elf64 -g -F dwarf -I. -I include/
CFLAGS := -Wall -Wextra -g -O2

# Directories
SRC_DIR := src
TEST_DIR := test
BUILD_DIR := build
OBJ_DIR := $(BUILD_DIR)/obj

# Source files - both root level and src/ directory
ROOT_ASM_SOURCES := $(wildcard *.asm)
SRC_ASM_SOURCES := $(wildcard $(SRC_DIR)/*.asm)
ASM_SOURCES := $(ROOT_ASM_SOURCES) $(SRC_ASM_SOURCES)

# Object files
ROOT_ASM_OBJECTS := $(patsubst %.asm,$(OBJ_DIR)/%.o,$(ROOT_ASM_SOURCES))
SRC_ASM_OBJECTS := $(patsubst $(SRC_DIR)/%.asm,$(OBJ_DIR)/src_%.o,$(SRC_ASM_SOURCES))
ASM_OBJECTS := $(ROOT_ASM_OBJECTS) $(SRC_ASM_OBJECTS)

# Test sources - support both test/ and tests/ directories
TEST_C_SOURCES := $(wildcard $(TEST_DIR)/*.c) $(wildcard tests/*.c)
TEST_C_OBJECTS := $(patsubst %.c,$(OBJ_DIR)/%.o,$(notdir $(TEST_C_SOURCES)))

# Output
VM_LIB := $(BUILD_DIR)/libdtrules_vm.a
TEST_BIN := $(BUILD_DIR)/test_vm

# Default target
.PHONY: all
all: dirs $(VM_LIB)

# Create directories
.PHONY: dirs
dirs:
	@mkdir -p $(BUILD_DIR) $(OBJ_DIR)

# Assemble root-level NASM sources
$(OBJ_DIR)/%.o: %.asm vm_constants.inc vm_state.inc | dirs
	$(NASM) $(NASM_FLAGS) -o $@ $<

# Assemble src/ directory NASM sources
$(OBJ_DIR)/src_%.o: $(SRC_DIR)/%.asm | dirs
	$(NASM) $(NASM_FLAGS) -o $@ $<

# Compile C test sources from test/
$(OBJ_DIR)/%.o: $(TEST_DIR)/%.c | dirs
	$(CC) $(CFLAGS) -I. -c -o $@ $<

# Compile C test sources from tests/
$(OBJ_DIR)/%.o: tests/%.c | dirs
	$(CC) $(CFLAGS) -I. -c -o $@ $<

# Create static library
$(VM_LIB): $(ASM_OBJECTS) | dirs
	ar rcs $@ $^

# Build test binary
$(TEST_BIN): $(TEST_C_OBJECTS) $(VM_LIB) | dirs
	$(CC) $(CFLAGS) -o $@ $(TEST_C_OBJECTS) $(VM_LIB)

# Run tests
.PHONY: test
test: $(TEST_BIN)
	./$(TEST_BIN)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# Show assembly listing
%.lst: %.asm
	$(NASM) $(NASM_FLAGS) -l $@ $<

# Show object files (for debugging)
.PHONY: show
show:
	@echo "Root ASM sources: $(ROOT_ASM_SOURCES)"
	@echo "Src ASM sources: $(SRC_ASM_SOURCES)"
	@echo "All ASM objects: $(ASM_OBJECTS)"
	@echo "Test sources: $(TEST_C_SOURCES)"
	@echo "Test objects: $(TEST_C_OBJECTS)"

# Help
.PHONY: help
help:
	@echo "DTRules NASM VM Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build the VM library (default)"
	@echo "  test    - Build and run tests"
	@echo "  clean   - Remove build artifacts"
	@echo "  show    - Show source/object files"
	@echo "  help    - Show this help"
