# DTRules NASM VM - Makefile
# Copyright 2024 Paul Snow - Apache License 2.0

NASM = nasm
NASMFLAGS = -f elf64 -g -F dwarf -I include/
LD = ld
AR = ar

SRCDIR = src
INCDIR = include
TESTDIR = test
BUILDDIR = build

# Source files
SRCS = $(SRCDIR)/vm_core.asm \
       $(SRCDIR)/vm_entry.asm \
       $(SRCDIR)/vm_jump_table.asm \
       $(SRCDIR)/vm_trace.asm

# Object files
OBJS = $(BUILDDIR)/vm_core.o \
       $(BUILDDIR)/vm_entry.o \
       $(BUILDDIR)/vm_jump_table.o \
       $(BUILDDIR)/vm_trace.o

# Library output
LIB = $(BUILDDIR)/libdtrules_vm.a

# Test files
TEST_SRCS = $(wildcard $(TESTDIR)/*.asm)
TEST_OBJS = $(TEST_SRCS:$(TESTDIR)/%.asm=$(BUILDDIR)/test_%.o)
TEST_BINS = $(TEST_SRCS:$(TESTDIR)/%.asm=$(BUILDDIR)/test_%)

.PHONY: all clean test

all: $(BUILDDIR) $(LIB)

$(BUILDDIR):
	mkdir -p $(BUILDDIR)

$(LIB): $(OBJS)
	$(AR) rcs $@ $^

$(BUILDDIR)/vm_core.o: $(SRCDIR)/vm_core.asm $(INCDIR)/dtrules.inc
	$(NASM) $(NASMFLAGS) -o $@ $<

$(BUILDDIR)/vm_entry.o: $(SRCDIR)/vm_entry.asm $(INCDIR)/dtrules.inc
	$(NASM) $(NASMFLAGS) -o $@ $<

$(BUILDDIR)/vm_jump_table.o: $(SRCDIR)/vm_jump_table.asm $(INCDIR)/dtrules.inc
	$(NASM) $(NASMFLAGS) -o $@ $<

$(BUILDDIR)/vm_trace.o: $(SRCDIR)/vm_trace.asm $(INCDIR)/dtrules.inc
	$(NASM) $(NASMFLAGS) -o $@ $<

# Test targets
test: $(BUILDDIR) $(TEST_BINS)
	@echo "Running tests..."
	@for test in $(TEST_BINS); do \
		echo "  Running $$test"; \
		$$test && echo "    PASS" || echo "    FAIL"; \
	done

$(BUILDDIR)/test_%: $(BUILDDIR)/test_%.o $(LIB)
	$(LD) -o $@ $< $(LIB) -lc --dynamic-linker /lib64/ld-linux-x86-64.so.2

$(BUILDDIR)/test_%.o: $(TESTDIR)/%.asm $(INCDIR)/dtrules.inc
	$(NASM) $(NASMFLAGS) -o $@ $<

clean:
	rm -rf $(BUILDDIR)

# Show info
info:
	@echo "Sources: $(SRCS)"
	@echo "Objects: $(OBJS)"
	@echo "Library: $(LIB)"
