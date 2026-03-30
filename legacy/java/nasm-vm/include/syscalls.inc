; syscalls.inc - Linux x86-64 syscall numbers
; DTRules NASM VM Implementation
; Direct kernel interface (no libc)

%ifndef SYSCALLS_INC
%define SYSCALLS_INC

; File operations
SYS_READ        equ 0
SYS_WRITE       equ 1
SYS_OPEN        equ 2
SYS_CLOSE       equ 3
SYS_LSEEK       equ 8

; Memory operations
SYS_MMAP        equ 9
SYS_MPROTECT    equ 10
SYS_MUNMAP      equ 11
SYS_BRK         equ 12

; Process control
SYS_EXIT        equ 60
SYS_EXIT_GROUP  equ 231

; File flags for open
O_RDONLY        equ 0
O_WRONLY        equ 1
O_RDWR          equ 2
O_CREAT         equ 64
O_TRUNC         equ 512

; mmap protection flags
PROT_NONE       equ 0
PROT_READ       equ 1
PROT_WRITE      equ 2
PROT_EXEC       equ 4

; mmap flags
MAP_SHARED      equ 1
MAP_PRIVATE     equ 2
MAP_ANONYMOUS   equ 32
MAP_FIXED       equ 16

; Standard file descriptors
STDIN           equ 0
STDOUT          equ 1
STDERR          equ 2

; Seek whence values
SEEK_SET        equ 0
SEEK_CUR        equ 1
SEEK_END        equ 2

%endif ; SYSCALLS_INC
