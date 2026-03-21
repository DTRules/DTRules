# DTRules NASM VM Makefile
# Copyright 2024 Paul Snow
# Licensed under Apache License 2.0

# Compiler/Assembler settings
NASM = nasm
CC = gcc
NASMFLAGS = -f elf64 -g -F dwarf -I./
CFLAGS = -Wall -Wextra -g -O2 -std=c11
LDFLAGS = -no-pie

# Output directories
BUILD_DIR = build
OBJ_DIR = $(BUILD_DIR)/obj

# Source files - vm_jump_table includes vm_core via %include
ASM_SOURCES = vm_entry.asm vm_jump_table.asm
C_SOURCES = test_harness.c

# Object files
ASM_OBJECTS = $(patsubst %.asm,$(OBJ_DIR)/%.o,$(ASM_SOURCES))
C_OBJECTS = $(patsubst %.c,$(OBJ_DIR)/%.o,$(C_SOURCES))

# Main targets
.PHONY: all clean test directories

all: directories $(BUILD_DIR)/test_vm

directories:
	@mkdir -p $(BUILD_DIR) $(OBJ_DIR)

# Link the test executable
$(BUILD_DIR)/test_vm: $(ASM_OBJECTS) $(C_OBJECTS)
	$(CC) $(LDFLAGS) -o $@ $^

# Assemble NASM files
# vm_entry.asm includes vm_core.asm
# vm_jump_table.asm includes vm_core.asm
$(OBJ_DIR)/vm_entry.o: vm_entry.asm vm_core.asm vm_constants.inc vm_state.inc
	$(NASM) $(NASMFLAGS) -o $@ $<

$(OBJ_DIR)/vm_jump_table.o: vm_jump_table.asm vm_core.asm vm_constants.inc vm_state.inc
	$(NASM) $(NASMFLAGS) -o $@ $<

# Compile C files
$(OBJ_DIR)/%.o: %.c
	$(CC) $(CFLAGS) -c -o $@ $<

# Run tests
test: $(BUILD_DIR)/test_vm
	./$(BUILD_DIR)/test_vm

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Debug build with extra symbols
debug: NASMFLAGS += -DDEBUG
debug: CFLAGS += -DDEBUG -O0
debug: all

# Show structure sizes (for verification)
sizes: $(OBJ_DIR)/vm_entry.o
	@echo "Checking structure sizes..."
	@nm $(OBJ_DIR)/vm_entry.o | grep -i size || true

# Disassemble for inspection
disasm: $(OBJ_DIR)/vm_jump_table.o
	objdump -d -M intel $(OBJ_DIR)/vm_jump_table.o

# Help
help:
	@echo "DTRules NASM VM Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all      - Build test executable (default)"
	@echo "  test     - Build and run tests"
	@echo "  clean    - Remove build artifacts"
	@echo "  debug    - Build with debug symbols"
	@echo "  disasm   - Disassemble object files"
	@echo "  help     - Show this help"
