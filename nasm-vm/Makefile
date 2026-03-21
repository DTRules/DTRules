# Makefile for DTRules NASM VM
# Build and test assembly-based virtual machine

NASM = nasm
LD = ld

NASM_FLAGS = -f elf64 -g -F dwarf -I ./
LD_FLAGS = -nostdlib

SRC_DIR = src
INC_DIR = include
TEST_DIR = test
BUILD_DIR = build

# Include files
INC_FILES = $(wildcard $(INC_DIR)/*.inc)

# Source files
VM_SRCS = $(SRC_DIR)/vm_core.asm $(SRC_DIR)/data.asm
HARNESS_SRC = $(SRC_DIR)/test_harness.asm

# Object files
VM_OBJS = $(BUILD_DIR)/vm_core.o $(BUILD_DIR)/data.o
HARNESS_OBJ = $(BUILD_DIR)/test_harness.o

# Test executables
TEST_COMPARISON = $(BUILD_DIR)/test_comparison

.PHONY: all clean test

all: $(BUILD_DIR) $(TEST_COMPARISON)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Compile VM core
$(BUILD_DIR)/vm_core.o: $(SRC_DIR)/vm_core.asm $(INC_FILES)
	$(NASM) $(NASM_FLAGS) -o $@ $<

# Compile data segment
$(BUILD_DIR)/data.o: $(SRC_DIR)/data.asm $(INC_FILES)
	$(NASM) $(NASM_FLAGS) -o $@ $<

# Compile test harness
$(BUILD_DIR)/test_harness.o: $(SRC_DIR)/test_harness.asm $(INC_FILES)
	$(NASM) $(NASM_FLAGS) -o $@ $<

# Compile comparison tests
$(BUILD_DIR)/test_comparison.o: $(TEST_DIR)/test_comparison.asm $(INC_FILES)
	$(NASM) $(NASM_FLAGS) -o $@ $<

# Link comparison test
$(TEST_COMPARISON): $(VM_OBJS) $(HARNESS_OBJ) $(BUILD_DIR)/test_comparison.o
	$(LD) $(LD_FLAGS) -o $@ $^

# Run tests
test: $(TEST_COMPARISON)
	@echo "Running comparison tests..."
	@./$(TEST_COMPARISON) && echo "Comparison tests passed!" || (echo "Comparison tests FAILED"; exit 1)

clean:
	rm -rf $(BUILD_DIR)
