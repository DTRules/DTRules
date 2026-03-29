; date.asm - Date operations
; DTRules Assembly Implementation
; Dates stored as Unix timestamp (seconds since 1970-01-01)

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc
extern string_data
extern string_length

;-----------------------------------------------------------------------------
; Date constants
;-----------------------------------------------------------------------------
SECONDS_PER_DAY     equ 86400
SECONDS_PER_HOUR    equ 3600
SECONDS_PER_MINUTE  equ 60

; Days in each month (non-leap year)
section .data
    align 8
    days_in_month: dd 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31
    ; Cumulative days before each month (non-leap)
    days_before_month: dd 0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334

section .text

;-----------------------------------------------------------------------------
; is_leap_year - Check if year is a leap year
; Input: rdi = year
; Output: rax = 1 if leap year, 0 otherwise
;-----------------------------------------------------------------------------
is_leap_year:
    ; Leap year: divisible by 4, except centuries unless divisible by 400
    mov rax, rdi
    test rax, 3             ; Quick check: not divisible by 4
    jnz .not_leap

    ; Divisible by 4, check for century
    mov rcx, 100
    xor edx, edx
    div rcx
    test rdx, rdx
    jnz .is_leap            ; Not century, is leap

    ; Century, check divisibility by 400
    mov rax, rdi
    mov rcx, 400
    xor edx, edx
    div rcx
    test rdx, rdx
    jnz .not_leap

.is_leap:
    mov eax, 1
    ret

.not_leap:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; days_in_year - Get number of days in year
; Input: rdi = year
; Output: rax = 365 or 366
;-----------------------------------------------------------------------------
days_in_year:
    call is_leap_year
    add rax, 365
    ret

;-----------------------------------------------------------------------------
; get_days_in_month - Get days in a specific month
; Input: rdi = year, rsi = month (1-12)
; Output: rax = days in month
;-----------------------------------------------------------------------------
get_days_in_month:
    push rbx

    ; Get base days
    dec rsi                 ; Convert to 0-indexed
    lea rax, [days_in_month]
    mov eax, [rax + rsi * 4]
    mov rbx, rax

    ; Check for February in leap year
    cmp rsi, 1              ; February (0-indexed)
    jne .done

    push rdi
    call is_leap_year
    add rbx, rax
    pop rdi

.done:
    mov rax, rbx
    pop rbx
    ret

;-----------------------------------------------------------------------------
; timestamp_to_ymd - Convert Unix timestamp to year/month/day
; Input: rdi = timestamp
; Output: rax = year, rcx = month (1-12), rdx = day (1-31)
;-----------------------------------------------------------------------------
timestamp_to_ymd:
    push rbx
    push r12
    push r13
    push r14
    push r15

    ; Get days since epoch
    mov rax, rdi
    mov rcx, SECONDS_PER_DAY
    xor edx, edx
    div rcx
    mov r12, rax            ; days since epoch

    ; Start from 1970
    mov r13, 1970           ; year

    ; Find year
.year_loop:
    mov rdi, r13
    call days_in_year
    cmp r12, rax
    jb .found_year
    sub r12, rax
    inc r13
    jmp .year_loop

.found_year:
    ; r13 = year, r12 = day of year (0-based)

    ; Find month
    mov r14, 1              ; month (1-based)

.month_loop:
    mov rdi, r13
    mov rsi, r14
    call get_days_in_month
    cmp r12, rax
    jb .found_month
    sub r12, rax
    inc r14
    cmp r14, 12
    jbe .month_loop

.found_month:
    ; r14 = month, r12 = day of month (0-based)
    inc r12                 ; Convert to 1-based

    mov rax, r13            ; year
    mov rcx, r14            ; month
    mov rdx, r12            ; day

    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; ymd_to_timestamp - Convert year/month/day to Unix timestamp
; Input: rdi = year, rsi = month (1-12), rdx = day (1-31)
; Output: rax = timestamp
;-----------------------------------------------------------------------------
ymd_to_timestamp:
    push rbx
    push r12
    push r13
    push r14
    push r15

    mov r12, rdi            ; year
    mov r13, rsi            ; month
    mov r14, rdx            ; day

    ; Calculate days since epoch
    xor r15, r15            ; total days

    ; Add days for complete years since 1970
    mov rbx, 1970
.year_count:
    cmp rbx, r12
    jge .years_done
    mov rdi, rbx
    call days_in_year
    add r15, rax
    inc rbx
    jmp .year_count

.years_done:
    ; Add days for complete months in current year
    mov rbx, 1
.month_count:
    cmp rbx, r13
    jge .months_done
    mov rdi, r12
    mov rsi, rbx
    call get_days_in_month
    add r15, rax
    inc rbx
    jmp .month_count

.months_done:
    ; Add days in current month
    add r15, r14
    dec r15                 ; Convert day from 1-based to 0-based

    ; Convert to seconds
    imul rax, r15, SECONDS_PER_DAY

    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_create - Create a date from year/month/day
; Input: rdi = year, rsi = month, rdx = day
; Output: rax = timestamp
;-----------------------------------------------------------------------------
global date_create
date_create:
    jmp ymd_to_timestamp

;-----------------------------------------------------------------------------
; date_get_year - Extract year from timestamp
; Input: rdi = timestamp
; Output: rax = year
;-----------------------------------------------------------------------------
global date_get_year
date_get_year:
    push rbx
    call timestamp_to_ymd
    pop rbx
    ret                     ; rax already contains year

;-----------------------------------------------------------------------------
; date_get_month - Extract month from timestamp
; Input: rdi = timestamp
; Output: rax = month (1-12)
;-----------------------------------------------------------------------------
global date_get_month
date_get_month:
    push rbx
    call timestamp_to_ymd
    mov rax, rcx
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_get_day - Extract day from timestamp
; Input: rdi = timestamp
; Output: rax = day (1-31)
;-----------------------------------------------------------------------------
global date_get_day
date_get_day:
    push rbx
    call timestamp_to_ymd
    mov rax, rdx
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_add_days - Add days to a date
; Input: rdi = timestamp, rsi = days (can be negative)
; Output: rax = new timestamp
;-----------------------------------------------------------------------------
global date_add_days
date_add_days:
    imul rax, rsi, SECONDS_PER_DAY
    add rax, rdi
    ret

;-----------------------------------------------------------------------------
; date_add_months - Add months to a date
; Input: rdi = timestamp, rsi = months (can be negative)
; Output: rax = new timestamp
;-----------------------------------------------------------------------------
global date_add_months
date_add_months:
    push rbx
    push r12
    push r13
    push r14

    mov r12, rsi            ; months to add

    ; Get current date components
    call timestamp_to_ymd
    mov r13, rax            ; year
    mov r14, rcx            ; month
    mov rbx, rdx            ; day

    ; Add months
    add r14, r12

    ; Handle month overflow/underflow
.adjust_months:
    cmp r14, 1
    jl .underflow
    cmp r14, 12
    jg .overflow
    jmp .adjusted

.overflow:
    sub r14, 12
    inc r13
    jmp .adjust_months

.underflow:
    add r14, 12
    dec r13
    jmp .adjust_months

.adjusted:
    ; Clamp day to valid range for new month
    mov rdi, r13
    mov rsi, r14
    call get_days_in_month
    cmp rbx, rax
    jle .day_ok
    mov rbx, rax            ; Clamp to last day of month

.day_ok:
    ; Convert back to timestamp
    mov rdi, r13
    mov rsi, r14
    mov rdx, rbx
    call ymd_to_timestamp

    pop r14
    pop r13
    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_add_years - Add years to a date
; Input: rdi = timestamp, rsi = years (can be negative)
; Output: rax = new timestamp
;-----------------------------------------------------------------------------
global date_add_years
date_add_years:
    push rbx
    push r12
    push r13
    push r14

    mov r12, rsi            ; years to add

    ; Get current date components
    call timestamp_to_ymd
    mov r13, rax            ; year
    mov r14, rcx            ; month
    mov rbx, rdx            ; day

    ; Add years
    add r13, r12

    ; Handle Feb 29 -> Feb 28 for non-leap years
    cmp r14, 2
    jne .day_ok
    cmp rbx, 29
    jne .day_ok

    mov rdi, r13
    call is_leap_year
    test rax, rax
    jnz .day_ok
    mov rbx, 28             ; Feb 28 in non-leap year

.day_ok:
    mov rdi, r13
    mov rsi, r14
    mov rdx, rbx
    call ymd_to_timestamp

    pop r14
    pop r13
    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_days_between - Calculate days between two dates
; Input: rdi = timestamp1, rsi = timestamp2
; Output: rax = days (timestamp2 - timestamp1) / 86400
;-----------------------------------------------------------------------------
global date_days_between
date_days_between:
    mov rax, rsi
    sub rax, rdi
    mov rcx, SECONDS_PER_DAY
    cqo                     ; Sign extend rax to rdx:rax
    idiv rcx
    ret

;-----------------------------------------------------------------------------
; date_months_between - Calculate months between two dates
; Input: rdi = timestamp1, rsi = timestamp2
; Output: rax = approximate months
;-----------------------------------------------------------------------------
global date_months_between
date_months_between:
    push rbx
    push r12
    push r13
    push r14
    push r15

    mov r12, rdi            ; timestamp1
    mov r13, rsi            ; timestamp2

    ; Get date components for both
    mov rdi, r12
    call timestamp_to_ymd
    mov r14, rax            ; year1
    shl r14, 8
    or r14, rcx             ; year1 << 8 | month1

    mov rdi, r13
    call timestamp_to_ymd
    mov r15, rax            ; year2
    shl r15, 8
    or r15, rcx             ; year2 << 8 | month2

    ; Calculate: (year2 - year1) * 12 + (month2 - month1)
    mov rax, r15
    shr rax, 8              ; year2
    mov rcx, r14
    shr rcx, 8              ; year1
    sub rax, rcx            ; year_diff
    imul rax, 12

    movzx rcx, r15b         ; month2
    movzx rdx, r14b         ; month1
    sub rcx, rdx
    add rax, rcx

    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_years_between - Calculate years between two dates
; Input: rdi = timestamp1, rsi = timestamp2
; Output: rax = years
;-----------------------------------------------------------------------------
global date_years_between
date_years_between:
    push rbx
    push r12

    mov r12, rsi            ; save timestamp2

    call date_get_year
    mov rbx, rax            ; year1

    mov rdi, r12
    call date_get_year
    sub rax, rbx            ; year2 - year1

    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_compare - Compare two dates
; Input: rdi = timestamp1, rsi = timestamp2
; Output: rax = -1 if t1 < t2, 0 if equal, 1 if t1 > t2
;-----------------------------------------------------------------------------
global date_compare
date_compare:
    cmp rdi, rsi
    jl .less
    jg .greater
    xor eax, eax
    ret
.less:
    mov eax, -1
    ret
.greater:
    mov eax, 1
    ret

;-----------------------------------------------------------------------------
; date_first_of_month - Get first day of month for given date
; Input: rdi = timestamp
; Output: rax = timestamp of first day of that month
;-----------------------------------------------------------------------------
global date_first_of_month
date_first_of_month:
    push rbx
    push r12

    call timestamp_to_ymd
    mov r12, rax            ; year
    mov rbx, rcx            ; month

    mov rdi, r12
    mov rsi, rbx
    mov rdx, 1              ; day 1
    call ymd_to_timestamp

    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_first_of_year - Get first day of year for given date
; Input: rdi = timestamp
; Output: rax = timestamp of Jan 1 of that year
;-----------------------------------------------------------------------------
global date_first_of_year
date_first_of_year:
    push rbx

    call date_get_year
    mov rbx, rax

    mov rdi, rbx
    mov rsi, 1              ; January
    mov rdx, 1              ; Day 1
    call ymd_to_timestamp

    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_end_of_month - Get last day of month for given date
; Input: rdi = timestamp
; Output: rax = timestamp of last day of that month
;-----------------------------------------------------------------------------
global date_end_of_month
date_end_of_month:
    push rbx
    push r12
    push r13

    call timestamp_to_ymd
    mov r12, rax            ; year
    mov r13, rcx            ; month

    ; Get days in month
    mov rdi, r12
    mov rsi, r13
    call get_days_in_month
    mov rbx, rax

    mov rdi, r12
    mov rsi, r13
    mov rdx, rbx            ; last day
    call ymd_to_timestamp

    pop r13
    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_days_in_year - Get number of days in the year of the given date
; Input: rdi = timestamp
; Output: rax = 365 or 366
;-----------------------------------------------------------------------------
global date_days_in_year
date_days_in_year:
    call date_get_year
    mov rdi, rax
    jmp days_in_year

;-----------------------------------------------------------------------------
; date_days_in_month - Get number of days in the month of the given date
; Input: rdi = timestamp
; Output: rax = 28-31
;-----------------------------------------------------------------------------
global date_days_in_month
date_days_in_month:
    push rbx
    push r12

    call timestamp_to_ymd
    mov r12, rax            ; year
    mov rbx, rcx            ; month

    mov rdi, r12
    mov rsi, rbx
    call get_days_in_month

    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; date_parse - Parse date from string (YYYY-MM-DD format)
; Input: rdi = pointer to string
; Output: rax = timestamp, or 0 on error
;-----------------------------------------------------------------------------
global date_parse
date_parse:
    push rbx
    push r12
    push r13
    push r14

    mov r12, rdi            ; save string pointer

    ; Get string data pointer
    call string_data
    mov rbx, rax            ; data pointer

    ; Parse year (4 digits)
    xor r13, r13            ; year accumulator
    mov ecx, 4
.parse_year:
    movzx eax, byte [rbx]
    sub al, '0'
    cmp al, 9
    ja .error
    imul r13, 10
    add r13, rax
    inc rbx
    dec ecx
    jnz .parse_year

    ; Skip separator
    cmp byte [rbx], '-'
    jne .error
    inc rbx

    ; Parse month (2 digits)
    xor r14, r14
    mov ecx, 2
.parse_month:
    movzx eax, byte [rbx]
    sub al, '0'
    cmp al, 9
    ja .error
    imul r14, 10
    add r14, rax
    inc rbx
    dec ecx
    jnz .parse_month

    ; Skip separator
    cmp byte [rbx], '-'
    jne .error
    inc rbx

    ; Parse day (2 digits)
    xor eax, eax
    mov ecx, 2
.parse_day:
    movzx edx, byte [rbx]
    sub dl, '0'
    cmp dl, 9
    ja .error
    imul eax, 10
    add eax, edx
    inc rbx
    dec ecx
    jnz .parse_day

    ; Validate ranges
    cmp r14, 1
    jl .error
    cmp r14, 12
    jg .error
    cmp rax, 1
    jl .error
    cmp rax, 31
    jg .error

    ; Convert to timestamp
    mov rdx, rax
    mov rdi, r13
    mov rsi, r14
    call ymd_to_timestamp

    pop r14
    pop r13
    pop r12
    pop rbx
    ret

.error:
    xor eax, eax
    pop r14
    pop r13
    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; time_get_timestamp - Get current Unix timestamp
; Output: rax = current timestamp in seconds
;-----------------------------------------------------------------------------
global time_get_timestamp
time_get_timestamp:
    ; Use clock_gettime syscall
    sub rsp, 16             ; timespec struct
    mov rdi, 0              ; CLOCK_REALTIME
    mov rsi, rsp
    mov eax, 228            ; SYS_clock_gettime
    syscall

    mov rax, [rsp]          ; tv_sec
    add rsp, 16
    ret
