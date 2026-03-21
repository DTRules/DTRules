/*
 * vm_test.c - Test harness for DTRules NASM VM
 * Copyright 2024 Paul Snow
 * Licensed under Apache License 2.0
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <assert.h>

/* Value representation (must match vm_core.asm) */
typedef struct {
    int64_t tag;    /* Type tag */
    int64_t value;  /* Value (integer, boolean, or pointer) */
} Value;

/* VM Context structure (must match vm_core.asm) */
typedef struct {
    uint8_t*  bytecode;       /* +0: Bytecode pointer */
    size_t    bytecode_len;   /* +8: Bytecode length */
    Value*    constants;      /* +16: Constants pool */
    size_t    constants_len;  /* +24: Constants count */
    void**    names;          /* +32: Names pool */
    size_t    names_len;      /* +40: Names count */
    Value*    data_stack;     /* +48: Data stack */
    size_t    stack_size;     /* +56: Current stack size */
    size_t    stack_capacity; /* +64: Stack capacity */
    uint8_t*  trace_buf;      /* +72: Trace buffer (optional) */
    size_t    trace_size;     /* +80: Trace buffer size */
} VMContext;

/* Type tags (must match opcodes.inc) */
#define VTAG_NULL     0
#define VTAG_BOOLEAN  1
#define VTAG_INTEGER  2
#define VTAG_DOUBLE   3
#define VTAG_STRING   4
#define VTAG_NAME     5
#define VTAG_ARRAY    6
#define VTAG_ENTITY   7
#define VTAG_MARK     8

/* Opcodes (must match opcodes.inc) */
#define OP_NOP        0
#define OP_PUSH       1
#define OP_PUSH_INT   2
#define OP_POP        3
#define OP_DUP        4
#define OP_SWAP       5
#define OP_ROT        6
#define OP_ROLL       7
#define OP_INDEX      8
#define OP_CLEAR      9
#define OP_MARK       10

#define OP_ADD        20
#define OP_SUB        21
#define OP_MUL        22
#define OP_DIV        23
#define OP_MOD        24
#define OP_NEG        25
#define OP_ABS        26
#define OP_INC        27
#define OP_DEC        28

#define OP_EQ         30
#define OP_NE         31
#define OP_LT         32
#define OP_LE         33
#define OP_GT         34
#define OP_GE         35

#define OP_AND        40
#define OP_OR         41
#define OP_NOT        42
#define OP_XOR        43

#define OP_PUSH_TRUE  100
#define OP_PUSH_FALSE 101
#define OP_PUSH_NULL  102
#define OP_PUSH_ZERO  103
#define OP_PUSH_ONE   104

#define OP_OVER       105
#define OP_PICK       106
#define OP_DROP       107

/* Error codes */
#define ERR_NONE            0
#define ERR_STACK_UNDERFLOW 1
#define ERR_STACK_OVERFLOW  2
#define ERR_INVALID_OPCODE  3
#define ERR_DIV_BY_ZERO     4
#define ERR_OUT_OF_BOUNDS   5

/* External functions from vm_core.asm */
extern int vm_execute(VMContext* ctx);
extern int vm_get_error(void);
extern int vm_get_error_pc(void);

/* Test utilities */
static int tests_run = 0;
static int tests_passed = 0;

#define STACK_CAPACITY 100

static VMContext* create_context(uint8_t* bytecode, size_t len) {
    VMContext* ctx = (VMContext*)calloc(1, sizeof(VMContext));
    ctx->bytecode = bytecode;
    ctx->bytecode_len = len;
    ctx->data_stack = (Value*)calloc(STACK_CAPACITY, sizeof(Value));
    ctx->stack_size = 0;
    ctx->stack_capacity = STACK_CAPACITY;
    return ctx;
}

static void free_context(VMContext* ctx) {
    free(ctx->data_stack);
    free(ctx);
}

static void push_int(VMContext* ctx, int64_t val) {
    ctx->data_stack[ctx->stack_size].tag = VTAG_INTEGER;
    ctx->data_stack[ctx->stack_size].value = val;
    ctx->stack_size++;
}

static void print_stack(VMContext* ctx) {
    printf("Stack [%zu]: ", ctx->stack_size);
    for (size_t i = 0; i < ctx->stack_size; i++) {
        if (ctx->data_stack[i].tag == VTAG_INTEGER) {
            printf("%ld ", ctx->data_stack[i].value);
        } else if (ctx->data_stack[i].tag == VTAG_BOOLEAN) {
            printf("%s ", ctx->data_stack[i].value ? "true" : "false");
        } else if (ctx->data_stack[i].tag == VTAG_NULL) {
            printf("null ");
        } else if (ctx->data_stack[i].tag == VTAG_MARK) {
            printf("mark ");
        } else {
            printf("<%ld:%ld> ", ctx->data_stack[i].tag, ctx->data_stack[i].value);
        }
    }
    printf("\n");
}

#define TEST_START(name) \
    do { \
        tests_run++; \
        printf("Test: %s ... ", name);

#define TEST_END(cond) \
        if (cond) { \
            printf("PASSED\n"); \
            tests_passed++; \
        } else { \
            printf("FAILED\n"); \
        } \
    } while(0)

/* =============================================================================
 * Tests
 * ============================================================================= */

/* Test NOP does nothing */
void test_nop(void) {
    TEST_START("nop");
    uint8_t code[] = { OP_NOP };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 42);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test PUSH_ZERO */
void test_push_zero(void) {
    TEST_START("push_zero");
    uint8_t code[] = { OP_PUSH_ZERO };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_INTEGER &&
              ctx->data_stack[0].value == 0);
    free_context(ctx);
    TEST_END(ok);
}

/* Test PUSH_ONE */
void test_push_one(void) {
    TEST_START("push_one");
    uint8_t code[] = { OP_PUSH_ONE };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_INTEGER &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test PUSH_TRUE */
void test_push_true(void) {
    TEST_START("push_true");
    uint8_t code[] = { OP_PUSH_TRUE };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test PUSH_FALSE */
void test_push_false(void) {
    TEST_START("push_false");
    uint8_t code[] = { OP_PUSH_FALSE };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 0);
    free_context(ctx);
    TEST_END(ok);
}

/* Test PUSH_NULL */
void test_push_null(void) {
    TEST_START("push_null");
    uint8_t code[] = { OP_PUSH_NULL };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_NULL);
    free_context(ctx);
    TEST_END(ok);
}

/* Test POP */
void test_pop(void) {
    TEST_START("pop");
    uint8_t code[] = { OP_POP };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 42);
    push_int(ctx, 100);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test POP underflow */
void test_pop_underflow(void) {
    TEST_START("pop_underflow");
    uint8_t code[] = { OP_POP };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == ERR_STACK_UNDERFLOW);
    free_context(ctx);
    TEST_END(ok);
}

/* Test DUP */
void test_dup(void) {
    TEST_START("dup");
    uint8_t code[] = { OP_DUP };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 42);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 2 &&
              ctx->data_stack[0].value == 42 &&
              ctx->data_stack[1].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test DUP underflow */
void test_dup_underflow(void) {
    TEST_START("dup_underflow");
    uint8_t code[] = { OP_DUP };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == ERR_STACK_UNDERFLOW);
    free_context(ctx);
    TEST_END(ok);
}

/* Test SWAP */
void test_swap(void) {
    TEST_START("swap");
    uint8_t code[] = { OP_SWAP };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);
    push_int(ctx, 2);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 2 &&
              ctx->data_stack[0].value == 2 &&
              ctx->data_stack[1].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test SWAP underflow */
void test_swap_underflow(void) {
    TEST_START("swap_underflow");
    uint8_t code[] = { OP_SWAP };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);

    int result = vm_execute(ctx);

    int ok = (result == ERR_STACK_UNDERFLOW);
    free_context(ctx);
    TEST_END(ok);
}

/* Test OVER */
void test_over(void) {
    TEST_START("over");
    uint8_t code[] = { OP_OVER };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);
    push_int(ctx, 2);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 3 &&
              ctx->data_stack[0].value == 1 &&
              ctx->data_stack[1].value == 2 &&
              ctx->data_stack[2].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test ROT */
void test_rot(void) {
    TEST_START("rot");
    uint8_t code[] = { OP_ROT };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);  /* bottom */
    push_int(ctx, 2);
    push_int(ctx, 3);  /* top */

    int result = vm_execute(ctx);

    /* (1 2 3 -- 2 3 1) */
    int ok = (result == 0 &&
              ctx->stack_size == 3 &&
              ctx->data_stack[0].value == 2 &&
              ctx->data_stack[1].value == 3 &&
              ctx->data_stack[2].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test ROT underflow */
void test_rot_underflow(void) {
    TEST_START("rot_underflow");
    uint8_t code[] = { OP_ROT };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);
    push_int(ctx, 2);

    int result = vm_execute(ctx);

    int ok = (result == ERR_STACK_UNDERFLOW);
    free_context(ctx);
    TEST_END(ok);
}

/* Test CLEAR */
void test_clear(void) {
    TEST_START("clear");
    uint8_t code[] = { OP_CLEAR };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);
    push_int(ctx, 2);
    push_int(ctx, 3);

    int result = vm_execute(ctx);

    int ok = (result == 0 && ctx->stack_size == 0);
    free_context(ctx);
    TEST_END(ok);
}

/* Test MARK */
void test_mark(void) {
    TEST_START("mark");
    uint8_t code[] = { OP_MARK };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_MARK);
    free_context(ctx);
    TEST_END(ok);
}

/* Test ADD */
void test_add(void) {
    TEST_START("add");
    uint8_t code[] = { OP_ADD };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 10);
    push_int(ctx, 32);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test SUB */
void test_sub(void) {
    TEST_START("sub");
    uint8_t code[] = { OP_SUB };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 50);
    push_int(ctx, 8);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test MUL */
void test_mul(void) {
    TEST_START("mul");
    uint8_t code[] = { OP_MUL };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 6);
    push_int(ctx, 7);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test DIV */
void test_div(void) {
    TEST_START("div");
    uint8_t code[] = { OP_DIV };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 84);
    push_int(ctx, 2);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test DIV by zero */
void test_div_zero(void) {
    TEST_START("div_zero");
    uint8_t code[] = { OP_DIV };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 42);
    push_int(ctx, 0);

    int result = vm_execute(ctx);

    int ok = (result == ERR_DIV_BY_ZERO);
    free_context(ctx);
    TEST_END(ok);
}

/* Test MOD */
void test_mod(void) {
    TEST_START("mod");
    uint8_t code[] = { OP_MOD };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 47);
    push_int(ctx, 5);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 2);
    free_context(ctx);
    TEST_END(ok);
}

/* Test NEG */
void test_neg(void) {
    TEST_START("neg");
    uint8_t code[] = { OP_NEG };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 42);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == -42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test ABS positive */
void test_abs_positive(void) {
    TEST_START("abs_positive");
    uint8_t code[] = { OP_ABS };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 42);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test ABS negative */
void test_abs_negative(void) {
    TEST_START("abs_negative");
    uint8_t code[] = { OP_ABS };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, -42);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test INC */
void test_inc(void) {
    TEST_START("inc");
    uint8_t code[] = { OP_INC };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 41);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test DEC */
void test_dec(void) {
    TEST_START("dec");
    uint8_t code[] = { OP_DEC };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 43);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test EQ true */
void test_eq_true(void) {
    TEST_START("eq_true");
    uint8_t code[] = { OP_EQ };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 42);
    push_int(ctx, 42);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test EQ false */
void test_eq_false(void) {
    TEST_START("eq_false");
    uint8_t code[] = { OP_EQ };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 42);
    push_int(ctx, 43);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 0);
    free_context(ctx);
    TEST_END(ok);
}

/* Test NE */
void test_ne(void) {
    TEST_START("ne");
    uint8_t code[] = { OP_NE };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 42);
    push_int(ctx, 43);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test LT */
void test_lt(void) {
    TEST_START("lt");
    uint8_t code[] = { OP_LT };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);
    push_int(ctx, 2);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test LE */
void test_le(void) {
    TEST_START("le");
    uint8_t code[] = { OP_LE };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 2);
    push_int(ctx, 2);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test GT */
void test_gt(void) {
    TEST_START("gt");
    uint8_t code[] = { OP_GT };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 2);
    push_int(ctx, 1);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test GE */
void test_ge(void) {
    TEST_START("ge");
    uint8_t code[] = { OP_GE };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 2);
    push_int(ctx, 2);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test AND */
void test_and(void) {
    TEST_START("and");
    uint8_t code[] = { OP_AND };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);
    push_int(ctx, 1);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test AND false */
void test_and_false(void) {
    TEST_START("and_false");
    uint8_t code[] = { OP_AND };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);
    push_int(ctx, 0);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 0);
    free_context(ctx);
    TEST_END(ok);
}

/* Test OR */
void test_or(void) {
    TEST_START("or");
    uint8_t code[] = { OP_OR };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 0);
    push_int(ctx, 1);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test NOT */
void test_not(void) {
    TEST_START("not");
    uint8_t code[] = { OP_NOT };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 0);
    free_context(ctx);
    TEST_END(ok);
}

/* Test XOR */
void test_xor(void) {
    TEST_START("xor");
    uint8_t code[] = { OP_XOR };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 1);
    push_int(ctx, 0);

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_BOOLEAN &&
              ctx->data_stack[0].value == 1);
    free_context(ctx);
    TEST_END(ok);
}

/* Test PUSH_INT with small value */
void test_push_int(void) {
    TEST_START("push_int");
    uint8_t code[] = { OP_PUSH_INT, 42 };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_INTEGER &&
              ctx->data_stack[0].value == 42);
    free_context(ctx);
    TEST_END(ok);
}

/* Test PUSH_INT with larger varint value */
void test_push_int_varint(void) {
    TEST_START("push_int_varint");
    /* 200 = 0xC8 = 0b11001000 -> varint: 0xC8, 0x01 */
    uint8_t code[] = { OP_PUSH_INT, 0xC8, 0x01 };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].tag == VTAG_INTEGER &&
              ctx->data_stack[0].value == 200);
    free_context(ctx);
    TEST_END(ok);
}

/* Test combined operations: (5 + 3) * 2 = 16 */
void test_combined_ops(void) {
    TEST_START("combined_ops");
    uint8_t code[] = {
        OP_PUSH_INT, 5,
        OP_PUSH_INT, 3,
        OP_ADD,
        OP_PUSH_INT, 2,
        OP_MUL
    };
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == 0 &&
              ctx->stack_size == 1 &&
              ctx->data_stack[0].value == 16);
    free_context(ctx);
    TEST_END(ok);
}

/* Test invalid opcode */
void test_invalid_opcode(void) {
    TEST_START("invalid_opcode");
    uint8_t code[] = { 255 };  /* Invalid opcode */
    VMContext* ctx = create_context(code, sizeof(code));

    int result = vm_execute(ctx);

    int ok = (result == ERR_INVALID_OPCODE);
    free_context(ctx);
    TEST_END(ok);
}

/* Test INDEX/PICK */
void test_pick(void) {
    TEST_START("pick");
    /* Push 1, 2, 3 then pick 2nd (index 1) */
    uint8_t code[] = { OP_PICK };
    VMContext* ctx = create_context(code, sizeof(code));
    push_int(ctx, 100);  /* bottom */
    push_int(ctx, 200);
    push_int(ctx, 300);  /* top */
    push_int(ctx, 1);    /* pick index */

    int result = vm_execute(ctx);

    /* Stack should be: 100, 200, 300, 200 */
    int ok = (result == 0 &&
              ctx->stack_size == 4 &&
              ctx->data_stack[3].value == 200);
    free_context(ctx);
    TEST_END(ok);
}

/* =============================================================================
 * Main
 * ============================================================================= */

int main(void) {
    printf("DTRules NASM VM Tests\n");
    printf("=====================\n\n");

    /* Stack operations */
    test_nop();
    test_push_zero();
    test_push_one();
    test_push_true();
    test_push_false();
    test_push_null();
    test_push_int();
    test_push_int_varint();
    test_pop();
    test_pop_underflow();
    test_dup();
    test_dup_underflow();
    test_swap();
    test_swap_underflow();
    test_over();
    test_rot();
    test_rot_underflow();
    test_clear();
    test_mark();
    test_pick();

    /* Arithmetic */
    test_add();
    test_sub();
    test_mul();
    test_div();
    test_div_zero();
    test_mod();
    test_neg();
    test_abs_positive();
    test_abs_negative();
    test_inc();
    test_dec();

    /* Comparison */
    test_eq_true();
    test_eq_false();
    test_ne();
    test_lt();
    test_le();
    test_gt();
    test_ge();

    /* Boolean */
    test_and();
    test_and_false();
    test_or();
    test_not();
    test_xor();

    /* Combined */
    test_combined_ops();

    /* Error cases */
    test_invalid_opcode();

    printf("\n=====================\n");
    printf("Results: %d/%d tests passed\n", tests_passed, tests_run);

    return (tests_passed == tests_run) ? 0 : 1;
}
