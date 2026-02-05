# DTRules x86-64 Assembly Unit Tests
# Makefile for building and running unit test executables

# Assembler and linker
ASM = nasm
LD = ld

# Directories - paths relative to this Makefile
PROJECT_DIR = ../..
SRC_DIR = $(PROJECT_DIR)/src
INC_DIR = $(PROJECT_DIR)/include
BUILD_DIR = $(PROJECT_DIR)/build
TEST_BUILD_DIR = build

# Flags
ASMFLAGS = -f elf64 -g -F dwarf -I $(PROJECT_DIR)/
LDFLAGS = -nostdlib -static

# Source files needed from main project
CORE_OBJ = \
	$(BUILD_DIR)/core/value.o \
	$(BUILD_DIR)/core/stack_data.o \
	$(BUILD_DIR)/core/stack_entity.o \
	$(BUILD_DIR)/core/stack_control.o

MEMORY_OBJ = \
	$(BUILD_DIR)/memory/heap.o \
	$(BUILD_DIR)/memory/pool.o

TYPES_OBJ = \
	$(BUILD_DIR)/types/integer.o \
	$(BUILD_DIR)/types/double.o \
	$(BUILD_DIR)/types/string.o \
	$(BUILD_DIR)/types/array.o \
	$(BUILD_DIR)/types/entity.o

IO_OBJ = \
	$(BUILD_DIR)/io/file.o \
	$(BUILD_DIR)/io/print.o

# Common objects needed by all tests
COMMON_OBJ = $(CORE_OBJ) $(MEMORY_OBJ) $(TYPES_OBJ) $(IO_OBJ)

# Test source files
TEST_SOURCES = $(wildcard test_*.asm)
TEST_NAMES = $(TEST_SOURCES:.asm=)
TEST_EXECUTABLES = $(addprefix $(TEST_BUILD_DIR)/,$(TEST_NAMES))

#------------------------------------------------------------------------------
# Main targets
#------------------------------------------------------------------------------

.PHONY: all clean run test dirs build-main

all: dirs build-main $(TEST_EXECUTABLES)

# Build main project first to ensure objects exist
build-main:
	@$(MAKE) -C $(PROJECT_DIR) all

dirs:
	@mkdir -p $(TEST_BUILD_DIR)

#------------------------------------------------------------------------------
# Test harness - provides _start and test infrastructure
#------------------------------------------------------------------------------

$(TEST_BUILD_DIR)/test_harness.o: test_harness.asm | dirs
	$(ASM) $(ASMFLAGS) -o $@ $<

#------------------------------------------------------------------------------
# Individual test builds
#------------------------------------------------------------------------------

$(TEST_BUILD_DIR)/test_value.o: test_value.asm | dirs
	$(ASM) $(ASMFLAGS) -o $@ $<

$(TEST_BUILD_DIR)/test_stack.o: test_stack.asm | dirs
	$(ASM) $(ASMFLAGS) -o $@ $<

$(TEST_BUILD_DIR)/test_arithmetic.o: test_arithmetic.asm | dirs
	$(ASM) $(ASMFLAGS) -o $@ $<

$(TEST_BUILD_DIR)/test_comparison.o: test_comparison.asm | dirs
	$(ASM) $(ASMFLAGS) -o $@ $<

$(TEST_BUILD_DIR)/test_boolean.o: test_boolean.asm | dirs
	$(ASM) $(ASMFLAGS) -o $@ $<

#------------------------------------------------------------------------------
# Link test executables
#------------------------------------------------------------------------------

$(TEST_BUILD_DIR)/test_value: $(TEST_BUILD_DIR)/test_value.o $(TEST_BUILD_DIR)/test_harness.o $(COMMON_OBJ)
	$(LD) $(LDFLAGS) -o $@ $^

$(TEST_BUILD_DIR)/test_stack: $(TEST_BUILD_DIR)/test_stack.o $(TEST_BUILD_DIR)/test_harness.o $(COMMON_OBJ)
	$(LD) $(LDFLAGS) -o $@ $^

$(TEST_BUILD_DIR)/test_arithmetic: $(TEST_BUILD_DIR)/test_arithmetic.o $(TEST_BUILD_DIR)/test_harness.o $(COMMON_OBJ)
	$(LD) $(LDFLAGS) -o $@ $^

$(TEST_BUILD_DIR)/test_comparison: $(TEST_BUILD_DIR)/test_comparison.o $(TEST_BUILD_DIR)/test_harness.o $(COMMON_OBJ)
	$(LD) $(LDFLAGS) -o $@ $^

$(TEST_BUILD_DIR)/test_boolean: $(TEST_BUILD_DIR)/test_boolean.o $(TEST_BUILD_DIR)/test_harness.o $(COMMON_OBJ)
	$(LD) $(LDFLAGS) -o $@ $^

#------------------------------------------------------------------------------
# Run tests
#------------------------------------------------------------------------------

run: all
	@echo ""
	@echo "Running Unit Tests"
	@echo "=================="
	@echo ""
	@failed=0; \
	for test in $(TEST_EXECUTABLES); do \
		if [ -f "$$test" ]; then \
			name=$$(basename $$test); \
			echo -n "Running $$name... "; \
			if $$test; then \
				echo "OK"; \
			else \
				echo "FAILED"; \
				failed=$$((failed + 1)); \
			fi; \
		fi; \
	done; \
	echo ""; \
	if [ $$failed -eq 0 ]; then \
		echo "All tests passed!"; \
	else \
		echo "$$failed test(s) failed"; \
		exit 1; \
	fi

test: run

#------------------------------------------------------------------------------
# Cleaning
#------------------------------------------------------------------------------

clean:
	rm -rf $(TEST_BUILD_DIR)

#------------------------------------------------------------------------------
# Help
#------------------------------------------------------------------------------

help:
	@echo "DTRules Unit Test Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all    - Build all test executables"
	@echo "  run    - Run all unit tests"
	@echo "  test   - Same as run"
	@echo "  clean  - Remove test build artifacts"
	@echo "  help   - Show this message"
