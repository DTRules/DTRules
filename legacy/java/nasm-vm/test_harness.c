/**
 * DTRules NASM VM Test Harness
 * Copyright 2024 Paul Snow
 * Licensed under Apache License 2.0
 *
 * This file provides a C test harness for the NASM VM.
 * It tests individual opcodes and validates correctness.
 */

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <assert.h>

/* Value type tags */
#define VTAG_NULL       0
#define VTAG_INTEGER    1
#define VTAG_DOUBLE     2
#define VTAG_BOOLEAN    3
#define VTAG_STRING     4
#define VTAG_NAME       5
#define VTAG_ARRAY      6
#define VTAG_ENTITY     7
#define VTAG_OBJECT     8

/* Error codes */
#define ERR_NONE            0
#define ERR_STACK_OVERFLOW  1
#define ERR_STACK_UNDERFLOW 2
#define ERR_INVALID_OPCODE  3
#define ERR_TYPE_MISMATCH   4
#define ERR_DIV_ZERO        5
#define ERR_OUT_OF_BOUNDS   6
#define ERR_NULL_PTR        7
#define ERR_NOT_IMPLEMENTED 8

/* Opcodes */
#define OP_NOP          0
#define OP_PUSH         1
#define OP_PUSH_INT     2
#define OP_POP          3
#define OP_DUP          4
#define OP_SWAP         5
#define OP_ROT          6

#define OP_ADD          20
#define OP_SUB          21
#define OP_MUL          22
#define OP_DIV          23
#define OP_MOD          24
#define OP_NEG          25
#define OP_ABS          26
#define OP_INC          27
#define OP_DEC          28

#define OP_EQ           30
#define OP_NE           31
#define OP_LT           32
#define OP_LE           33
#define OP_GT           34
#define OP_GE           35

#define OP_AND          40
#define OP_OR           41
#define OP_NOT          42
#define OP_XOR          43

#define OP_PUSH_TRUE    100
#define OP_PUSH_FALSE   101
#define OP_PUSH_NULL    102
#define OP_PUSH_ZERO    103
#define OP_PUSH_ONE     104

#define OP_HALT         255

/* Value structure (24 bytes, matching NASM layout) */
typedef struct {
    uint8_t tag;
    uint8_t padding[7];
    int64_t num;
    void *ptr;
} Value;

/* VMContext structure (must match NASM layout exactly) */
typedef struct {
    /* Stack pointers */
    void *data_sp;
    void *entity_sp;
    void *ctrl_sp;

    /* Stack bases */
    void *data_base;
    void *entity_base;
    void *ctrl_base;

    /* Stack limits */
    void *data_limit;
    void *entity_limit;
    void *ctrl_limit;

    /* Bytecode execution */
    uint8_t *bytecode;
    uint64_t bytecode_len;
    uint64_t pc;

    /* Constant pools */
    Value *constants;
    uint64_t const_count;
    void **names;
    uint64_t name_count;

    /* Jump table */
    void **jump_table;

    /* State flags */
    uint32_t state_flags;
    uint32_t error_code;

    /* Frame management */
    uint64_t current_frame;
    void *frames;
    uint64_t frame_count;

    /* Trace buffer */
    void *trace_buf;
    void *trace_pos;
    void *trace_limit;

    /* Session reference */
    void *session;

    /* Operator table */
    void *op_table;
    uint64_t op_count;

    /* Statistics */
    uint64_t insn_count;
    uint64_t cycle_count;
} VMContext;

/* External functions from NASM */
extern int vm_execute(VMContext *ctx);
extern void vm_load_bytecode(VMContext *ctx, void *bytecode, size_t len);
extern void vm_set_jump_table(VMContext *ctx, void **jump_table);
extern size_t vm_get_stack_depth(VMContext *ctx);
extern int vm_peek_stack(VMContext *ctx, size_t index, Value *out);
extern uint64_t vm_get_insn_count(VMContext *ctx);
extern int vm_get_error(VMContext *ctx);

/* External jump table from NASM */
extern void *vm_jump_table[];

/* Stack size in bytes */
#define STACK_SIZE (1000 * sizeof(Value))

/* Test context with allocated stacks */
typedef struct {
    VMContext ctx;
    uint8_t data_stack[STACK_SIZE];
    uint8_t entity_stack[STACK_SIZE];
    uint8_t ctrl_stack[STACK_SIZE];
} TestContext;

/* Initialize a test context */
static void init_test_context(TestContext *tc) {
    memset(tc, 0, sizeof(*tc));

    /* Data stack */
    tc->ctx.data_base = tc->data_stack;
    tc->ctx.data_sp = tc->data_stack;
    tc->ctx.data_limit = tc->data_stack + STACK_SIZE;

    /* Entity stack */
    tc->ctx.entity_base = tc->entity_stack;
    tc->ctx.entity_sp = tc->entity_stack;
    tc->ctx.entity_limit = tc->entity_stack + STACK_SIZE;

    /* Control stack */
    tc->ctx.ctrl_base = tc->ctrl_stack;
    tc->ctx.ctrl_sp = tc->ctrl_stack;
    tc->ctx.ctrl_limit = tc->ctrl_stack + STACK_SIZE;

    /* Jump table */
    tc->ctx.jump_table = vm_jump_table;
}

/* Test counter */
static int tests_run = 0;
static int tests_passed = 0;

#define TEST_START(name) \
    do { \
        tests_run++; \
        printf("  Test: %s... ", name); \
    } while(0)

#define TEST_PASS() \
    do { \
        tests_passed++; \
        printf("PASS\n"); \
    } while(0)

#define TEST_FAIL(msg) \
    do { \
        printf("FAIL: %s\n", msg); \
    } while(0)

#define ASSERT_EQ(expected, actual, msg) \
    do { \
        if ((expected) != (actual)) { \
            char buf[256]; \
            snprintf(buf, sizeof(buf), "%s (expected %ld, got %ld)", \
                     msg, (long)(expected), (long)(actual)); \
            TEST_FAIL(buf); \
            return; \
        } \
    } while(0)

/* ========================================================================== */
/* Stack Operation Tests */
/* ========================================================================== */

static void test_push_int(void) {
    TEST_START("push_int");
    TestContext tc;
    init_test_context(&tc);

    /* Bytecode: push_int 42, halt */
    uint8_t code[] = {OP_PUSH_INT, 42, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");
    ASSERT_EQ(1, vm_get_stack_depth(&tc.ctx), "stack depth");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(VTAG_INTEGER, v.tag, "value tag");
    ASSERT_EQ(42, v.num, "value");

    TEST_PASS();
}

static void test_push_zero_one(void) {
    TEST_START("push_zero/one");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_ZERO, OP_PUSH_ONE, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");
    ASSERT_EQ(2, vm_get_stack_depth(&tc.ctx), "stack depth");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(VTAG_INTEGER, v.tag, "top tag");
    ASSERT_EQ(1, v.num, "top value");

    vm_peek_stack(&tc.ctx, 1, &v);
    ASSERT_EQ(VTAG_INTEGER, v.tag, "second tag");
    ASSERT_EQ(0, v.num, "second value");

    TEST_PASS();
}

static void test_push_bool(void) {
    TEST_START("push_true/false");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_FALSE, OP_PUSH_TRUE, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");
    ASSERT_EQ(2, vm_get_stack_depth(&tc.ctx), "stack depth");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(VTAG_BOOLEAN, v.tag, "top tag");
    ASSERT_EQ(1, v.num, "top value (true)");

    vm_peek_stack(&tc.ctx, 1, &v);
    ASSERT_EQ(VTAG_BOOLEAN, v.tag, "second tag");
    ASSERT_EQ(0, v.num, "second value (false)");

    TEST_PASS();
}

static void test_push_null(void) {
    TEST_START("push_null");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_NULL, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");
    ASSERT_EQ(1, vm_get_stack_depth(&tc.ctx), "stack depth");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(VTAG_NULL, v.tag, "value tag");

    TEST_PASS();
}

static void test_pop(void) {
    TEST_START("pop");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 1, OP_PUSH_INT, 2, OP_POP, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");
    ASSERT_EQ(1, vm_get_stack_depth(&tc.ctx), "stack depth");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(1, v.num, "remaining value");

    TEST_PASS();
}

static void test_dup(void) {
    TEST_START("dup");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 42, OP_DUP, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");
    ASSERT_EQ(2, vm_get_stack_depth(&tc.ctx), "stack depth");

    Value v1, v2;
    vm_peek_stack(&tc.ctx, 0, &v1);
    vm_peek_stack(&tc.ctx, 1, &v2);
    ASSERT_EQ(42, v1.num, "top value");
    ASSERT_EQ(42, v2.num, "second value");

    TEST_PASS();
}

static void test_swap(void) {
    TEST_START("swap");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 1, OP_PUSH_INT, 2, OP_SWAP, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");
    ASSERT_EQ(2, vm_get_stack_depth(&tc.ctx), "stack depth");

    Value v1, v2;
    vm_peek_stack(&tc.ctx, 0, &v1);
    vm_peek_stack(&tc.ctx, 1, &v2);
    ASSERT_EQ(1, v1.num, "top value");
    ASSERT_EQ(2, v2.num, "second value");

    TEST_PASS();
}

/* ========================================================================== */
/* Arithmetic Tests */
/* ========================================================================== */

static void test_add(void) {
    TEST_START("add");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 10, OP_PUSH_INT, 32, OP_ADD, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");
    ASSERT_EQ(1, vm_get_stack_depth(&tc.ctx), "stack depth");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(42, v.num, "10 + 32 = 42");

    TEST_PASS();
}

static void test_sub(void) {
    TEST_START("sub");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 100, OP_PUSH_INT, 58, OP_SUB, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(42, v.num, "100 - 58 = 42");

    TEST_PASS();
}

static void test_mul(void) {
    TEST_START("mul");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 6, OP_PUSH_INT, 7, OP_MUL, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(42, v.num, "6 * 7 = 42");

    TEST_PASS();
}

static void test_div(void) {
    TEST_START("div");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 84, OP_PUSH_INT, 2, OP_DIV, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(42, v.num, "84 / 2 = 42");

    TEST_PASS();
}

static void test_div_zero(void) {
    TEST_START("div by zero");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 42, OP_PUSH_INT, 0, OP_DIV, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_DIV_ZERO, err, "should error on div by zero");

    TEST_PASS();
}

static void test_mod(void) {
    TEST_START("mod");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 47, OP_PUSH_INT, 5, OP_MOD, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(2, v.num, "47 % 5 = 2");

    TEST_PASS();
}

static void test_neg(void) {
    TEST_START("neg");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 42, OP_NEG, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(-42, v.num, "-42");

    TEST_PASS();
}

static void test_abs(void) {
    TEST_START("abs");
    TestContext tc;
    init_test_context(&tc);

    /* Push 42, negate, then abs to get back to 42 */
    uint8_t code[] = {OP_PUSH_INT, 42, OP_NEG, OP_ABS, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(42, v.num, "abs(-42) = 42");

    TEST_PASS();
}

static void test_inc_dec(void) {
    TEST_START("inc/dec");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 41, OP_INC, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(42, v.num, "41 + 1 = 42");

    /* Test dec */
    init_test_context(&tc);
    uint8_t code2[] = {OP_PUSH_INT, 43, OP_DEC, OP_HALT};
    vm_load_bytecode(&tc.ctx, code2, sizeof(code2));

    err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(42, v.num, "43 - 1 = 42");

    TEST_PASS();
}

/* ========================================================================== */
/* Comparison Tests */
/* ========================================================================== */

static void test_eq(void) {
    TEST_START("eq");
    TestContext tc;
    init_test_context(&tc);

    /* Test equal */
    uint8_t code1[] = {OP_PUSH_INT, 42, OP_PUSH_INT, 42, OP_EQ, OP_HALT};
    vm_load_bytecode(&tc.ctx, code1, sizeof(code1));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(VTAG_BOOLEAN, v.tag, "result should be boolean");
    ASSERT_EQ(1, v.num, "42 == 42 should be true");

    /* Test not equal */
    init_test_context(&tc);
    uint8_t code2[] = {OP_PUSH_INT, 42, OP_PUSH_INT, 43, OP_EQ, OP_HALT};
    vm_load_bytecode(&tc.ctx, code2, sizeof(code2));

    err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(0, v.num, "42 == 43 should be false");

    TEST_PASS();
}

static void test_lt(void) {
    TEST_START("lt");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 10, OP_PUSH_INT, 20, OP_LT, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(VTAG_BOOLEAN, v.tag, "result should be boolean");
    ASSERT_EQ(1, v.num, "10 < 20 should be true");

    TEST_PASS();
}

static void test_gt(void) {
    TEST_START("gt");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_INT, 20, OP_PUSH_INT, 10, OP_GT, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(1, v.num, "20 > 10 should be true");

    TEST_PASS();
}

/* ========================================================================== */
/* Boolean Tests */
/* ========================================================================== */

static void test_and(void) {
    TEST_START("and");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_TRUE, OP_PUSH_TRUE, OP_AND, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(VTAG_BOOLEAN, v.tag, "result should be boolean");
    ASSERT_EQ(1, v.num, "true && true = true");

    /* Test false case */
    init_test_context(&tc);
    uint8_t code2[] = {OP_PUSH_TRUE, OP_PUSH_FALSE, OP_AND, OP_HALT};
    vm_load_bytecode(&tc.ctx, code2, sizeof(code2));

    err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(0, v.num, "true && false = false");

    TEST_PASS();
}

static void test_or(void) {
    TEST_START("or");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_FALSE, OP_PUSH_TRUE, OP_OR, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(1, v.num, "false || true = true");

    TEST_PASS();
}

static void test_not(void) {
    TEST_START("not");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_PUSH_TRUE, OP_NOT, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_NONE, err, "execution error");

    Value v;
    vm_peek_stack(&tc.ctx, 0, &v);
    ASSERT_EQ(VTAG_BOOLEAN, v.tag, "result should be boolean");
    ASSERT_EQ(0, v.num, "!true = false");

    TEST_PASS();
}

/* ========================================================================== */
/* Error Handling Tests */
/* ========================================================================== */

static void test_stack_underflow(void) {
    TEST_START("stack underflow");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {OP_POP, OP_HALT};
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_STACK_UNDERFLOW, err, "should error on underflow");

    TEST_PASS();
}

static void test_invalid_opcode(void) {
    TEST_START("invalid opcode");
    TestContext tc;
    init_test_context(&tc);

    uint8_t code[] = {150, OP_HALT};  /* 150 is unused */
    vm_load_bytecode(&tc.ctx, code, sizeof(code));

    int err = vm_execute(&tc.ctx);
    ASSERT_EQ(ERR_INVALID_OPCODE, err, "should error on invalid opcode");

    TEST_PASS();
}

/* ========================================================================== */
/* Main */
/* ========================================================================== */

int main(void) {
    printf("DTRules NASM VM Test Suite\n");
    printf("==========================\n\n");

    printf("Stack Operations:\n");
    test_push_int();
    test_push_zero_one();
    test_push_bool();
    test_push_null();
    test_pop();
    test_dup();
    test_swap();

    printf("\nArithmetic Operations:\n");
    test_add();
    test_sub();
    test_mul();
    test_div();
    test_div_zero();
    test_mod();
    test_neg();
    test_abs();
    test_inc_dec();

    printf("\nComparison Operations:\n");
    test_eq();
    test_lt();
    test_gt();

    printf("\nBoolean Operations:\n");
    test_and();
    test_or();
    test_not();

    printf("\nError Handling:\n");
    test_stack_underflow();
    test_invalid_opcode();

    printf("\n==========================\n");
    printf("Results: %d/%d tests passed\n", tests_passed, tests_run);

    return (tests_passed == tests_run) ? 0 : 1;
}
