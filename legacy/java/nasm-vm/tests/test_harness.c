/**
 * DTRules NASM VM - Test Harness
 * Copyright 2024 Paul Snow
 * Licensed under the Apache License, Version 2.0
 *
 * This provides a C test harness for the NASM VM.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

/* Value type tags - must match value.inc */
#define VTAG_NULL       0
#define VTAG_INTEGER    1
#define VTAG_DOUBLE     2
#define VTAG_BOOLEAN    3
#define VTAG_STRING     4
#define VTAG_NAME       5
#define VTAG_ARRAY      6
#define VTAG_ENTITY     7
#define VTAG_OBJECT     8

/* Error codes - must match value.inc */
#define ERR_NONE            0
#define ERR_STACK_UNDERFLOW 1
#define ERR_STACK_OVERFLOW  2
#define ERR_TYPE_MISMATCH   3
#define ERR_DIV_BY_ZERO     4
#define ERR_INVALID_OPCODE  5

/* Opcodes - must match opcodes.inc */
#define OP_NOP          0
#define OP_PUSH         1
#define OP_PUSH_INT     2
#define OP_POP          3
#define OP_DUP          4
#define OP_SWAP         5
#define OP_ADD          20
#define OP_SUB          21
#define OP_MUL          22
#define OP_DIV          23
#define OP_MOD          24
#define OP_NEG          25
#define OP_ABS          26
#define OP_INC          27
#define OP_DEC          28
#define OP_PUSH_TRUE    100
#define OP_PUSH_FALSE   101
#define OP_PUSH_NULL    102
#define OP_PUSH_ZERO    103
#define OP_PUSH_ONE     104
#define OP_HALT         255

/* Value structure - must match vm_state.inc */
typedef struct {
    uint8_t tag;
    uint8_t padding[7];
    int64_t num;
    void*   ptr;
} Value;

/* VM State structure - must match vm_state.inc */
typedef struct {
    uint8_t* bytecode;
    uint64_t bytecode_len;
    uint64_t pc;

    Value*   stack;
    uint64_t stack_size;
    uint64_t sp;

    Value*   constants;
    uint64_t const_count;

    void**   names;
    uint64_t name_count;

    uint8_t* trace_buf;
    uint64_t trace_pos;
    uint64_t trace_size;

    uint64_t error_code;
    uint64_t error_pc;
} VMState;

/* External VM function */
extern int vm_execute(VMState* state);

/* Test counters */
static int tests_run = 0;
static int tests_passed = 0;

/* Initialize VM state */
static void vm_init(VMState* state, uint8_t* bytecode, size_t len, Value* stack, size_t stack_size) {
    memset(state, 0, sizeof(VMState));
    state->bytecode = bytecode;
    state->bytecode_len = len;
    state->pc = 0;
    state->stack = stack;
    state->stack_size = stack_size;
    state->sp = 0;
}

/* Push an integer onto the stack (for test setup) */
static void push_int(VMState* state, int64_t val) {
    Value* v = &state->stack[state->sp / sizeof(Value)];
    v->tag = VTAG_INTEGER;
    v->num = val;
    v->ptr = NULL;
    state->sp += sizeof(Value);
}

/* Get integer from top of stack */
static int64_t top_int(VMState* state) {
    if (state->sp == 0) return 0;
    Value* v = &state->stack[(state->sp / sizeof(Value)) - 1];
    return v->num;
}

/* Get tag from top of stack */
static uint8_t top_tag(VMState* state) {
    if (state->sp == 0) return VTAG_NULL;
    Value* v = &state->stack[(state->sp / sizeof(Value)) - 1];
    return v->tag;
}

/* Get stack depth */
static size_t stack_depth(VMState* state) {
    return state->sp / sizeof(Value);
}

/* Test assertion macros */
#define ASSERT(cond, msg) do { \
    tests_run++; \
    if (cond) { \
        tests_passed++; \
        printf("  PASS: %s\n", msg); \
    } else { \
        printf("  FAIL: %s\n", msg); \
    } \
} while(0)

#define ASSERT_EQ(expected, actual, msg) do { \
    tests_run++; \
    if ((expected) == (actual)) { \
        tests_passed++; \
        printf("  PASS: %s (got %ld)\n", msg, (long)(actual)); \
    } else { \
        printf("  FAIL: %s (expected %ld, got %ld)\n", msg, (long)(expected), (long)(actual)); \
    } \
} while(0)

/* Test: NOP instruction */
static void test_nop(void) {
    printf("Test: NOP\n");

    uint8_t bytecode[] = { OP_NOP, OP_NOP, OP_NOP };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "NOP returns success");
    ASSERT_EQ(3, state.pc, "PC advances through NOPs");
    ASSERT_EQ(0, stack_depth(&state), "Stack is empty");
}

/* Test: PUSH_ZERO and PUSH_ONE */
static void test_push_constants(void) {
    printf("Test: PUSH_ZERO and PUSH_ONE\n");

    uint8_t bytecode[] = { OP_PUSH_ZERO, OP_PUSH_ONE };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "Returns success");
    ASSERT_EQ(2, stack_depth(&state), "Stack has 2 entries");
    ASSERT_EQ(1, top_int(&state), "Top is 1");
    ASSERT_EQ(VTAG_INTEGER, top_tag(&state), "Top is integer");
}

/* Test: ADD operation */
static void test_add(void) {
    printf("Test: ADD\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 5, OP_PUSH_INT, 3, OP_ADD };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "ADD returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(8, top_int(&state), "5 + 3 = 8");
}

/* Test: SUB operation */
static void test_sub(void) {
    printf("Test: SUB\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 10, OP_PUSH_INT, 3, OP_SUB };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "SUB returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(7, top_int(&state), "10 - 3 = 7");
}

/* Test: MUL operation */
static void test_mul(void) {
    printf("Test: MUL\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 6, OP_PUSH_INT, 7, OP_MUL };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "MUL returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(42, top_int(&state), "6 * 7 = 42");
}

/* Test: DIV operation */
static void test_div(void) {
    printf("Test: DIV\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 20, OP_PUSH_INT, 4, OP_DIV };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "DIV returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(5, top_int(&state), "20 / 4 = 5");
}

/* Test: DIV by zero */
static void test_div_by_zero(void) {
    printf("Test: DIV by zero\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 10, OP_PUSH_ZERO, OP_DIV };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_DIV_BY_ZERO, result, "DIV by zero returns error");
    ASSERT_EQ(ERR_DIV_BY_ZERO, state.error_code, "Error code is set");
}

/* Test: MOD operation */
static void test_mod(void) {
    printf("Test: MOD\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 17, OP_PUSH_INT, 5, OP_MOD };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "MOD returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(2, top_int(&state), "17 % 5 = 2");
}

/* Test: NEG operation */
static void test_neg(void) {
    printf("Test: NEG\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 42, OP_NEG };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "NEG returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(-42, top_int(&state), "neg(42) = -42");
}

/* Test: ABS operation */
static void test_abs(void) {
    printf("Test: ABS\n");

    /* Test with negative number - we need to push a negative value manually */
    Value stack[16];
    VMState state;

    /* First, use neg to create -42, then abs it */
    uint8_t bytecode[] = { OP_PUSH_INT, 42, OP_NEG, OP_ABS };

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "ABS returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(42, top_int(&state), "abs(-42) = 42");
}

/* Test: INC operation */
static void test_inc(void) {
    printf("Test: INC\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 99, OP_INC };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "INC returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(100, top_int(&state), "inc(99) = 100");
}

/* Test: DEC operation */
static void test_dec(void) {
    printf("Test: DEC\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 100, OP_DEC };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "DEC returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(99, top_int(&state), "dec(100) = 99");
}

/* Test: DUP operation */
static void test_dup(void) {
    printf("Test: DUP\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 42, OP_DUP };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "DUP returns success");
    ASSERT_EQ(2, stack_depth(&state), "Stack has 2 entries");
    ASSERT_EQ(42, top_int(&state), "Top is 42");
}

/* Test: SWAP operation */
static void test_swap(void) {
    printf("Test: SWAP\n");

    uint8_t bytecode[] = { OP_PUSH_INT, 1, OP_PUSH_INT, 2, OP_SWAP };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "SWAP returns success");
    ASSERT_EQ(2, stack_depth(&state), "Stack has 2 entries");
    ASSERT_EQ(1, top_int(&state), "Top is 1 after swap");
}

/* Test: Stack underflow */
static void test_stack_underflow(void) {
    printf("Test: Stack underflow\n");

    uint8_t bytecode[] = { OP_ADD };  /* ADD with empty stack */
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_STACK_UNDERFLOW, result, "ADD on empty stack returns underflow");
}

/* Test: Complex expression: (5 + 3) * 2 */
static void test_complex_expression(void) {
    printf("Test: Complex expression (5 + 3) * 2\n");

    uint8_t bytecode[] = {
        OP_PUSH_INT, 5,
        OP_PUSH_INT, 3,
        OP_ADD,
        OP_PUSH_INT, 2,
        OP_MUL
    };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "Expression returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(16, top_int(&state), "(5 + 3) * 2 = 16");
}

/* Test: PUSH_INT with varint encoding */
static void test_push_int_large(void) {
    printf("Test: PUSH_INT with value 128 (multi-byte varint)\n");

    /* 128 in varint: 0x80, 0x01 */
    uint8_t bytecode[] = { OP_PUSH_INT, 0x80, 0x01 };
    Value stack[16];
    VMState state;

    vm_init(&state, bytecode, sizeof(bytecode), stack, 16);

    int result = vm_execute(&state);

    ASSERT_EQ(ERR_NONE, result, "PUSH_INT returns success");
    ASSERT_EQ(1, stack_depth(&state), "Stack has 1 entry");
    ASSERT_EQ(128, top_int(&state), "Pushed value is 128");
}

int main(void) {
    printf("DTRules NASM VM - Arithmetic Operations Tests\n");
    printf("=============================================\n\n");

    test_nop();
    test_push_constants();
    test_add();
    test_sub();
    test_mul();
    test_div();
    test_div_by_zero();
    test_mod();
    test_neg();
    test_abs();
    test_inc();
    test_dec();
    test_dup();
    test_swap();
    test_stack_underflow();
    test_complex_expression();
    test_push_int_large();

    printf("\n=============================================\n");
    printf("Tests passed: %d/%d\n", tests_passed, tests_run);

    return tests_passed == tests_run ? 0 : 1;
}
