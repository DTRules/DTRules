# DTRules x86-64 Assembly Implementation
# Makefile for building the rules engine

# Assembler and linker
ASM = nasm
LD = ld

# Flags
ASMFLAGS = -f elf64 -g -F dwarf -I $(CURDIR)/
LDFLAGS = -nostdlib -static

# Debug build
ifdef DEBUG
ASMFLAGS += -DDEBUG
endif

# Directories
SRC_DIR = src
INC_DIR = include
BUILD_DIR = build
TEST_DIR = test

# Source files
BOOT_SRC = $(SRC_DIR)/boot/start.asm

CORE_SRC = \
	$(SRC_DIR)/core/value.asm \
	$(SRC_DIR)/core/stack_data.asm \
	$(SRC_DIR)/core/stack_entity.asm \
	$(SRC_DIR)/core/stack_control.asm

MEMORY_SRC = \
	$(SRC_DIR)/memory/heap.asm \
	$(SRC_DIR)/memory/pool.asm

TYPES_SRC = \
	$(SRC_DIR)/types/integer.asm \
	$(SRC_DIR)/types/double.asm \
	$(SRC_DIR)/types/string.asm \
	$(SRC_DIR)/types/array.asm \
	$(SRC_DIR)/types/entity.asm

VM_SRC = \
	$(SRC_DIR)/vm/bytecode.asm

DT_SRC = \
	$(SRC_DIR)/dt/cnode.asm \
	$(SRC_DIR)/dt/anode.asm \
	$(SRC_DIR)/dt/table.asm

XML_SRC = \
	$(SRC_DIR)/xml/lexer.asm \
	$(SRC_DIR)/xml/parser.asm \
	$(SRC_DIR)/xml/edd_loader.asm \
	$(SRC_DIR)/xml/dt_loader.asm

IO_SRC = \
	$(SRC_DIR)/io/file.asm \
	$(SRC_DIR)/io/print.asm

ALL_SRC = $(BOOT_SRC) $(CORE_SRC) $(MEMORY_SRC) $(TYPES_SRC) $(VM_SRC) $(DT_SRC) $(XML_SRC) $(IO_SRC)

# Object files
ALL_OBJ = $(patsubst $(SRC_DIR)/%.asm,$(BUILD_DIR)/%.o,$(ALL_SRC))

# Main executable
TARGET = dtrules-asm

# Test executable
TEST_TARGET = dtrules-test

#------------------------------------------------------------------------------
# Main targets
#------------------------------------------------------------------------------

.PHONY: all clean test test-unit test-integration test-comparison dirs

all: dirs $(TARGET)

$(TARGET): $(ALL_OBJ)
	$(LD) $(LDFLAGS) -o $@ $^

#------------------------------------------------------------------------------
# Object file compilation
#------------------------------------------------------------------------------

$(BUILD_DIR)/%.o: $(SRC_DIR)/%.asm $(INC_DIR)/*.inc | dirs
	@mkdir -p $(dir $@)
	$(ASM) $(ASMFLAGS) -o $@ $<

#------------------------------------------------------------------------------
# Directory creation
#------------------------------------------------------------------------------

dirs:
	@mkdir -p $(BUILD_DIR)/boot
	@mkdir -p $(BUILD_DIR)/core
	@mkdir -p $(BUILD_DIR)/memory
	@mkdir -p $(BUILD_DIR)/types
	@mkdir -p $(BUILD_DIR)/vm
	@mkdir -p $(BUILD_DIR)/dt
	@mkdir -p $(BUILD_DIR)/xml
	@mkdir -p $(BUILD_DIR)/io

#------------------------------------------------------------------------------
# Testing
#------------------------------------------------------------------------------

test: test-unit

test-unit: $(TARGET)
	@echo "Running unit tests..."
	@./$(TEST_DIR)/run_tests.sh

test-integration: $(TARGET)
	@echo "Running integration tests..."
	@./$(TEST_DIR)/integration/run_tests.sh

test-comparison: $(TARGET)
	@echo "Running comparison tests..."
	@./$(TEST_DIR)/comparison/runner.sh

#------------------------------------------------------------------------------
# Debug build
#------------------------------------------------------------------------------

debug: ASMFLAGS += -DDEBUG
debug: clean all

#------------------------------------------------------------------------------
# Cleaning
#------------------------------------------------------------------------------

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(TARGET)
	rm -f $(TEST_TARGET)

#------------------------------------------------------------------------------
# Help
#------------------------------------------------------------------------------

help:
	@echo "DTRules x86-64 Assembly Implementation"
	@echo ""
	@echo "Targets:"
	@echo "  all              Build the main executable (default)"
	@echo "  clean            Remove build artifacts"
	@echo "  test             Run unit tests"
	@echo "  test-unit        Run unit tests"
	@echo "  test-integration Run integration tests"
	@echo "  test-comparison  Compare output with Go implementation"
	@echo "  debug            Build with debug symbols and assertions"
	@echo "  help             Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  DEBUG=1          Enable debug build"

#------------------------------------------------------------------------------
# Dependencies
#------------------------------------------------------------------------------

# All objects depend on include files
$(ALL_OBJ): $(INC_DIR)/syscalls.inc $(INC_DIR)/constants.inc $(INC_DIR)/macros.inc
