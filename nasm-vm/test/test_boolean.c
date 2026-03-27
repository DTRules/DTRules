/*
 * DTRules NASM VM Boolean Operations Test
 * Copyright 2024 Paul Snow
 * Licensed under Apache License 2.0
 *
 * Tests for boolean operations:
 *   - OP_AND, OP_OR, OP_NOT, OP_XOR
 *   - OP_ISNULL, OP_NOTNULL, OP_BEQ
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

/* Value type tags - must match vm_constants.inc */
#define VTAG_NULL      0
#define VTAG_INTEGER   1
#define VTAG_DOUBLE    2
#define VTAG_BOOLEAN   3
#define VTAG_STRING    4
#define VTAG_NAME      5
#define VTAG_ARRAY     6
#define VTAG_ENTITY    7
#define VTAG_OBJECT    8

/* Opcodes - must match vm_constants.inc */
#define OP_NOP         0
#define OP_POP         3
#define OP_DUP         4
#define OP_SWAP        5
#define OP_AND         40
#define OP_OR          41
#define OP_NOT         42
#define OP_XOR         43
#define OP_ISNULL      44
#define OP_NOTNULL     45
#define OP_BEQ         46
#define OP_PUSH_TRUE   100
#define OP_PUSH_FALSE  101
#define OP_PUSH_NULL   102
#define OP_PUSH_ZERO   103
#define OP_PUSH_ONE    104
#define OP_HALT        255

/* Error codes - must match vm_constants.inc */
#define ERR_NONE             0
#define ERR_STACK_OVERFLOW   1
#define ERR_STACK_UNDERFLOW  2
#define ERR_INVALID_OPCODE   3
#define ERR_TYPE_MISMATCH    4
#define ERR_DIV_ZERO         5
#define ERR_OUT_OF_BOUNDS    6
#define ERR_NULL_PTR         7
#define ERR_NOT_IMPLEMENTED  8

/* Value structure - 24 bytes, matches vm_state.inc */
typedef struct {
    uint8_t  tag;
    uint8_t  padding[7];
    int64_t  num;
    void    *ptr;
} Value;

#define VALUE_SIZE 24

/* VMContext structure - must match vm_state.inc exactly */
typedef struct {
    /* Stack pointers */
    void    *data_sp;
    void    *entity_sp;
    void    *ctrl_sp;

    /* Stack bases */
    void    *data_base;
    void    *entity_base;
    void    *ctrl_base;

    /* Stack limits */
    void    *data_limit;
    void    *entity_limit;
    void    *ctrl_limit;

    /* Bytecode execution */
    void    *bytecode;
    uint64_t bytecode_len;
    uint64_t pc;

    /* Constant pools */
    void    *constants;
    uint64_t const_count;
    void    *names;
    uint64_t name_count;

    /* Jump table */
    void    *jump_table;

    /* State flags */
    uint32_t state_flags;
    uint32_t error_code;

    /* Frame management */
    uint64_t current_frame;
    void    *frames;
    uint64_t frame_count;

    /* Trace buffer */
    void    *trace_buf;
    void    *trace_pos;
    void    *trace_limit;

    /* Session reference */
    void    *session;

    /* Operator table */
    void    *op_table;
    uint64_t op_count;

    /* Statistics */
    uint64_t insn_count;
    uint64_t cycle_count;
} VMContext;

/* External functions from assembly */
extern int vm_execute(VMContext *ctx);
extern void vm_init_context(VMContext *ctx,
                           void *data_stack, size_t data_size,
                           void *entity_stack, size_t entity_size,
                           void *ctrl_stack, size_t ctrl_size);
extern void vm_load_bytecode(VMContext *ctx, void *bytecode, size_t len);
extern void vm_set_jump_table(VMContext *ctx, void **jump_table);
extern size_t vm_get_stack_depth(VMContext *ctx);
extern int vm_peek_stack(VMContext *ctx, size_t index, Value *out);
extern uint64_t vm_get_insn_count(VMContext *ctx);

/* External jump table from assembly */
extern void *vm_jump_table[];

/* Test counters */
static int tests_passed = 0;
static int tests_failed = 0;

/* Helper to run bytecode and check result */
static int run_test(const char *name, uint8_t *bytecode, size_t len,
                   int expected_result, int expected_depth,
                   uint8_t expected_tag, int64_t expected_num)
{
    VMContext ctx;
    char data_stack[4096];
    char entity_stack[1024];
    char ctrl_stack[1024];
    Value result;

    /* Initialize context */
    vm_init_context(&ctx,
                   data_stack, sizeof(data_stack),
                   entity_stack, sizeof(entity_stack),
                   ctrl_stack, sizeof(ctrl_stack));
    vm_load_bytecode(&ctx, bytecode, len);
    vm_set_jump_table(&ctx, vm_jump_table);

    /* Execute */
    int rc = vm_execute(&ctx);

    /* Check results */
    int passed = 1;

    if (rc != expected_result) {
        printf("FAIL %s: expected rc=%d, got rc=%d (error_code=%d)\n",
               name, expected_result, rc, ctx.error_code);
        passed = 0;
    }

    size_t depth = vm_get_stack_depth(&ctx);
    if ((int)depth != expected_depth) {
        printf("FAIL %s: expected depth=%d, got depth=%zu\n",
               name, expected_depth, depth);
        passed = 0;
    }

    if (expected_depth > 0 && passed) {
        if (vm_peek_stack(&ctx, 0, &result) != 0) {
            printf("FAIL %s: could not peek stack\n", name);
            passed = 0;
        } else {
            if (result.tag != expected_tag) {
                printf("FAIL %s: expected tag=%d, got tag=%d\n",
                       name, expected_tag, result.tag);
                passed = 0;
            }
            if (result.num != expected_num) {
                printf("FAIL %s: expected num=%ld, got num=%ld\n",
                       name, (long)expected_num, (long)result.num);
                passed = 0;
            }
        }
    }

    if (passed) {
        printf("PASS %s\n", name);
        tests_passed++;
    } else {
        tests_failed++;
    }

    return passed;
}

/* Test cases */

void test_push_true(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_HALT };
    run_test("push_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_push_false(void)
{
    uint8_t bc[] = { OP_PUSH_FALSE, OP_HALT };
    run_test("push_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_push_null(void)
{
    uint8_t bc[] = { OP_PUSH_NULL, OP_HALT };
    run_test("push_null", bc, sizeof(bc), ERR_NONE, 1, VTAG_NULL, 0);
}

void test_and_true_true(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_TRUE, OP_AND, OP_HALT };
    run_test("and_true_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_and_true_false(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_FALSE, OP_AND, OP_HALT };
    run_test("and_true_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_and_false_true(void)
{
    uint8_t bc[] = { OP_PUSH_FALSE, OP_PUSH_TRUE, OP_AND, OP_HALT };
    run_test("and_false_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_and_false_false(void)
{
    uint8_t bc[] = { OP_PUSH_FALSE, OP_PUSH_FALSE, OP_AND, OP_HALT };
    run_test("and_false_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_or_true_true(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_TRUE, OP_OR, OP_HALT };
    run_test("or_true_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_or_true_false(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_FALSE, OP_OR, OP_HALT };
    run_test("or_true_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_or_false_true(void)
{
    uint8_t bc[] = { OP_PUSH_FALSE, OP_PUSH_TRUE, OP_OR, OP_HALT };
    run_test("or_false_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_or_false_false(void)
{
    uint8_t bc[] = { OP_PUSH_FALSE, OP_PUSH_FALSE, OP_OR, OP_HALT };
    run_test("or_false_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_not_true(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_NOT, OP_HALT };
    run_test("not_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_not_false(void)
{
    uint8_t bc[] = { OP_PUSH_FALSE, OP_NOT, OP_HALT };
    run_test("not_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_xor_true_true(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_TRUE, OP_XOR, OP_HALT };
    run_test("xor_true_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_xor_true_false(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_FALSE, OP_XOR, OP_HALT };
    run_test("xor_true_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_xor_false_true(void)
{
    uint8_t bc[] = { OP_PUSH_FALSE, OP_PUSH_TRUE, OP_XOR, OP_HALT };
    run_test("xor_false_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_xor_false_false(void)
{
    uint8_t bc[] = { OP_PUSH_FALSE, OP_PUSH_FALSE, OP_XOR, OP_HALT };
    run_test("xor_false_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_isnull_null(void)
{
    uint8_t bc[] = { OP_PUSH_NULL, OP_ISNULL, OP_HALT };
    run_test("isnull_null", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_isnull_true(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_ISNULL, OP_HALT };
    run_test("isnull_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_isnull_integer(void)
{
    uint8_t bc[] = { OP_PUSH_ZERO, OP_ISNULL, OP_HALT };
    run_test("isnull_integer", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_notnull_null(void)
{
    uint8_t bc[] = { OP_PUSH_NULL, OP_NOTNULL, OP_HALT };
    run_test("notnull_null", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_notnull_true(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_NOTNULL, OP_HALT };
    run_test("notnull_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_notnull_integer(void)
{
    uint8_t bc[] = { OP_PUSH_ONE, OP_NOTNULL, OP_HALT };
    run_test("notnull_integer", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_beq_true_true(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_TRUE, OP_BEQ, OP_HALT };
    run_test("beq_true_true", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_beq_true_false(void)
{
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_FALSE, OP_BEQ, OP_HALT };
    run_test("beq_true_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 0);
}

void test_beq_false_false(void)
{
    uint8_t bc[] = { OP_PUSH_FALSE, OP_PUSH_FALSE, OP_BEQ, OP_HALT };
    run_test("beq_false_false", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_stack_underflow_and(void)
{
    /* On error, we don't check the stack depth - just verify the error code */
    VMContext ctx;
    char data_stack[4096];
    char entity_stack[1024];
    char ctrl_stack[1024];
    uint8_t bc[] = { OP_PUSH_TRUE, OP_AND, OP_HALT };

    vm_init_context(&ctx,
                   data_stack, sizeof(data_stack),
                   entity_stack, sizeof(entity_stack),
                   ctrl_stack, sizeof(ctrl_stack));
    vm_load_bytecode(&ctx, bc, sizeof(bc));
    vm_set_jump_table(&ctx, vm_jump_table);

    int rc = vm_execute(&ctx);
    if (rc == ERR_STACK_UNDERFLOW) {
        printf("PASS stack_underflow_and\n");
        tests_passed++;
    } else {
        printf("FAIL stack_underflow_and: expected rc=%d, got rc=%d\n",
               ERR_STACK_UNDERFLOW, rc);
        tests_failed++;
    }
}

void test_stack_underflow_not(void)
{
    /* On error, we don't check the stack depth - just verify the error code */
    VMContext ctx;
    char data_stack[4096];
    char entity_stack[1024];
    char ctrl_stack[1024];
    uint8_t bc[] = { OP_NOT, OP_HALT };

    vm_init_context(&ctx,
                   data_stack, sizeof(data_stack),
                   entity_stack, sizeof(entity_stack),
                   ctrl_stack, sizeof(ctrl_stack));
    vm_load_bytecode(&ctx, bc, sizeof(bc));
    vm_set_jump_table(&ctx, vm_jump_table);

    int rc = vm_execute(&ctx);
    if (rc == ERR_STACK_UNDERFLOW) {
        printf("PASS stack_underflow_not\n");
        tests_passed++;
    } else {
        printf("FAIL stack_underflow_not: expected rc=%d, got rc=%d\n",
               ERR_STACK_UNDERFLOW, rc);
        tests_failed++;
    }
}

void test_complex_expression(void)
{
    /* (true AND false) OR (NOT false) = false OR true = true */
    uint8_t bc[] = {
        OP_PUSH_TRUE, OP_PUSH_FALSE, OP_AND,    /* -> false */
        OP_PUSH_FALSE, OP_NOT,                   /* -> true */
        OP_OR,                                    /* false OR true -> true */
        OP_HALT
    };
    run_test("complex_expression", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

void test_dup_and_swap(void)
{
    /* Push true, dup, should have two trues */
    uint8_t bc[] = { OP_PUSH_TRUE, OP_DUP, OP_HALT };
    run_test("dup", bc, sizeof(bc), ERR_NONE, 2, VTAG_BOOLEAN, 1);
}

void test_swap(void)
{
    /* Push true, push false, swap - top should be true */
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_FALSE, OP_SWAP, OP_HALT };
    run_test("swap", bc, sizeof(bc), ERR_NONE, 2, VTAG_BOOLEAN, 1);
}

void test_pop(void)
{
    /* Push true, push false, pop - should have one value (true) */
    uint8_t bc[] = { OP_PUSH_TRUE, OP_PUSH_FALSE, OP_POP, OP_HALT };
    run_test("pop", bc, sizeof(bc), ERR_NONE, 1, VTAG_BOOLEAN, 1);
}

int main(void)
{
    printf("DTRules NASM VM Boolean Operations Test\n");
    printf("========================================\n\n");

    /* Basic push tests */
    test_push_true();
    test_push_false();
    test_push_null();

    /* AND tests */
    test_and_true_true();
    test_and_true_false();
    test_and_false_true();
    test_and_false_false();

    /* OR tests */
    test_or_true_true();
    test_or_true_false();
    test_or_false_true();
    test_or_false_false();

    /* NOT tests */
    test_not_true();
    test_not_false();

    /* XOR tests */
    test_xor_true_true();
    test_xor_true_false();
    test_xor_false_true();
    test_xor_false_false();

    /* ISNULL tests */
    test_isnull_null();
    test_isnull_true();
    test_isnull_integer();

    /* NOTNULL tests */
    test_notnull_null();
    test_notnull_true();
    test_notnull_integer();

    /* BEQ tests */
    test_beq_true_true();
    test_beq_true_false();
    test_beq_false_false();

    /* Error handling tests */
    test_stack_underflow_and();
    test_stack_underflow_not();

    /* Complex expression test */
    test_complex_expression();

    /* Stack operation tests */
    test_dup_and_swap();
    test_swap();
    test_pop();

    printf("\n========================================\n");
    printf("Results: %d passed, %d failed\n", tests_passed, tests_failed);

    return tests_failed > 0 ? 1 : 0;
}
