# DTRules NASM VM Makefile
# Copyright 2024 Paul Snow
# Licensed under Apache License 2.0

# Toolchain
NASM = nasm
CC = gcc
LD = gcc

# Directories
SRC_DIR = src
BUILD_DIR = build
TEST_DIR = tests

# Flags
NASM_FLAGS = -f elf64 -g -F dwarf
CFLAGS = -Wall -Wextra -g -O2 -I$(SRC_DIR)
LDFLAGS = -no-pie

# Source files
ASM_SOURCES = $(SRC_DIR)/vm_core.asm
C_SOURCES = $(SRC_DIR)/vm_test.c

# Object files
ASM_OBJECTS = $(BUILD_DIR)/vm_core.o
C_OBJECTS = $(BUILD_DIR)/vm_test.o

# Targets
TEST_BIN = $(BUILD_DIR)/vm_test

.PHONY: all clean test

all: $(TEST_BIN)

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Assemble NASM files
$(BUILD_DIR)/vm_core.o: $(SRC_DIR)/vm_core.asm $(SRC_DIR)/opcodes.inc | $(BUILD_DIR)
	$(NASM) $(NASM_FLAGS) -o $@ $<

# Compile C files
$(BUILD_DIR)/vm_test.o: $(SRC_DIR)/vm_test.c | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c -o $@ $<

# Link test binary
$(TEST_BIN): $(ASM_OBJECTS) $(C_OBJECTS)
	$(LD) $(LDFLAGS) -o $@ $^

# Run tests
test: $(TEST_BIN)
	./$(TEST_BIN)

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Debug build with extra symbols
debug: NASM_FLAGS += -l $(BUILD_DIR)/vm_core.lst
debug: CFLAGS += -DDEBUG -O0
debug: all

# Help
help:
	@echo "DTRules NASM VM Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build everything (default)"
	@echo "  test    - Build and run tests"
	@echo "  clean   - Remove build artifacts"
	@echo "  debug   - Build with debug symbols and listing"
	@echo "  help    - Show this help"
