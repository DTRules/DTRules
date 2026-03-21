# DTRules NASM VM Makefile
# Copyright 2024 Paul Snow
# Licensed under Apache License 2.0

# Assembler and flags
NASM = nasm
NASMFLAGS = -f elf64 -g -F dwarf -I.

# C compiler for test harness
CC = gcc
CFLAGS = -Wall -Wextra -g -O2 -I.

# Directories
BUILD_DIR = build
SRC_DIR = .
TEST_DIR = test

# Source files
ASM_SOURCES = \
	vm_entry.asm \
	vm_stack.asm \
	vm_boolean.asm \
	vm_jump_table.asm

# Object files
ASM_OBJECTS = $(patsubst %.asm,$(BUILD_DIR)/%.o,$(ASM_SOURCES))

# Test sources
TEST_SOURCES = $(wildcard $(TEST_DIR)/*.c)
TEST_OBJECTS = $(patsubst $(TEST_DIR)/%.c,$(BUILD_DIR)/%.o,$(TEST_SOURCES))

# Library
LIB = $(BUILD_DIR)/libdtrules_vm.a

# Test executable
TEST_EXE = $(BUILD_DIR)/test_vm

.PHONY: all clean test lib dirs

all: dirs lib

dirs:
	@mkdir -p $(BUILD_DIR)

lib: dirs $(LIB)

$(LIB): $(ASM_OBJECTS)
	ar rcs $@ $^

# Assembly compilation
$(BUILD_DIR)/%.o: %.asm vm_constants.inc vm_state.inc
	$(NASM) $(NASMFLAGS) -o $@ $<

# Test harness compilation
$(BUILD_DIR)/%.o: $(TEST_DIR)/%.c
	$(CC) $(CFLAGS) -c -o $@ $<

# Test executable
$(TEST_EXE): $(LIB) $(TEST_OBJECTS)
	$(CC) $(CFLAGS) -o $@ $(TEST_OBJECTS) -L$(BUILD_DIR) -ldtrules_vm

test: dirs $(TEST_EXE)
	./$(TEST_EXE)

clean:
	rm -rf $(BUILD_DIR)

# Dependency tracking for includes
$(BUILD_DIR)/vm_entry.o: vm_entry.asm vm_constants.inc vm_state.inc
$(BUILD_DIR)/vm_stack.o: vm_stack.asm vm_constants.inc vm_state.inc
$(BUILD_DIR)/vm_boolean.o: vm_boolean.asm vm_constants.inc vm_state.inc
$(BUILD_DIR)/vm_jump_table.o: vm_jump_table.asm vm_constants.inc vm_state.inc

# Show assembly listing
%.lst: %.asm
	$(NASM) $(NASMFLAGS) -l $@ $<

# Help
help:
	@echo "DTRules NASM VM Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build the VM library (default)"
	@echo "  lib     - Build libdtrules_vm.a"
	@echo "  test    - Build and run tests"
	@echo "  clean   - Remove build artifacts"
	@echo "  help    - Show this message"
