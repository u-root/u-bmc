/*
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with this program; if not, write to the Free Software
 *  Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA
 */
/*
 * Board specific setup info
 *
 ******************************************************************************
 * ASPEED Technology Inc.
 * AST2300/AST2400 DDR2/DDR3 SDRAM controller initialization and calibration sequence
 *
 * Gary Hsu, <gary_hsu@aspeedtech.com>
 *
 * Release date: 2015.02.09 formal release for SDK0.61
 *
 * Modified list from v0.23
 * EC1. Modify DQIDLY and DQSI-MCLK2X calibration algorithm
 * EC2. Remove pass 2 DQIDLY finetune process
 * EC3. Modify ECC code
 * EC4. Add AST2400 supporting
 * EC5. Add SPI timing calibration for AST2400
 * EC6. Remove AST2300-A0 PCI-e workaround
 * EC7. Add CK duty calibration for AST2400
 * EC8. Remove #define CONFIG_DRAM_UART_OUT, default has message output to UART5
 * EC9. Add DRAM size auto-detection
 * EC10. Add GPIO register clear when watchdog reboot (only for AST2400)
 * EC11. Move the "Solve ASPM" code position of AST2300 to avoid watchdog reset
 *
 * Modified list from v0.53
 * EC1. Add solution of LPC lock issue due to watchdog reset. (AP note A2300-11)
 *
 * Modified list from v0.56
 * EC1. Fix read DQS input mask window too late issue if DRAM's t_DQSCK is earlier too much
 *      (ex. Nanya NT5CB64M16FP)
 *      1. Change init value of MCR18[4] from '1' to '0'
 *      2. Add CBR4 code to finetune MCR18[4]
 *
 * Modified list from v0.59
 * EC1. Add DQS input gating window delay tuning (1/2 T) when CBR retry
 * EC2. Modify DLL1 MAdj = 0x4C
 *
 * Modified list from v0.60
 * EC1. Modify DDR2 init preliminary size to 1Gbit, and BL=4.
 *
 * Modified list from v0.61
 * EC1. Set for wide screen supporting, 0x1e6e2040[0] = 1
 *
 * Modified list from v0.62
 * EC1. Clear MCR04[10] = 0 before doing DRAM initial
 *
 * Optional define variable
 * 1. DRAM Speed             //
 *    CONFIG_DRAM_336        // 336MHz (DDR-667)
 *    CONFIG_DRAM_408        // 408MHz (DDR-800) (default)
 * 2. ECC Function enable
 *    CONFIG_DRAM_ECC        // define to enable ECC function
 *                           // when enabled, must define the ECC protected memory size at 0x1e6e0054
 * 3. UART5 message output   //
 *    CONFIG_DRAM_UART_38400 // set the UART baud rate to 38400, default is 115200
 *    CONFIG_DRAM_UART_57600 // set the UART baud rate to 57600
 ******************************************************************************
 */

#include <config.h>
/******************************************************************************
 Calibration Macro Start
 Usable registers:
  r0, r1, r2, r3, r5, r6, r7, r8, r9, r10, r11
 ******************************************************************************/
/* PATTERN_TABLE,
   init_delay_timer,
   check_delay_timer,
   clear_delay_timer,
   record_dll2_pass_range,
   record_dll2_pass_range_h,
   are for DRAM calibration */

PATTERN_TABLE:
    .word   0xff00ff00
    .word   0xcc33cc33
    .word   0xaa55aa55
    .word   0x88778877
    .word   0x92cc4d6e       @ 5
    .word   0x543d3cde
    .word   0xf1e843c7
    .word   0x7c61d253
    .word   0x00000000       @ 8

    .macro init_delay_timer
    ldr r0, =0x1e782024                          @ Set Timer3 Reload
    str r2, [r0]

    ldr r0, =0x1e6c0038                          @ Clear Timer3 ISR
    ldr r1, =0x00040000
    str r1, [r0]

    ldr r0, =0x1e782030                          @ Enable Timer3
    ldr r1, [r0]
    mov r2, #7
    orr r1, r1, r2, lsl #8
    str r1, [r0]

    ldr r0, =0x1e6c0090                          @ Check ISR for Timer3 timeout
    .endm

    .macro check_delay_timer
    ldr r1, [r0]
    bic r1, r1, #0xFFFBFFFF
    mov r2, r1, lsr #18
    cmp r2, #0x01
    .endm

    .macro clear_delay_timer
    ldr r0, =0x1e782030                          @ Disable Timer3
    ldr r1, [r0]
    bic r1, r1, #0x00000F00
    str r1, [r0]

    ldr r0, =0x1e6c0038                          @ Clear Timer3 ISR
    ldr r1, =0x00040000
    str r1, [r0]
    .endm

    .macro record_dll2_pass_range
    ldr                 r1, [r0]
    bic                 r2, r1, #0xFFFFFF00
    cmp                 r2, r3                   @ record min
    bicgt               r1, r1, #0x000000FF
    orrgt               r1, r1, r3
    bic                 r2, r1, #0xFFFF00FF
    cmp                 r3, r2, lsr #8           @ record max
    bicgt               r1, r1, #0x0000FF00
    orrgt               r1, r1, r3, lsl #8
    str                 r1, [r0]
    .endm

    .macro record_dll2_pass_range_h
    ldr                 r1, [r0]
    bic                 r2, r1, #0xFF00FFFF
    mov                 r2, r2, lsr #16
    cmp                 r2, r3                   @ record min
    bicgt               r1, r1, #0x00FF0000
    orrgt               r1, r1, r3, lsl #16
    bic                 r2, r1, #0x00FFFFFF
    cmp                 r3, r2, lsr #24          @ record max
    bicgt               r1, r1, #0xFF000000
    orrgt               r1, r1, r3, lsl #24
    str                 r1, [r0]
    .endm

    .macro init_spi_checksum
    ldr r0, =0x1e620084
    ldr r1, =0x20010000
    str r1, [r0]
    ldr r0, =0x1e62008C
    ldr r1, =0x20000200
    str r1, [r0]
    ldr r0, =0x1e620080
    ldr r1, =0x0000000D
    orr r2, r2, r7
    orr r1, r1, r2, lsl #8
    and r2, r6, #0xF
    orr r1, r1, r2, lsl #4
    str r1, [r0]
    ldr r0, =0x1e620008
    ldr r2, =0x00000800
    .endm

/******************************************************************************
 Calibration Macro End
 ******************************************************************************/
LPC_Patch:                                       @ load to SRAM base 0x1e720400
    str   r1, [r0]
    str   r3, [r2]
    bic   r1, r1, #0xFF
LPC_Patch_S1:
    subs  r5, r5, #0x01
    moveq pc, r8
    ldr   r3, [r2]
    tst   r3, #0x01
    movne pc, r8
    mov   pc, r7
LPC_Patch_S2:                                    @ load to SRAM base 0x1e720480
    str   r1, [r0]
    mov   pc, r9
LPC_Patch_E:

.globl lowlevel_init
lowlevel_init:

init_dram:
    /* save lr */
    mov r4, lr
/* Test - DRAM initial time */
    ldr r0, =0x1e782044
    ldr r1, =0xFFFFFFFF
    str r1, [r0]

    ldr r0, =0x1e782030
    ldr r1, [r0]
    bic r1, r1, #0x0000F000
    str r1, [r0]
    mov r2, #3
    orr r1, r1, r2, lsl #12
    str r1, [r0]
/* Test - DRAM initial time */

    /*Set Scratch register Bit 7 before initialize*/
    ldr r0, =0x1e6e2000
    ldr r1, =0x1688a8a8
    str r1, [r0]

    ldr r0, =0x1e6e2040
    ldr r1, [r0]
    orr r1, r1, #0x80
    str r1, [r0]

    /* Fix LPC lock issue for AST2300 */
    ldr r0, =0x1e6e207c                          @ Check AST2300
    ldr r1, [r0]
    mov r1, r1, lsr #24
    cmp r1, #0x01
    bne lpc_recover_end                          @ not match AST2300

    mov r3, #0x0
lpc_recover_check:
    ldr r0, =0x1e78900c                          @ check HICR3[4]=0x1
    ldr r1, [r0]
    tst r1, #0x10
    beq lpc_recover_end
    ldr r0, =0x1e789004                          @ check HICR1[7]=0x1
    ldr r1, [r0]
    tst r1, #0x80
    beq lpc_recover_end
    ldr r0, =0x1e7890a0                          @ check LHCR0[27:24]=0x6
    ldr r1, [r0]
    mov r1, r1, lsr #24
    and r1, r1, #0xF
    cmp r1, #0x06
    bne lpc_recover_end
    add r3, r3, #0x01
    cmp r3, #0x5                                 @ repeat 5 times
    ble lpc_recover_check

    mov r3, #0x0
lpc_recover_init:
    ldr r0, =0x1e7890a4                          @ set LHCR1[1:0]=0x0
    ldr r1, =0x00000000
    str r1, [r0]
    add r3, r3, #0x01
    cmp r3, #0x20
    bge lpc_recover_end
    ldr r1, [r0]
    tst r1, #0x01
    bne lpc_recover_init

    ldr r0, =0x1e7890b0                          @ set LHCR4[7:0]=0xFF
    ldr r1, =0x000000FF
    str r1, [r0]
    ldr r0, =0x1e7890b4                          @ set LHCR5[31:0]=0xFFFFFFFF
    ldr r1, =0xFFFFFFFF
    str r1, [r0]
    ldr r0, =0x1e7890b8                          @ set LHCR6[31:0]=0xFFFFFFFF
    str r1, [r0]

    adr r6, LPC_Patch
    adr r7, LPC_Patch_S2
    ldr r0, =0x1e720400
copy_lpc_patch_1:
    ldr r1, [r6]
    str r1, [r0]
    add r6, r6, #0x4
    add r0, r0, #0x4
    cmp r6, r7
    bne copy_lpc_patch_1

    adr r6, LPC_Patch_S2
    adr r7, LPC_Patch_E
    ldr r0, =0x1e720480
copy_lpc_patch_2:
    ldr r1, [r6]
    str r1, [r0]
    add r6, r6, #0x4
    add r0, r0, #0x4
    cmp r6, r7
    bne copy_lpc_patch_2

    ldr r0, =0x1e7890a0                          @ set LHCR0[31:0]=0xFFFFFF01
    ldr r1, =0xFFFFFF01
    add r2, r0, #0x4
    mov r3, #0x01
    mov r5, #0x10
    adr r9, lpc_recover_end
    adr r6, LPC_Patch
    adr r7, LPC_Patch_S1
    sub r6, r7, r6
    ldr r7, =0x1e720400
    add r7, r7, r6
    ldr r8, =0x1e720480
    ldr pc, =0x1e720400

lpc_recover_end:
    ldr r0, =0x1e7890a0                          @ set LHCR0[31:0]=0xFFFFFF00
    ldr r1, =0xFFFFFF00
    str r1, [r0]
    /* <END> Fix LPC lock issue for AST2300 */

    /* Check Scratch Register Bit 6 */
    ldr r0, =0x1e6e2040
    ldr r1, [r0]
    bic r1, r1, #0xFFFFFFBF
    mov r2, r1, lsr #6
    cmp r2, #0x01
    beq platform_exit

    ldr r2, =0x033103F1                          @ load PLL parameter for 24Mhz CLKIN (396:324)
/*  ldr r2, =0x019001F0                          @ load PLL parameter for 24Mhz CLKIN (408:336) */
    ldr r0, =0x1e6e207c                          @ Check Revision ID
    ldr r1, [r0]
    mov r1, r1, lsr #24
    cmp r1, #0x02
    bne set_MPLL                                 @ not match AST2400

    ldr r0, =0x1e6e2070                          @ Check CLKIN freq
    ldr r1, [r0]
    mov r1, r1, lsr #23
    tst r1, #0x01
    ldrne r2, =0x017001D0                        @ load PLL parameter for 25Mhz CLKIN (400:325)

set_MPLL:
    ldr r0, =0x1e6e2020                          @ M-PLL (DDR SDRAM) Frequency
    ldr r1, =0xFFFF
#if defined(CONFIG_DRAM_336)
    mov r2, r2, lsr #16
#endif
    and r1, r2, r1
    str r1, [r0]

/* Debug - UART console message */
    // Enable FIFO
    ldr r0, =0x1e784008
    mov r1, #0x1
    str r1, [r0]
    ldr r0, =0x1e78400c
    mov r1, #0x83
    str r1, [r0]

    ldr r0, =0x1e6e202c
    ldr r2, [r0]
    mov r2, r2, lsr #12
    tst r2, #0x01
    ldr r0, =0x1e784000
#if defined(CONFIG_DRAM_UART_38400)
    moveq r1, #0x27                              @ Baudrate 38400
    movne r1, #0x03                              @ Baudrate 38400 , div13
#elif defined(CONFIG_DRAM_UART_57600)
    moveq r1, #0x1A                              @ Baudrate 57600
    movne r1, #0x02                              @ Baudrate 57600 , div13
#else
    moveq r1, #0x0D                              @ Baudrate 115200
    movne r1, #0x01                              @ Baudrate 115200, div13
#endif
    str r1, [r0]

    ldr r0, =0x1e784004
    mov r1, #0x00
    str r1, [r0]

    ldr r0, =0x1e78400c
    mov r1, #0x03
    str r1, [r0]

    ldr r0, =0x1e784008
    mov r1, #0x07
    str r1, [r0]

    ldr r0, =0x1e784000
    mov r1, #0x0D                                @ '\r'
    str r1, [r0]
    mov r1, #0x0A                                @ '\n'
    str r1, [r0]
    mov r1, #0x44                                @ 'D'
    str r1, [r0]
    mov r1, #0x52                                @ 'R'
    str r1, [r0]
    mov r1, #0x41                                @ 'A'
    str r1, [r0]
    mov r1, #0x4D                                @ 'M'
    str r1, [r0]
    mov r1, #0x20                                @ ' '
    str r1, [r0]
    mov r1, #0x49                                @ 'I'
    str r1, [r0]
    mov r1, #0x6E                                @ 'n'
    str r1, [r0]
    mov r1, #0x69                                @ 'i'
    str r1, [r0]
    mov r1, #0x74                                @ 't'
    str r1, [r0]
    mov r1, #0x2D                                @ '-'
    str r1, [r0]
    mov r1, #0x44                                @ 'D'
    str r1, [r0]
    mov r1, #0x44                                @ 'D'
    str r1, [r0]
    mov r1, #0x52                                @ 'R'
    str r1, [r0]
/* Debug - UART console message */

    /* Delay about 100us */
    ldr r0, =0x1e782030                          @ Init Timer3 Control
    ldr r1, [r0]
    bic r1, r1, #0x00000F00
    str r1, [r0]

    ldr r2, =0x00000064                          @ Set Timer3 Reload = 100 us
    init_delay_timer
delay_0:
    check_delay_timer
    bne delay_0
    clear_delay_timer
    /* end delay 100us */

/******************************************************************************
 Init DRAM common registers
 ******************************************************************************/
    ldr r0, =0x1e6e0000
    ldr r1, =0xfc600309
    str r1, [r0]

    /* Reset MMC */
    ldr r1, =0x00000000
    ldr r0, =0x1e6e0004
    str r1, [r0]
    ldr r0, =0x1e6e0034
    str r1, [r0]
    ldr r0, =0x1e6e0018
    str r1, [r0]
    ldr r0, =0x1e6e0024
    str r1, [r0]
    ldr r0, =0x1e6e0064                          @ REG_MADJ, power down DLL
    str r1, [r0]

    ldr r1, =0x00034C4C                          @ REG_MADJ, reset DLL
    str r1, [r0]

    ldr r0, =0x1e6e0068                          @ REG_SADJ
    ldr r1, =0x00001800
    str r1, [r0]

    /* Delay about 10us */
    ldr r2, =0x0000000B                          @ Set Timer3 Reload = 10 us
    init_delay_timer
delay_1:
    check_delay_timer
    bne delay_1
    clear_delay_timer
    /* end delay 10us */

    ldr r0, =0x1e6e0064                          @ REG_MADJ | 0xC0000, enable DLL
    ldr r1, [r0]
    ldr r2, =0xC0000
    orr r1, r1, r2
    str r1, [r0]

    ldr r0, =0x1e6e0008
    ldr r1, =0x0090040f                          /* VGA : remember to clear MCR04[10] = 0 */
    str r1, [r0]

    ldr r0, =0x1e6e0018
    ldr r1, =0x4000A120
    str r1, [r0]

    ldr r0, =0x1e6e0018
    ldr r1, =0x00000120
    str r1, [r0]

    ldr r0, =0x1e6e0038
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e0040
    ldr r1, =0xFF444444
    str r1, [r0]

    ldr r0, =0x1e6e0044
    ldr r1, =0x22222222
    str r1, [r0]

    ldr r0, =0x1e6e0048
    ldr r1, =0x22222222
    str r1, [r0]

    ldr r0, =0x1e6e004c
    ldr r1, =0x22222222
    str r1, [r0]

    ldr r0, =0x1e6e0050
    ldr r1, =0x80000000
    str r1, [r0]

    ldr r0, =0x1e6e0050
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e0054
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e0060                          @ REG_DRV
    ldr r1, =0x000000FA                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x000000FA
#endif
    str r1, [r0]

    ldr r0, =0x1e6e0070
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e0074
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e0078
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e007c
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e0080
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e0084
    ldr r1, =0x00FFFFFF
    str r1, [r0]

    ldr r0, =0x1e6e0088                          @ REG_DQIDLY
    ldr r1, =0x00000089                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00000074
#endif
    str r1, [r0]

    ldr r0, =0x1e6e0020                          @ REG_DQSIC
    ldr r1, =0x000000E2                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x000000BA
#endif
    str r1, [r0]

    /* Delay about 10us */
    ldr r2, =0x0000000B                          @ Set Timer3 Reload = 10 us
    init_delay_timer
delay_2:
    check_delay_timer
    bne delay_2
    clear_delay_timer
    /* end delay 10us */

    /* Check DRAM Type by H/W Trapping */
    ldr r0, =0x1e6e2070
    ldr r1, [r0]
    bic r1, r1, #0xFEFFFFFF                      @ bit[24]=1 => DDR2
    mov r2, r1, lsr #24
    cmp r2, #0x01
    beq ddr2_init
    b   ddr3_init
.LTORG

/******************************************************************************
 DDR3 Init

 tRCD   = 15     ns
 tRAS   = 37.5   ns
 tRRD   = max(4 CK,10 ns)
 tRP    = 15     ns
 tRFC   = 110ns/1Gbit, 160ns/2Gbit, 300ns/4Gbit
 tRTP   = max(4 CK,7.5 ns)
 tWR    = 15     ns
 tXSNR  = max(10 CK,200 ns)
 tWTR   = max(4 CK,7.5 ns)
 tFAW   = 50     ns
 tMRD   = max(15 CK,20 ns)
 ******************************************************************************/
ddr3_init:
/* Debug - UART console message */
    ldr r0, =0x1e784000
    mov r1, #0x33                                @ '3'
    str r1, [r0]
    mov r1, #0x0D                                @ '\r'
    str r1, [r0]
    mov r1, #0x0A                                @ '\n'
    str r1, [r0]
/* Debug - UART console message */

    ldr r0, =0x1e6e0004
    ldr r1, =0x00000531                          @ Default set to 1Gbit
    str r1, [r0]

    ldr r0, =0x1e6e0010                          @ REG_AC1
    ldr r1, =0x33302825                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x22202725
#endif
    str r1, [r0]

    /* Check DRAM CL Timing by H/W Trapping */
    ldr r0, =0x1e6e2070
    ldr r1, [r0]
    bic r1, r1, #0xF9FFFFFF
    mov r2, r1, lsr #9                           @ Set CL
    ldr r1, =0x00020000
    add r2, r2, r1
    ldr r1, [r0]
    bic r1, r1, #0xFBFFFFFF
    mov r1, r1, lsr #6                           @ Set CWL
    orr r2, r2, r1
    ldr r1, =0x00300000
    add r2, r2, r1

    ldr r0, =0x1e6e0014                          @ REG_AC2
    ldr r1, =0xCC00963F                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0xAA007636
#endif
    orr r1, r1, r2
    str r1, [r0]

    ldr r0, =0x1e6e0004                          @ check 2400 mode
    ldr r2, [r0]
    mov r2, r2, lsr #10

    ldr r0, =0x1e6e006c                          @ REG_IOZ
    ldr r1, =0x00002312                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00002312
#endif
    tst r2, #0x01
    moveq r1, r1, lsr #8
    str r1, [r0]

    ldr r0, =0x1e6e0120
    mov r1, #0
    str r1, [r0]
    tst r2, #0x01                                @ check AST2300
    beq CBRDLL1_2300_Start
    ldr r0, =0x1e6e207c                          @ check AST2400 revision A0
    ldr r1, [r0]
    mov r1, r1, lsr #16
    and r1, r1, #0xFF
    cmp r1, #0x0
    beq CBRDLL1_2300_Start
    b   CBRDLL1_2400_Start
MCLK2X_Phase_CBR_Done_DDR3:
    ldr r0, =0x1e6e0018
    ldr r1, [r0]
    orr r1, r1, #0x40
    str r1, [r0]

    ldr r0, =0x1e6e0034
    ldr r1, =0x00000001
    str r1, [r0]

    ldr r0, =0x1e6e000c
    ldr r1, =0x00000040
    str r1, [r0]

    /* Delay about 400us */
    ldr r2, =0x00000190                          @ Set Timer3 Reload = 400 us
    init_delay_timer
delay3_4:
    check_delay_timer
    bne delay3_4
    clear_delay_timer
    /* end delay 400us */

    /* Check DRAM CL Timing by H/W Trapping */
    ldr r0, =0x1e6e2070
    ldr r1, [r0]
    bic r1, r1, #0xF9FFFFFF
    mov r2, r1, lsr #21                          @ Set CL
    ldr r1, =0x00000010
    add r2, r2, r1
    ldr r1, [r0]
    bic r1, r1, #0xFBFFFFFF
    mov r1, r1, lsr #7                           @ Set CWL
    orr r2, r2, r1

    ldr r0, =0x1e6e002c                          @ REG_MRS
    ldr r1, =0x04001700                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x04001500
#endif
    orr r1, r1, r2
    str r1, [r0]

    ldr r0, =0x1e6e0030                          @ REG_EMRS
    ldr r1, =0x00000000                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00000000
#endif
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set EMRS2
    ldr r1, =0x00000005
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set EMRS3
    ldr r1, =0x00000007
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set EMRS
    ldr r1, =0x00000003
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set MRS
    ldr r1, =0x00000001
    str r1, [r0]

    ldr r0, =0x1e6e002c                          @ REG_MRS
    ldr r1, =0x04001600                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x04001400
#endif
    orr r1, r1, r2
    str r1, [r0]

    ldr r0, =0x1e6e000c                          @ Refresh 8 times
    ldr r1, =0x00005C48
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set MRS
    ldr r1, =0x00000001
    str r1, [r0]

    ldr r0, =0x1e6e000c                          @ Set refresh cycle
    ldr r1, =0x00002001
    str r1, [r0]

    ldr r0, =0x1e6e0014
    ldr r1, [r0]
    bic r1, r1, #0xFFF9FFFF
    mov r2, r1, lsr #3                           @ get CL

    ldr r0, =0x1e6e0034                          @ REG_PWC
    ldr r1, =0x00000303                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00000303
#endif
    orr r1, r1, r2
    str r1, [r0]

    b   Calibration_Start
.LTORG
/******************************************************************************
 End DDR3 Init
 ******************************************************************************/

/******************************************************************************
 DDR2 Init

 tRCD   = 15    ns
 tRAS   = 45    ns
 tRRD   = 10    ns
 tRP    = 15    ns
 tRFC   = 105ns/512Mbit, 127.5ns/1Gbit, 197.5ns/2Gbit, 327.5ns/4Gbit
 tRTP   = 7.5   ns
 tWR    = 15    ns
 tXSNR  = 200   ns
 tWTR   = 7.5   ns
 tFAW   = 50    ns
 tMRD   = 4     CK
 ******************************************************************************/
ddr2_init:
/* Debug - UART console message */
    ldr r0, =0x1e784000
    mov r1, #0x32                                @ '2'
    str r1, [r0]
    mov r1, #0x0D                                @ '\r'
    str r1, [r0]
    mov r1, #0x0A                                @ '\n'
    str r1, [r0]
/* Debug - UART console message */

    ldr r0, =0x1e6e0004
    ldr r1, =0x00000521                          @ Default set to 1Gbit
    str r1, [r0]

    ldr r0, =0x1e6e0010                          @ REG_AC1
    ldr r1, =0x33302714                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x22201613
#endif
    str r1, [r0]

    /* Check DRAM CL Timing by H/W Trapping */
    ldr r0, =0x1e6e2070
    ldr r1, [r0]
    bic r1, r1, #0xF9FFFFFF
    mov r2, r1, lsr #5                           @ Set CL
    mov r1, r2, lsr #4                           @ Set CWL
    orr r2, r2, r1
    ldr r1, =0x00110000
    add r2, r2, r1

    ldr r0, =0x1e6e0014                          @ REG_AC2
    ldr r1, =0xCC00B03F                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0xAA00903B
#endif
    orr r1, r1, r2
    str r1, [r0]

    ldr r0, =0x1e6e0004                          @ check 2400 mode
    ldr r2, [r0]
    mov r2, r2, lsr #10

    ldr r0, =0x1e6e006c                          @ REG_IOZ
    ldr r1, =0x00002312                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00002312
#endif
    tst r2, #0x01
    moveq r1, r1, lsr #8
    str r1, [r0]

    ldr r0, =0x1e6e0120
    mov r1, #1
    str r1, [r0]
    tst r2, #0x01                                @ check AST2300
    beq CBRDLL1_2300_Start
    ldr r0, =0x1e6e207c                          @ check AST2400 revision A0
    ldr r1, [r0]
    mov r1, r1, lsr #16
    and r1, r1, #0xFF
    cmp r1, #0x0
    beq CBRDLL1_2300_Start
    b   CBRDLL1_2400_Start
MCLK2X_Phase_CBR_Done_DDR2:

    ldr r0, =0x1e6e0034
    ldr r1, =0x00000001
    str r1, [r0]

    ldr r0, =0x1e6e000c
    ldr r1, =0x00000000
    str r1, [r0]

    /* Delay about 400us */
    ldr r2, =0x00000190                          @ Set Timer3 Reload = 400 us
    init_delay_timer
delay2_4:
    check_delay_timer
    bne delay2_4
    clear_delay_timer
    /* end delay 400us */

    /* Check DRAM CL Timing by H/W Trapping */
    ldr r0, =0x1e6e2070
    ldr r1, [r0]
    bic r1, r1, #0xF9FFFFFF
    mov r2, r1, lsr #21                          @ Set CL
    ldr r1, =0x00000040
    orr r2, r2, r1

    ldr r0, =0x1e6e002c                          @ REG_MRS
    ldr r1, =0x00000D02                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00000B02
#endif
    orr r1, r1, r2
    str r1, [r0]

    ldr r0, =0x1e6e0030                          @ REG_EMRS
    ldr r1, =0x00000040                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00000040
#endif
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set EMRS2
    ldr r1, =0x00000005
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set EMRS3
    ldr r1, =0x00000007
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set EMRS
    ldr r1, =0x00000003
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set MRS
    ldr r1, =0x00000001
    str r1, [r0]

    ldr r0, =0x1e6e000c                          @ Refresh 8 times
    ldr r1, =0x00005C08
    str r1, [r0]

    ldr r0, =0x1e6e002c                          @ REG_MRS
    ldr r1, =0x00000C02                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00000A02
#endif
    orr r1, r1, r2
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set MRS
    ldr r1, =0x00000001
    str r1, [r0]

    ldr r0, =0x1e6e0030                          @ REG_EMRS
    ldr r1, =0x000003C0                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x000003C0
#endif
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set EMRS
    ldr r1, =0x00000003
    str r1, [r0]

    ldr r0, =0x1e6e0030                          @ REG_EMRS
    ldr r1, =0x00000040                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00000040
#endif
    str r1, [r0]

    ldr r0, =0x1e6e0028                          @ Set EMRS
    ldr r1, =0x00000003
    str r1, [r0]

    ldr r0, =0x1e6e000c                          @ Set refresh cycle
    ldr r1, =0x00002001
    str r1, [r0]

    ldr r0, =0x1e6e0014
    ldr r1, [r0]
    bic r1, r1, #0xFFF9FFFF
    mov r2, r1, lsr #3                           @ get CL

    ldr r0, =0x1e6e0034                          @ REG_PWC
    ldr r1, =0x00000503                          @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr r1, =0x00000503
#endif
    orr r1, r1, r2
    str r1, [r0]

    b   Calibration_Start
.LTORG
/******************************************************************************
 End DDR2 Init
 ******************************************************************************/
/******************************************************************************
 DDR CK duty finetune program
    SRAM buffer definition
    0x1E720204 : gdll     golden DLL1 record
    0x1E720208 : gduty    golden duty setting record
    0x1E72020C : gdutysum golden duty data record
    0x1E720210 : duty record of delay 0  invert
    0x1E720214 : duty record of delay 1  invert
    ....
    0x1E72024C : duty record of delay 15 invert
    0x1E720250 : duty record of delay 0
    0x1E720254 : duty record of delay 1
    ....
    0x1E72028C : duty record of delay 15

    Register usage
    r0 - r3 = free
    r4  = record the return pc value, do not use
    r5  = free
    r6  = free
    r7  = duty count
    r8  = gdll
    r9  = gduty
    r10 = gdutysum
 ******************************************************************************/
CBRDLL1_2400_Start:
    ldr   r0, =0x1e6e0120
    ldr   r1, [r0]
    orr   r1, r1, #0x02
    str   r1, [r0]

    ldr   r1, =0x00000000
    ldr   r0, =0x1e720204
    ldr   r2, =0x1e7202a0
init_sram_start0:
    str   r1, [r0]
    add   r0, r0, #4
    cmp   r0, r2
    blt   init_sram_start0

    ldr   r0, =0x1e6e0034
    mov   r1, #0x20
    str   r1, [r0]

    ldr   r0, =0x1e6e0060
    ldr   r1, [r0]
    mov   r2, #0x01
    orr   r1, r1, r2, lsl #13
    str   r1, [r0]

    mov   r7, #0x0                               @ init duty count
    mov   r8, #0x0                               @ init gdll
    mov   r9, #0x0                               @ init gduty
    mov   r10, #0x0                              @ init gdutysum
cbrdll1_duty_start:
    cmp   r7, #32
    bge   cbrdll1_duty_end

    ldr   r0, =0x1e6e0018
    ldr   r1, =0x00008120
    str   r1, [r0]

    ldr   r0, =0x1e6e0060
    ldr   r1, [r0]
    bic   r1, r1, #0x00001F00
    orr   r1, r1, r7, lsl #8
    mov   r2, #0x10
    eor   r1, r1, r2, lsl #8
    str   r1, [r0]

    ldr   r0, =0x1e6e0000                        @ dummy read
    ldr   r1, [r0]

    b     CBRDLL1_2300_Start
CBRDLL1_2400_Call:

    mov   r5, #0x01                              @ init dqidly count
    mov   r6, #0x00                              @ init duty sum
cbrdll1_duty_cal_start:
    cmp   r5, #0x05
    bge   cbrdll1_duty_cal_end

    ldr   r0, =0x1e6e0018
    ldr   r1, =0x00200120
    orr   r1, r1, r5, lsl #16
    str   r1, [r0]

    ldr   r0, =0x1e6e0000
    ldr   r1, [r0]

    ldr   r0, =0x1e6e0018
    ldr   r1, [r0]
    mov   r2, #0x10
    orr   r1, r1, r2, lsl #24
    str   r1, [r0]

    ldr   r0, =0x1e6e0080
    ldr   r1, =0x80000000                        @ init duty cal waiting
cbrdll1_duty_cal_wait:
    ldr   r2, [r0]
    tst   r2, r1
    beq   cbrdll1_duty_cal_wait

    ldr   r0, =0x1e6e008c
    ldr   r2, [r0]

    ldr   r0, =0x1e720210
    add   r0, r0, r7, lsl #2
    str   r2, [r0]

    ldr   r1, =0xFFFF
    and   r3, r1, r2
    cmp   r3, r1
    moveq r2, r2, lsr #16
    and   r3, r1, r2
    add   r6, r6, r3
    ldr   r1, =0xF000
    cmp   r3, r1
    blt   cbrdll1_duty_cal_end
    add   r5, r5, #0x01
    b     cbrdll1_duty_cal_start

cbrdll1_duty_cal_end:
    mov   r6, r6, lsr #2                         @ get dutysum
    cmp   r6, r10                                @ check dutysum > gdutysum
    ble   cbrdll1_duty_next
    ldr   r0, =0x1e6e0068
    ldr   r8, [r0]
    eor   r9, r7, #0x10
    mov   r10, r6

cbrdll1_duty_next:
    add   r7, r7, #0x01
    cmp   r7, #16                                @ check duty >= 15
    blt   cbrdll1_duty_start
    ldr   r0, =0xFA00                            @ check gdutysum > 0xFA00
    cmp   r10, r0
    blt   cbrdll1_duty_start

cbrdll1_duty_end:
    ldr   r0, =0x1e6e0060
    ldr   r1, [r0]
    bic   r1, r1, #0x00001F00
    orr   r1, r1, r9, lsl #8
    str   r1, [r0]

    ldr   r0, =0x1e6e0068
    bic   r8, r8, #0xFF000000
    bic   r8, r8, #0x00FF0000
    str   r8, [r0]

    ldr   r0, =0x1e720204                        @ record result
    str   r8, [r0]
    add   r0, r0, #0x04
    str   r9, [r0]
    add   r0, r0, #0x04
    str   r10, [r0]

    ldr   r0, =0x1e6e0018
    ldr   r1, =0x00008120
    str   r1, [r0]
    ldr   r0, =0x1e6e0000                        @ dummy read
    ldr   r1, [r0]
    ldr   r0, =0x1e6e0018
    ldr   r1, =0x00000120
    str   r1, [r0]

    ldr   r0, =0x1e6e0120
    ldr   r1, [r0]
    cmp   r1, #0x3
    beq   MCLK2X_Phase_CBR_Done_DDR2
    b     MCLK2X_Phase_CBR_Done_DDR3

/******************************************************************************
 MCLK2X lock to MCLK program
  r0 - r3 = free
  r5 = madjmax
  r6 = dllend
  0x1E720200 = 0x96cnt:failcnt:dllmax:dllmin
 ******************************************************************************/
CBRDLL1_2300_Start:
    ldr   r0, =0x1e6e0064
    ldr   r5, [r0]
    and   r5, r5, #0xFF                          @ init madjmax
    mov   r6, r5                                 @ init dllend

    ldr   r1, =0x000000ff
    ldr   r0, =0x1e720200
    str   r1, [r0]                               @ init dllcnt2:dllmax:dllmin

    mov   r3, #0x0                               @ init loop count
cbrdll1_scan_start:
    cmp   r3, r6
    bge   cbrdll1_scan_end

    ldr   r0, =0x1e6e0018
    ldr   r1, =0x00008120
    str   r1, [r0]

    ldr   r0, =0x1e6e0068
    mov   r1, r3
    cmp   r1, r5
    subge r1, r1, r5
    str   r1, [r0]

    ldr   r0, =0x1e6e0000                        @ dummy read
    ldr   r1, [r0]

    ldr   r0, =0x1e6e0018
    ldr   r1, =0x00000120
    str   r1, [r0]

    ldr   r0, =0x1e6e0000                        @ dummy read
    ldr   r1, [r0]
    ldr   r0, =0x1e6e0000                        @ dummy read
    ldr   r1, [r0]

    ldr   r0, =0x1e6e001c
    ldr   r1, [r0]
    mov   r1, r1, lsr #16
    and   r1, r1, #0xFF

    and   r2, r1, #0x96
    cmp   r2, #0x96
    beq   cbrdll1_scan_pass                      @ if (mclk2x_phase & 0x96) == 0x96
    ldr   r0, =0x1e720200
    ldr   r1, [r0]
    mov   r2, r1, lsr #8
    ands  r2, r2, #0xFF                          @ get dllmax
    beq   cbrdll1_scan_next                      @ if dllmax == 0
    mov   r2, r1, lsr #16
    and   r2, r2, #0xFF
    add   r2, r2, #0x01
    cmp   r2, #0x02
    movge r6, r3
    bic   r1, r1, #0x00FF0000
    orr   r1, r1, r2, lsl #16
    str   r1, [r0]
    b     cbrdll1_scan_next

cbrdll1_scan_pass:
    cmp   r3, #0x0                               @ if dll = 0
    moveq r3, #0x0F
    addeq r6, r6, #0x10
    beq   cbrdll1_scan_next
    ldr   r0, =0x1e720200
    ldr   r2, [r0]
    cmp   r1, #0x96
    bne   cbrdll1_scan_pass2
    mov   r1, r2, lsr #24
    add   r1, r1, #0x01
    bic   r2, r2, #0xFF000000
    orr   r2, r2, r1, lsl #24
    cmp   r1, #0x03                              @ check (phase == 0x96) count == 3
    bicge r2, r2, #0x0000FF00
    bicge r2, r2, #0x000000FF
    orrge r2, r2, r3, lsl #8
    orrge r2, r2, r3
    str   r2, [r0]
    bge   cbrdll1_scan_end

cbrdll1_scan_pass2:
    and   r1, r2, #0xFF                          @ if(dllmin > dll)
    cmp   r1, r3
    bicgt r2, r2, #0x000000FF
    orrgt r2, r2, r3

    mov   r1, r2, lsr #8                         @ if(dllmax < dll)
    and   r1, r1, #0xFF
    cmp   r1, r3
    biclt r2, r2, #0x0000FF00
    orrlt r2, r2, r3, lsl #8

    bic   r2, r2, #0x00FF0000
    str   r2, [r0]

cbrdll1_scan_next:
    add   r3, r3, #0x01
    b     cbrdll1_scan_start

cbrdll1_scan_end:
    ldr   r0, =0x1e720200
    ldr   r1, [r0]
    mov   r2, r1, lsr #8                         @ get dllmax
    ands  r2, r2, #0xFF
    bne   cbrdll1_scan_done                      @ if(dllmax != 0)
    ldr   r0, =0x1e6e0064
    ldr   r3, [r0]
    bic   r1, r3, #0x000C0000
    str   r1, [r0]
    add   r0, r0, #0x04
    mov   r1, #0x0
    str   r1, [r0]

    /* Delay about 10us */
    ldr r2, =0x0000000A                          @ Set Timer3 Reload = 10 us
    init_delay_timer
delay0_1:
    check_delay_timer
    bne delay0_1
    clear_delay_timer
    /* end delay 10us */

    ldr   r0, =0x1e6e0064
    str   r3, [r0]

    /* Delay about 10us */
    ldr r2, =0x0000000A                          @ Set Timer3 Reload = 10 us
    init_delay_timer
delay0_2:
    check_delay_timer
    bne delay0_2
    clear_delay_timer
    /* end delay 10us */

    b     CBRDLL1_2300_Start

cbrdll1_scan_done:
    and   r1, r1, #0xFF
    add   r1, r1, r2
    mov   r6, r1, lsr #1                         @ dll1.0 = (dllmin + dllmax) >> 1
    cmp   r6, r5
    subge r6, r6, r5
    add   r3, r6, r5, lsr #2                     @ dll1.1 = dll1.0 + (MADJ >> 2)

    ldr   r0, =0x1e6e0004
    ldr   r1, [r0]
    mov   r1, r1, lsr #10
    tst   r1, #0x1
    bne   cbrdll1_scan_set_2400
    cmp   r3, r5
    subge r3, r3, r5
    mov   r2, #0x0
    tst   r3, #0x08
    beq   cbrdll1_scan_set_2300_2                @ if !(dll & 8)
cbrdll1_scan_set_2300_1:                         @ if  (dll & 8)
    mov   r1, #0x0
    tst   r3, #0x08
    addeq r1, r1, #0x01
    cmp   r2, #0x05
    addge r1, r1, #0x01
    cmp   r1, #0x02
    beq   cbrdll1_scan_set
    add   r2, r2, #0x01
    add   r3, r3, #0x01
    cmp   r3, r5
    subge r3, r3, r5
    b     cbrdll1_scan_set_2300_1

cbrdll1_scan_set_2300_2:
    and   r1, r3, #0x07
    cmp   r1, #0x07
    beq   cbrdll1_scan_set
    cmp   r2, #0x05
    bge   cbrdll1_scan_set
    add   r2, r2, #0x01
    add   r3, r3, #0x01
    cmp   r3, r5
    subge r3, r3, r5
    b     cbrdll1_scan_set_2300_2

cbrdll1_scan_set_2400:
    add   r3, r3, #0x05                          @ dll1.1 = dll1.0 + (MADJ >> 2) + 5
    cmp   r3, r5
    subge r3, r3, r5

cbrdll1_scan_set:
    orr   r1, r6, r3, lsl #8
    ldr   r0, =0x1e6e0068
    str   r1, [r0]

    ldr   r0, =0x1e6e0120
    ldr   r1, [r0]
    cmp   r1, #0x0
    beq   MCLK2X_Phase_CBR_Done_DDR3
    cmp   r1, #0x1
    beq   MCLK2X_Phase_CBR_Done_DDR2
    b     CBRDLL1_2400_Call

.LTORG

/******************************************************************************
 Calibration Code Start
    SRAM buffer definition
    0x1E720000 : Pass 1, DLLI MIN value range
    0x1E720008 : DQS0 DLL valid range, 2nd time CBR
    0x1E72000C : DQS1 DLL valid range, 2nd time CBR
    0x1E720010 : DQ0  DLL valid range, Pass 1
    0x1E720014 : DQ1  DLL valid range, Pass 1
    ....
    0x1E720048 : DQ14 DLL valid range, Pass 1
    0x1E72004C : DQ15 DLL valid range, Pass 1
    0x1E720090 : DLL1 SAdj record
    0x1E720094 : DQL  Pass1 finetune result
    0x1E720098 : DQH  Pass1 finetune result
    0x1E72009C : DRAM initial time, (us)
    0x1E7200A0 : CBR3 retry counter
    0x1E7200A4 : DRAM initial time, (us)
    0x1E7200A8 : Released date
    0x1E7200AC : Released SDK version
    0x1E7200B0 : DQS input mask window for MCR18[4] = 0
    0x1E7200B4 : DQS input mask window for MCR18[4] = 1
    0x1E720100 : DQIDLY=00, DLL valid range
    0x1E720104 : DQIDLY=01, DLL valid range
    ....
    0x1E720178 : DQIDLY=30, DLL valid range
    0x1E72017C : DQIDLY=31, DLL valid range
    0x1E720180 : DQSI-MCLK2X P-phase pass record DLL2= 0-31
    0x1E720184 : DQSI-MCLK2X P-phase pass record DLL2=32-63
    0x1E720188 : DQSI-MCLK2X N-phase pass record DLL2= 0-31
    0x1E72018C : DQSI-MCLK2X N-phase pass record DLL2=32-63
 ******************************************************************************/
Calibration_Start_pre:                           @ Toggle DQSI mask delay
    ldr r0, =0x1e6e0018
    ldr r1, [r0]
    eor r1, r1, #0x10
    str r1, [r0]

Calibration_Start:
/* Init SRAM buffer */
    ldr r1, =0x000000ff
    ldr r0, =0x1e720000
    ldr r2, =0x1e720100
init_sram_start:
    str r1, [r0]
    add r0, r0, #4
    cmp r0, r2
    blt init_sram_start

    ldr r1, =0x00ff00ff
    ldr r0, =0x1e720100
    ldr r2, =0x1e720180
init_sram_start2:
    str r1, [r0]
    add r0, r0, #4
    cmp r0, r2
    blt init_sram_start2

    ldr r1, =0x00000000
    ldr r0, =0x1e720180
    ldr r2, =0x1e720200
init_sram_start3:
    str r1, [r0]
    add r0, r0, #4
    cmp r0, r2
    blt init_sram_start3

    ldr r0, =0x1e6e0068                          @ save the DLL1 SAdj initial value
    ldr r1, [r0]
    ldr r0, =0x1e720090
    str r1, [r0]

/* Start
  r0 = free
  r1 = free
  r2 = free
  r3 = free
  r4 = record the return pc value, do not use
  r5 = pattern table index
  r6 = pass count
  r7 = dram DLL2 parameter index (0x1e6e0068), max is 0x4C
*/
/******************************************************************************
 Fine DQI delay and DQSI-MCLK phase
  r8  = DQIDLY count
  r9  = DQSI-MCLK2X phase count
  r10 = pattern fail retry counter, initialize to 2 (fail 2 times)
  r11 = passcnt accumulator for each DQIDLY
 *****************************************************************************/
CBR0_START:
/* Debug - UART console message */
    ldr r0, =0x1e784000
    mov r1, #0x43                                @ 'C'
    str r1, [r0]
    mov r1, #0x42                                @ 'B'
    str r1, [r0]
    mov r1, #0x52                                @ 'R'
    str r1, [r0]
    mov r1, #0x30                                @ '0'
    str r1, [r0]
    mov r1, #0x2D                                @ '-'
    str r1, [r0]
/* Debug - UART console message */

    ldr   r0, =0x1e6e0018
    ldr   r1, [r0]
    bic   r1, r1, #0xFF000000
    bic   r1, r1, #0x00FF0000
    str   r1, [r0]

    ldr   r0, =0x1e6e0074                        @ set the testing DRAM size = 1KB
    ldr   r1, =0x000003FF
    str   r1, [r0]

    mov   r8, #0x00                              @ init DQIDLY
    mov   r9, #0x00                              @ init DQSI-MCLK2X phase
    mov   r11, #0x01                             @ init passcnt accumulator

cbr0_next_dqidly:
    cmp   r9, #0x00
    bne   cbr0_next_dqsiphase
    cmp   r11, #0x00
    addeq r8, r8, #0x01                          @ jump 1 stage if no pass at previous stage
    mov   r11, #0x00
    add   r8, r8, #0x01
    cmp   r8, #0x1F                              @ max DQIDLY = 31
    bgt   CBR0_END

/* Debug - UART console message */
    ldr   r0, =0x1e784000
    and   r1, r8, #0x07
    add   r1, r1, #0x30                          @ '0-7'
    str   r1, [r0]
/* Debug - UART console message */

    ldr   r0, =0x1e6e0018
    ldr   r1, [r0]
    bic   r1, r1, #0x00FF0000
    orr   r1, r1, r8, lsl #16
    str   r1, [r0]
    mov   r9, #0x01                              @ '1':p_phase, '0':n_phase

    /* Delay about 3us */                        @ wait DQIDLY load
    ldr r2, =0x00000003                          @ Set Timer4 Reload = 3 us
    init_delay_timer
delay_4:
    check_delay_timer
    bne delay_4
    clear_delay_timer
    /* end delay 3us */

    b     cbr0_dll2_scan_start

cbr0_next_dqsiphase:
    ldr   r0, =0x1e6e0018
    ldr   r1, [r0]
    orr   r1, r1, r9, lsl #23                    @ set DQSI-MCLK2X phase
    str   r1, [r0]
    mov   r9, #0x00

cbr0_dll2_scan_start:
    mov   r6, #0x00                              @ init pass count
    mov   r7, #0x00                              @ init DLL2 parameter index

/****************************
 DLL2 delay margin test loop
 ***************************/
cbr0_next_dll2_parameter:
    ldr   r0, =0x1e6e0068                        @ load DLL2 parameter
    ldr   r1, [r0]
    bic   r1, r1, #0x00FF0000
    bic   r1, r1, #0xFF000000
    orr   r1, r1, r7, lsl #16
    str   r1, [r0]
    ldr   r2, =0x40404040                        @ DLL2 max is 0x40404040
    cmp   r7, r2
    bge   cbr0_next_dqidly
    ldr   r2, =0x01010101
    add   r7, r7, r2

/* CBRScan3() start */
    adrl  r5, PATTERN_TABLE                      @ init pattern table index
/****************************
 Test pattern iteration loop
 ***************************/
cbr0_next_test_pattern:
    mov   r10, #2                                @ set the retry loop = 2 of each pattern
    ldr   r1, [r5]                               @ load test pattern
    ldr   r0, =0x1e6e007c
    str   r1, [r0]
    cmp   r1, #0x00                              @ the last data in pattern is 0x00
    bne   cbr0_test_burst

    and   r3, r7, #0xFF
    sub   r3, r3, #0x01                          @ we add 1 after loop check so we need to decrease 1
    cmp   r3, #0x00
    beq   cbr0_next_dqidly                       @ pass at dlli = 0, invalid
    add   r6, r6, #0x01                          @ increment pass count
    add   r11, r11, #0x01                        @ increment pass count

    ldr   r0, =0x1e720180                        @ record DLL2 pass window
    cmp   r9, #0x00                              @ DQSI-MCLK2X phase check
    addeq r0, r0, #0x08
    cmp   r3, #32
    addge r0, r0, #0x4
    and   r1, r3, #0x1F
    mov   r2, #0x1
    mov   r2, r2, lsl r1
    ldr   r1, [r0]
    orr   r1, r1, r2
    str   r1, [r0]

    ldr   r0, =0x1e720100                        @ record DLL2 min:max value for each DQIDLY
    add   r0, r0, r8, lsl #2
    cmp   r9, #0x00                              @ DQSI-MCLK2X phase check
    beq   cbr0_test_pass_dqsin
    record_dll2_pass_range
    b     cbr0_next_dll2_parameter

cbr0_test_pass_dqsin:
    record_dll2_pass_range_h
    b     cbr0_next_dll2_parameter

cbr0_test_pattern_fail:
    cmp   r6, #5                                 @ passcnt >= 5
    bge   cbr0_next_dqidly
    ldr   r0, =0x1e720100                        @ reset DLL2 min:max value
    add   r0, r0, r8, lsl #2
    ldr   r1, [r0]
    ldr   r2, =0xFFFF0000
    ldr   r3, =0x000000FF
    cmp   r9, #0x00
    moveq r2, r2, lsr #16
    moveq r3, r3, lsl #16
    and   r1, r1, r2
    orr   r1, r1, r3
    str   r1, [r0]
    b     cbr0_next_dll2_parameter               @ CBRScan3() end and test result fail, go to next step

/****************************
 Test fail retry loop
 ***************************/
cbr0_pattern_fail_retry:

/* CBRTest3() start */
cbr0_test_burst:
    ldr   r0, =0x1e6e0070
    ldr   r1, =0x00000000
    str   r1, [r0]
    ldr   r1, =0x000000C1
    str   r1, [r0]
    ldr   r3, =0x3000
cbr0_wait_engine_idle_0:
    ldr   r2, [r0]
    tst   r2, r3                                 @ D[12] = idle bit
    beq   cbr0_wait_engine_idle_0

    ldr   r2, [r0]                               @ read fail bit status
    mov   r1, #0x0
    str   r1, [r0]
    mov   r2, r2, lsr #13                        @ D[13] = fail bit
    cmp   r2, #0x00
    bne   cbr0_test_fail

cbr0_test_single:
    ldr   r0, =0x1e6e0070
    ldr   r1, =0x00000000
    str   r1, [r0]
    ldr   r1, =0x00000085
    str   r1, [r0]
    ldr   r3, =0x3000
cbr0_wait_engine_idle_1:
    ldr   r2, [r0]
    tst   r2, r3                                 @ D[12] = idle bit
    beq   cbr0_wait_engine_idle_1

    ldr   r2, [r0]                               @ read fail bit status
    mov   r1, #0x0
    str   r1, [r0]
    mov   r2, r2, lsr #13                        @ D[13] = fail bit
    cmp   r2, #0x00
    beq   cbr0_test_pass

/* CBRTest3() end */

cbr0_test_fail:
    subs  r10, r10, #1
    bne   cbr0_pattern_fail_retry
    b     cbr0_test_pattern_fail                 @ CBRScan3() return(0)

cbr0_test_pass:
    add   r5, r5, #0x04                          @ increase the test pattern index
    b     cbr0_next_test_pattern

CBR0_END:
    mov   r5, #0x0                               @ init DQIDLY search count
    mov   r6, #0x0                               @ init max_margin:g_margin
    mov   r8, #0x0                               @ init g_side
    mov   r7, #0x0                               @ init maximum margin DQIDLY,DQSI-MCLK2X phase
cbr0_search_dll_margin_s:
    ldr   r0, =0x1e720100
    add   r0, r0, r5, lsl #2
    ldr   r1, [r0]
    and   r2, r1, #0xFF                          @ get dllmin_p
    mov   r1, r1, lsr #8
    and   r3, r1, #0xFF                          @ get dllmax_p
    subs  r2, r3, r2                             @ get margin-P
    movmi r2, #0x0
    mov   r1, r1, lsr #8
    and   r3, r1, #0xFF                          @ get dllmin_n
    mov   r1, r1, lsr #8
    and   r1, r1, #0xFF                          @ get dllmax_n
    subs  r3, r1, r3                             @ get margin-N
    movmi r3, #0x0
    add   r1, r2, r3
    cmp   r1, #0x0
    beq   cbr0_search_dll_margin_e               @ if margin-P = 0 && margin-N = 0

    ldr   r9, [r0]
    ldr   r0, =0x1e720180
    cmp   r2, r3
    orrlt r5, r5, #0x80                          @ margin-N > margin-P
    addlt r0, r0, #0x08
    movlt r9, r9, lsr #16
    movge r3, r2                                 @ max(margin-P/N)
    add   r2, r3, #0x2                           @ define +/- 2 steps of variation
    mov   r1, r6, lsr #16
    cmp   r2, r1
    blt   cbr0_search_dll_margin_e               @ if max(margin-P/N) + 2 < max_margin

    and   r1, r9, #0xFF                          @ r1 = dlli counter
    cmp   r1, #32
    ldrge r2, [r0, #0x4]                         @ load pass window
    ldrlt r2, [r0]
    and   r1, r1, #0x1F
    mov   r10, #0x1                              @ init test bit mask
    mov   r10, r10, lsl r1
    and   r1, r9, #0xFF
cbr0_search_dllmin_margin_s:
    tst   r2, r10
    beq   cbr0_search_dllmin_margin_e
    mov   r10, r10, lsr #1
    cmp   r1, #32
    ldreq r2, [r0]
    ldreq r10, =0x80000000
    subs  r1, r1, #0x1
    bne   cbr0_search_dllmin_margin_s

cbr0_search_dllmin_margin_e:
    and   r2, r9, #0xFF
    sub   r11, r2, r1                            @ get dllmin side margin

    mov   r9, r9, lsr #8
    and   r1, r9, #0xFF                          @ r1 = dlli counter
    cmp   r1, #32
    ldrge r2, [r0, #0x4]                         @ load pass window
    ldrlt r2, [r0]
    and   r1, r1, #0x1F
    mov   r10, #0x1                              @ init test bit mask
    mov   r10, r10, lsl r1
    and   r1, r9, #0xFF
cbr0_search_dllmax_margin_s:
    tst   r2, r10
    beq   cbr0_search_dllmax_margin_e
    mov   r10, r10, lsl #1
    cmp   r1, #31
    ldreq r2, [r0, #0x4]
    ldreq r10, =0x00000001
    add   r1, r1, #0x1
    cmp   r1, #64
    bne   cbr0_search_dllmax_margin_s

cbr0_search_dllmax_margin_e:
    and   r2, r9, #0xFF
    sub   r1, r1, r2                             @ get dllmax side margin
    cmp   r1, r11
    movlt r11, r1                                @ get side_margin

cbr0_check_dll_margin:                           @ if max(margin-P/N) > g_margin && side_margin >= g_side && dqidly <= 20
    cmp   r5, #20
    bgt   cbr0_check_dll_margin2
    and   r1, r6, #0xFF
    cmp   r3, r1
    ble   cbr0_check_dll_margin3
    cmp   r11, r8
    bge   cbr0_set_dll_margin

cbr0_check_dll_margin2:                          @ if max(margin-P/N) > g_margin+1 && side_margin >= g_side)
    and   r1, r6, #0xFF
    add   r2, r1, #0x1
    cmp   r3, r2
    ble   cbr0_check_dll_margin3
    cmp   r11, r8
    bge   cbr0_set_dll_margin

cbr0_check_dll_margin3:                          @ if side_margin > g_side && g_side < 8
    cmp   r8, #8
    bge   cbr0_search_dll_margin_e
    cmp   r11, r8
    ble   cbr0_search_dll_margin_e

cbr0_set_dll_margin:
    mov   r1, r6, lsr #16
    cmp   r3, r1
    bicgt r6, r6, #0x00FF0000
    orrgt r6, r6, r3, lsl #16
    bic   r6, r6, #0x000000FF
    orr   r6, r6, r3
    mov   r7, r5
    mov   r8, r11

cbr0_search_dll_margin_e:
    and   r5, r5, #0x7F
    add   r5, r5, #0x01
    cmp   r5, #0x20                              @ last DQIDLY
    blt   cbr0_search_dll_margin_s

    ldr   r0, =0x1e6e0018
    ldr   r1, [r0]
    bic   r1, r1, #0x00FF0000
    orr   r1, r1, r7, lsl #16
    str   r1, [r0]

    ldr   r0, =0x1e6e0068
    ldr   r1, [r0]
    bic   r1, r1, #0x00FF0000
    bic   r1, r1, #0xFF000000
    str   r1, [r0]

    /* Delay about 5us */
    ldr r2, =0x00000005                          @ Set Timer5 Reload = 5 us
    init_delay_timer
delay_5:
    check_delay_timer
    bne delay_5
    clear_delay_timer
    /* end delay 5us */

    ldr r0, =0x1e6e000c                          @ Set refresh cycle
    ldr r1, =0x00005C01
    str r1, [r0]

/******************************************************************************
 Fine tune per bit DQ input delay -- Pass 1, left edge align
  r8  = free
  r9  = DQ fail bit accumulator
  r10 = pattern fail counter, initialize to 5 (fail 5 times)
  r11 = free
 *****************************************************************************/
CBR1_START:
/* Debug - UART console message */
    ldr r0, =0x1e784000
    mov r1, #0x0D                                @ '\r'
    str r1, [r0]
    mov r1, #0x0A                                @ '\n'
    str r1, [r0]
    mov r1, #0x43                                @ 'C'
    str r1, [r0]
    mov r1, #0x42                                @ 'B'
    str r1, [r0]
    mov r1, #0x52                                @ 'R'
    str r1, [r0]
    mov r1, #0x31                                @ '1'
    str r1, [r0]
/* Debug - UART console message */

    mov   r6, #0x00                              @ init pass count
    mov   r7, #0x00                              @ init DLL2 parameter index

/****************************
 DLL2 delay margin test loop
 ***************************/
cbr1_next_dll2_parameter:
    ldr   r0, =0x1e6e0068                        @ load DLL2 parameter
    ldr   r1, [r0]
    bic   r1, r1, #0x00FF0000
    bic   r1, r1, #0xFF000000
    orr   r1, r1, r7, lsl #16
    str   r1, [r0]
    ldr   r2, =0x40404040                        @ parameter's max is to 0x40404040
    cmp   r7, r2
    bge   CBR1_END
    ldr   r2, =0x01010101
    add   r7, r7, r2

    ldr   r0, =0x1e6e0074                        @ set the testing DRAM size = 4KB
    ldr   r1, =0x00000FFF
    str   r1, [r0]

/* CBRScan2() start */
    ldr   r9, =0xFFFF                            @ init test status
    adrl  r5, PATTERN_TABLE                      @ init pattern table index
/****************************
 Test pattern iteration loop
 ***************************/
cbr1_next_test_pattern:
    mov   r10, #5                                @ set the retry loop of each pattern
    ldr   r1, [r5]                               @ load test pattern
    ldr   r0, =0x1e6e007c
    str   r1, [r0]
    cmp   r1, #0x00                              @ the last data in pattern is 0x00
    bne   cbr1_test_single

cbr1_test_pattern_end:
    cmp   r9, #0x00
    bne   cbr1_test_pass_dqi
    cmp   r6, #10
    bge   CBR1_END
    b     cbr1_next_dll2_parameter               @ CBRScan2() end and test result fail, go to next step

cbr1_test_pass_dqi:
    and   r3, r7, #0xFF
    sub   r3, r3, #0x01                          @ we add 1 after loop check so we need to decrease 1
    add   r6, r6, #0x01                          @ increment pass count
    ldr   r0, =0x1e720010
    mov   r8, #0x01
cbr1_test_pass_dqi_loop_s:
    tst   r9, r8
    beq   cbr1_test_pass_dqi_loop_e
    record_dll2_pass_range

cbr1_test_pass_dqi_loop_e:
    add   r0, r0, #0x04
    mov   r8, r8, lsl #1
    ldr   r1, =0xFFFF
    tst   r8, r1
    bne   cbr1_test_pass_dqi_loop_s
    b     cbr1_next_dll2_parameter

/****************************
 Test fail retry loop
 ***************************/
cbr1_pattern_fail_retry:

/* CBRTest2() start */
cbr1_test_single:
    ldr   r0, =0x1e6e0070
    ldr   r1, =0x00000000
    str   r1, [r0]
    ldr   r1, =0x00000005
    str   r1, [r0]
    ldr   r3, =0x1000
    ldr   r1, =0x1000
cbr1_wait_engine_idle_0:
    subs  r1, r1, #1
    beq   cbr1_test_single_end
    ldr   r2, [r0]
    tst   r2, r3                                 @ D[12] = idle bit
    beq   cbr1_wait_engine_idle_0

cbr1_test_single_end:
    ldr   r0, =0x1e6e0078                        @ read fail bit status
    ldr   r11, [r0]
    orr   r11, r11, r11, lsr #16
    bic   r11, r11, #0xFF000000
    bic   r11, r11, #0x00FF0000

    ldr   r1, =0xFFFF
    cmp   r11, r1
    beq   cbr1_test_fail

cbr1_test_burst:
    ldr   r0, =0x1e6e0070
    ldr   r2, =0x00000000
    str   r2, [r0]
    ldr   r2, =0x00000041
    str   r2, [r0]
    ldr   r3, =0x1000
    ldr   r1, =0x1000
cbr1_wait_engine_idle_1:
    subs  r1, r1, #1
    beq   cbr1_test_burst_end
    ldr   r2, [r0]
    tst   r2, r3                                 @ D[12] = idle bit
    beq   cbr1_wait_engine_idle_1

cbr1_test_burst_end:
    ldr   r0, =0x1e6e0078                        @ read fail bit status
    ldr   r2, [r0]
    orr   r2, r2, r2, lsr #16
    bic   r2, r2, #0xFF000000
    bic   r2, r2, #0x00FF0000
    orr   r11, r11, r2

    ldr   r2, =0xFFFF
    cmp   r11, r2
    bne   cbr1_test_pass
/* CBRTest2() end */

cbr1_test_fail:
    subs  r10, r10, #1
    bne   cbr1_pattern_fail_retry
    mov   r9, #0x00
    b     cbr1_test_pattern_end                  @ CBRScan2() return(0)

cbr1_test_pass:
    ldr   r1, =0xFFFF                            @ record the pass bit
    eor   r11, r11, r1
    and   r9, r9, r11                            @ DQ pass bit
    cmp   r9, #0x00
    beq   cbr1_test_pattern_end                  @ CBRScan2() return(0)

    add   r5, r5, #0x04                          @ increase the test pattern index
    b     cbr1_next_test_pattern

CBR1_END:
    mov   r5, #0x0                               @ init DQ DLL_min sum
    mov   r6, #0x0                               @ init DQ DLL_min valid count
    ldr   r0, =0x1e72000c
    ldr   r3, =0x1e720050
cbr1_search_dllmin_s:
    add   r0, r0, #0x04
    cmp   r0, r3
    beq   cbr1_search_dllmin_e
    ldr   r1, [r0]
    mov   r2, r1, lsr #8
    and   r2, r2, #0xFF                          @ get dllmax
    and   r1, r1, #0xFF                          @ get dllmin
    subs  r2, r2, r1                             @ dllmax - dllmin
    bmi   cbr1_search_dllmin_s                   @ no valid margin found, bypass fine tune
    cmp   r2, #10                                @ (dllmax - dllmin) < 10
    blt   cbr1_search_dllmin_s                   @ no enough margin found, bypass fine tune
    add   r5, r5, r1
    add   r6, r6, #1
    b     cbr1_search_dllmin_s

cbr1_search_dllmin_e:
    cmp   r6, #16
    bne   Calibration_Start_pre                  @ not all bits valid, retry again

    mov   r5, r5, lsr #4
    ldr   r0, =0x1e720000
    str   r5, [r0]

    mov   r6, #0x00                              @ init DQL CBR value
    ldr   r0, =0x1e720030
    ldr   r7, =0x1e72000c
cbr1_set_result_dql:
    sub   r0, r0, #4
    cmp   r0, r7
    beq   cbr1_set_result_next
    mov   r6, r6, lsl #3
    ldr   r1, [r0]
    mov   r2, r1, lsr #8
    and   r2, r2, #0xFF                          @ get dllmax
    and   r1, r1, #0xFF                          @ get dllmin
    mov   r3, r1                                 @ dll = dllmin
    cmp   r5, r3
    blt   cbr1_set_result_dql_neg
    sub   r1, r5, r3
    mov   r2, #19
    mul   r1, r2, r1
    mov   r1, r1, lsr #5                         @ dqi_tune = ((gold_dll - dll) * 19) >> 5
    cmp   r1, #2                                 @ dqi_tune max = 2
    movgt r1, #2
    orr   r6, r6, r1
    b     cbr1_set_result_dql

cbr1_set_result_dql_neg:
    sub   r1, r3, r5
    mov   r2, #19
    mul   r1, r2, r1
    mov   r1, r1, lsr #5                         @ dqi_tune = ((gold_dll - dll) * 19) >> 5
    cmp   r1, #2                                 @ dqi_tune max = -2
    movgt r1, #2
    mov   r2, #8
    sub   r1, r2, r1
    and   r1, r1, #7
    orr   r6, r6, r1
    b     cbr1_set_result_dql

cbr1_set_result_next:
    ldr   r0, =0x1e6e0080                        @ save DQL fine tune result
    str   r6, [r0]
    ldr   r0, =0x1e720094
    str   r6, [r0]

    mov   r6, #0x00                              @ init DQH CBR value
    ldr   r0, =0x1e720050
    ldr   r7, =0x1e72002c
cbr1_set_result_dqh:
    sub   r0, r0, #4
    cmp   r0, r7
    beq   cbr1_set_result_end
    mov   r6, r6, lsl #3
    ldr   r1, [r0]
    mov   r2, r1, lsr #8
    and   r2, r2, #0xFF                          @ get dllmax
    and   r1, r1, #0xFF                          @ get dllmin
    mov   r3, r1                                 @ dll = dllmin
    cmp   r5, r3
    blt   cbr1_set_result_dqh_neg
    sub   r1, r5, r3
    mov   r2, #19
    mul   r1, r2, r1
    mov   r1, r1, lsr #5                         @ dqi_tune = ((gold_dll - dll) * 19) >> 5
    cmp   r1, #3                                 @ dqi_tune max = 2
    movgt r1, #3
    subs  r1, r1, #1
    movmi r1, #7
    orr   r6, r6, r1
    b     cbr1_set_result_dqh

cbr1_set_result_dqh_neg:
    sub   r1, r3, r5
    mov   r2, #19
    mul   r1, r2, r1
    mov   r1, r1, lsr #5                         @ dqi_tune = ((gold_dll - dll) * 19) >> 5
    add   r1, r1, #1
    cmp   r1, #2                                 @ dqi_tune max = -2
    movgt r1, #2
    mov   r2, #8
    sub   r1, r2, r1
    and   r1, r1, #7
    orr   r6, r6, r1
    b     cbr1_set_result_dqh

cbr1_set_result_end:
    ldr   r0, =0x1e6e0084                        @ save DQH fine tune result
    str   r6, [r0]
    ldr   r0, =0x1e720098
    str   r6, [r0]

/******************************************************************************
 Search the DLL2 detail margin
 *****************************************************************************/
    ldr   r0, =0x1e7200a0
    mov   r1, #0
    str   r1, [r0]

CBR3_START:
/* Debug - UART console message */
    ldr r0, =0x1e784000
    mov r1, #0x33                                @ '3'
    str r1, [r0]
/* Debug - UART console message */

    mov   r6, #0x00                              @ init pass count
    mov   r7, #0x00                              @ init DLL2 parameter index
    ldr   r1, =0x000000ff
    ldr   r0, =0x1e720008                        @ init DQL dllmax,dllmin
    str   r1, [r0]
    ldr   r0, =0x1e72000c                        @ init DQH dllmax,dllmin
    str   r1, [r0]

    ldr   r0, =0x1e7200a0                        @ CBR3 iteration counter
    ldr   r1, [r0]
    add   r1, r1, #1
    str   r1, [r0]

/****************************
 DLL2 delay margin test loop
 ***************************/
cbr3_next_dll2_parameter:
    ldr   r0, =0x1e6e0068                        @ load DLL2 parameter
    ldr   r1, [r0]
    bic   r1, r1, #0x00FF0000
    bic   r1, r1, #0xFF000000
    orr   r1, r1, r7, lsl #16
    str   r1, [r0]
    ldr   r2, =0x40404040                        @ parameter's max is to 0x40404040
    cmp   r7, r2
    bge   CBR3_END
    ldr   r2, =0x01010101
    add   r7, r7, r2

    ldr   r0, =0x1e6e0074                        @ set the testing DRAM size = 64KB
    ldr   r1, =0x0000FFFF
    str   r1, [r0]

/* CBRScan() start */
    mov   r9, #0x03                              @ init test status
    adrl  r5, PATTERN_TABLE                      @ init pattern table index
/****************************
 Test pattern iteration loop
 ***************************/
cbr3_next_test_pattern:
    mov   r10, #5                                @ set the retry loop of each pattern
    ldr   r1, [r5]                               @ load test pattern
    ldr   r0, =0x1e6e007c
    str   r1, [r0]
    cmp   r1, #0x00                              @ the last data in pattern is 0x00
    bne   cbr3_test_single

cbr3_test_pattern_end:
    cmp   r9, #0x00
    bne   cbr3_test_pass_dql
    cmp   r6, #10
    bge   CBR3_END
    b     cbr3_next_dll2_parameter               @ CBRScan() end and test result fail, go to next step

cbr3_test_pass_dql:
    and   r3, r7, #0xFF
    sub   r3, r3, #0x01                          @ we add one after loop check so we need to decrease 1
    add   r6, r6, #0x01                          @ increment pass count
    tst   r9, #0x01
    beq   cbr3_test_pass_dqh

    ldr   r0, =0x1E720008
    record_dll2_pass_range

cbr3_test_pass_dqh:
    tst   r9, #0x02
    beq   cbr3_next_dll2_parameter
    ldr   r0, =0x1E72000c
    record_dll2_pass_range
    b     cbr3_next_dll2_parameter

/****************************
 Test fail retry loop
 ***************************/
cbr3_pattern_fail_retry:

/* CBRTest() start */
cbr3_test_single:
    ldr   r0, =0x1e6e0070
    ldr   r1, =0x00000000
    str   r1, [r0]
    ldr   r1, =0x00000005
    str   r1, [r0]
    ldr   r3, =0x1000
    ldr   r8, =0x10000
cbr3_wait_engine_idle_0:
    subs  r8, r8, #1
    beq   cbr3_test_single_end
    ldr   r2, [r0]
    tst   r2, r3                                 @ D[12] = idle bit
    beq   cbr3_wait_engine_idle_0

cbr3_test_single_end:
    ldr   r0, =0x1e6e0078                        @ read fail bit status
    ldr   r11, [r0]
    orr   r11, r11, r11, lsr #16
    bic   r11, r11, #0xFF000000
    bic   r11, r11, #0x00FF0000

    ldr   r1, =0xFF
    tst   r11, r1
    beq   cbr3_test_burst
    tst   r11, r1, lsl #8
    bne   cbr3_test_fail

cbr3_test_burst:
    mov   r1, #0x00                              @ initialize loop index, r1 is loop's index
cbr3_test_burst_loop:
    ldr   r0, =0x1e6e0070
    ldr   r2, =0x00000000
    str   r2, [r0]
    mov   r2, r1, lsl #3
    orr   r2, r2, #0x41                          @ test command = 0x41 | (datagen << 3)
    str   r2, [r0]
    ldr   r3, =0x1000
    ldr   r8, =0x10000
cbr3_wait_engine_idle_1:
    subs  r8, r8, #1
    beq   cbr3_test_burst_end
    ldr   r2, [r0]
    tst   r2, r3                                 @ D[12] = idle bit
    beq   cbr3_wait_engine_idle_1

cbr3_test_burst_end:
    ldr   r0, =0x1e6e0078                        @ read fail bit status
    ldr   r2, [r0]
    orr   r2, r2, r2, lsr #16
    bic   r2, r2, #0xFF000000
    bic   r2, r2, #0x00FF0000
    orr   r11, r11, r2

    ldr   r2, =0xFF
    tst   r11, r2
    beq   cbr3_next_test_burst_mode
    tst   r11, r2, lsl #8
    beq   cbr3_next_test_burst_mode
/* CBRTest() end */

cbr3_test_fail:
    subs  r10, r10, #1
    bne   cbr3_pattern_fail_retry
    mov   r9, #0x00
    b     cbr3_test_pattern_end                  @ CBRScan() return(0)

cbr3_next_test_burst_mode:
    add   r1, r1, #1                             @ increase the test mode index
    cmp   r1, #0x08                              @ there are 8 modes
    bne   cbr3_test_burst_loop

    ldr   r1, =0xFF                              @ record the pass byte
    tst   r11, r1
    andne r9, r9, #0x02                          @ DQL fail
    tst   r11, r1, lsl #8
    andne r9, r9, #0x01                          @ DQH fail
    cmp   r9, #0x00
    beq   cbr3_test_pattern_end                  @ CBRScan() return(0)

    add   r5, r5, #0x04                          @ increase the test pattern index
    b     cbr3_next_test_pattern

CBR3_END:
    ldr   r0, =0x1e72000c                        @ check DQH margin
    ldr   r1, [r0]
    mov   r2, r1, lsr #8
    and   r2, r2, #0xFF                          @ get dllmax
    and   r1, r1, #0xFF                          @ get dllmin
    subs  r5, r2, r1                             @ dllmax - dllmin
    bmi   CBR3_START                             @ no valid margin found, retry again
    cmp   r5, #10                                @ (dllmax - dllmin) < 10
    blt   CBR3_START                             @ no enough margin found, retry again
    add   r2, r1, r2                             @ (dllmin[1] + dllmax[1] + 1) >> 1
    add   r2, r2, #0x01
    mov   r1, r2, lsr #1
    mov   r3, r1, lsl #8
    ldr   r1, [r0]                               @ store the dll search result
    bic   r1, r1, #0xFF000000
    bic   r1, r1, #0x00FF0000
    orr   r1, r1, r3, lsl #8
    str   r1, [r0]

    ldr   r0, =0x1e720008                        @ check DQL margin
    ldr   r1, [r0]
    mov   r2, r1, lsr #8
    and   r2, r2, #0xFF                          @ get dllmax
    and   r1, r1, #0xFF                          @ get dllmin
    subs  r5, r2, r1                             @ dllmax - dllmin
    bmi   CBR3_START                             @ no valid margin found, retry again
    cmp   r5, #10                                @ (dllmax - dllmin) < 10
    blt   CBR3_START                             @ no enough margin found, retry again
    add   r2, r1, r2                             @ (dllmin[0] + dllmax[0] + 1) >> 1
    add   r2, r2, #0x01
    mov   r1, r2, lsr #1
    ldr   r2, [r0]                               @ store the dll search result
    bic   r2, r2, #0xFF000000
    bic   r2, r2, #0x00FF0000
    orr   r2, r2, r1, lsl #16
    str   r2, [r0]
    orr   r3, r3, r1

    ldr   r0, =0x1e6e0068                        @ save the result dll value
    ldr   r1, [r0]
    bic   r1, r1, #0xFF000000
    bic   r1, r1, #0x00FF0000
    orr   r1, r1, r3, lsl #16
    str   r1, [r0]
    b     CBR4_START

.LTORG

/******************************************************************************
 Search the DQS input mask margin
 *****************************************************************************/
CBR4_START:
/* Debug - UART console message */
    ldr r0, =0x1e784000
    mov r1, #0x34                                @ '4'
    str r1, [r0]
/* Debug - UART console message */

    ldr   r0, =0x1e6e0074                        @ set the testing DRAM size = 4KB
    ldr   r1, =0x00000FFF
    str   r1, [r0]

    mov   r8, #0x00                              @ init MCR18[4]
    ldr   r1, =0x000000ff
    ldr   r0, =0x1e7200b0                        @ init MCR18[4]=0 max,min
    str   r1, [r0]
    ldr   r0, =0x1e7200b4                        @ init MCR18[4]=1 max,min
    str   r1, [r0]

    ldr   r0, =0x1e6e0018
    ldr   r1, [r0]
    bic   r1, r1, #0x0000001F
    str   r1, [r0]

    b     cbr4_scan_start

cbr4_next_maskdly:
    add   r8, r8, #0x01
    and   r2, r8, #0x01
    ldr   r0, =0x1e6e0018
    ldr   r1, [r0]
    bic   r1, r1, #0x0000001F
    orr   r1, r1, r2, lsl #4
    str   r1, [r0]
    cmp   r8, #0x02
    bge   CBR4_END

cbr4_scan_start:
    mov   r6, #0x00                              @ init pass count
    mov   r7, #0x00                              @ init mask delay

/****************************
 DQS Mask delay margin test loop
 ***************************/
cbr4_next_parameter:
    cmp   r7, #0x10                              @ max delay = 0xF
    bge   cbr4_next_maskdly
    ldr   r0, =0x1e6e0018                        @ load MCR18 parameter
    ldr   r1, [r0]
    bic   r1, r1, #0x0000000F
    orr   r1, r1, r7
    str   r1, [r0]
    add   r7, r7, #0x01

/* CBRScan3() start */
    adrl  r5, PATTERN_TABLE                      @ init pattern table index
/****************************
 Test pattern iteration loop
 ***************************/
cbr4_next_test_pattern:
    mov   r10, #2                                @ set the retry loop = 2 of each pattern
    ldr   r1, [r5]                               @ load test pattern
    ldr   r0, =0x1e6e007c
    str   r1, [r0]
    cmp   r1, #0x00                              @ the last data in pattern is 0x00
    bne   cbr4_test_burst

    and   r3, r7, #0xFF
    sub   r3, r3, #0x01                          @ we add 1 after loop check so we need to decrease 1
    add   r6, r6, #0x01                          @ increment pass count

    ldr   r0, =0x1e7200b0                        @ record pass window
    add   r0, r0, r8, lsl #2
    record_dll2_pass_range
    mov   r2, #0x01
    add   r1, r1, r2, lsl #16
    str   r1, [r0]
    b     cbr4_next_parameter

cbr4_test_pattern_fail:
    cmp   r6, #5                                 @ passcnt >= 5
    bge   cbr4_next_maskdly
    b     cbr4_next_parameter

/****************************
 Test fail retry loop
 ***************************/
cbr4_pattern_fail_retry:

/* CBRTest3() start */
cbr4_test_burst:
    ldr   r0, =0x1e6e0070
    ldr   r1, =0x00000000
    str   r1, [r0]
    ldr   r1, =0x000000C1
    str   r1, [r0]
    ldr   r3, =0x3000
cbr4_wait_engine_idle_0:
    ldr   r2, [r0]
    tst   r2, r3                                 @ D[12] = idle bit
    beq   cbr4_wait_engine_idle_0

    ldr   r2, [r0]                               @ read fail bit status
    mov   r1, #0x0
    str   r1, [r0]
    mov   r2, r2, lsr #13                        @ D[13] = fail bit
    cmp   r2, #0x00
    bne   cbr4_test_fail

cbr4_test_single:
    ldr   r0, =0x1e6e0070
    ldr   r1, =0x00000000
    str   r1, [r0]
    ldr   r1, =0x00000085
    str   r1, [r0]
    ldr   r3, =0x3000
cbr4_wait_engine_idle_1:
    ldr   r2, [r0]
    tst   r2, r3                                 @ D[12] = idle bit
    beq   cbr4_wait_engine_idle_1

    ldr   r2, [r0]                               @ read fail bit status
    mov   r1, #0x0
    str   r1, [r0]
    mov   r2, r2, lsr #13                        @ D[13] = fail bit
    cmp   r2, #0x00
    beq   cbr4_test_pass

/* CBRTest3() end */

cbr4_test_fail:
    subs  r10, r10, #1
    bne   cbr4_pattern_fail_retry
    b     cbr4_test_pattern_fail                 @ CBRScan3() return(0)

cbr4_test_pass:
    add   r5, r5, #0x04                          @ increase the test pattern index
    b     cbr4_next_test_pattern

CBR4_END:
    ldr   r0, =0x1e7200b0                        @ check mask margin
    ldr   r1, [r0]
    add   r0, r0, #0x04
    ldr   r2, [r0]
    ands  r6, r2, #0xFF                          @ get min of MCR18[4] = 1
    bne   cbr4_noset_delay
    ands  r5, r1, #0xFF                          @ get min of MCR18[4] = 0
    bne   cbr4_set_delay
    mov   r1, r1, lsr #8                         @ get max of MCR18[4] = 0
    and   r1, r1, #0xFF
    mov   r2, r2, lsr #8                         @ get max of MCR18[4] = 1
    and   r2, r2, #0xFF
    sub   r1, r1, r5
    sub   r2, r2, r6
    cmp   r1, r2
    bge   cbr4_noset_delay

cbr4_set_delay:
    ldr   r0, =0x1e6e0018
    ldr   r1, [r0]
    orr   r1, r1, #0x10
    str   r1, [r0]

cbr4_noset_delay:
    ldr   r0, =0x1e6e0070
    ldr   r1, =0x00000000
    str   r1, [r0]

/******************************************************************************
 CBR Finish
 *****************************************************************************/
/******************************************************************************
 Check DRAM Size
 *****************************************************************************/
    ldr   r0, =0x1e6e2070
    ldr   r1, [r0]
    bic   r1, r1, #0xFEFFFFFF                    @ bit[24]=1 => DDR2
    mov   r2, r1, lsr #24
    cmp   r2, #0x01
    beq   check_ddr2_size

    ldr   r0, =0x1e6e0004
    ldr   r5, [r0]
    bic   r5, r5, #0x00000003                    @ record MCR04
    orr   r1, r5, #0x3
    str   r1, [r0]                               @ set to 4Gbit
    ldr   r6, =0x003F2217
#if defined(CONFIG_DRAM_336)
    ldr   r6, =0x00361C13
#endif
    b     check_dram_size

check_ddr2_size:
    ldr   r0, =0x1e6e0004
    ldr   r5, [r0]
    bic   r5, r5, #0x00000023                    @ record MCR04
    orr   r1, r5, #0x23
    str   r1, [r0]                               @ set to 4Gbit
    ldr   r6, =0x3F2B1B16
#if defined(CONFIG_DRAM_336)
    ldr   r6, =0x3B231612
#endif

    ldr   r0, =0x40000000
    ldr   r1, =0x1817191A
    str   r1, [r0]
    ldr   r0, =0x40002000
    ldr   r1, =0x73616532
    str   r1, [r0]
    ldr   r0, =0x40000000
    ldr   r1, =0x1817191A
    ldr   r2, [r0]
    cmp   r1, r2
    bne   check_dram_size_end                    @ == 512Mbit
    orr   r5, r5, #0x20                          @ >= 1Gbit
    mov   r6, r6, lsr #8

check_dram_size:
    ldr   r0, =0x50100000
    ldr   r1, =0x41424344
    str   r1, [r0]
    ldr   r0, =0x48100000
    ldr   r1, =0x25262728
    str   r1, [r0]
    ldr   r0, =0x40100000
    ldr   r1, =0x191A1B1C
    str   r1, [r0]
    ldr   r0, =0x50100000
    ldr   r1, =0x41424344
    ldr   r2, [r0]
    cmp   r2, r1                                 @ == 4Gbit
    orreq r5, r5, #0x03
    moveq r6, r6, lsr #16
    beq   check_dram_size_end
    ldr   r0, =0x48100000
    ldr   r1, =0x25262728
    ldr   r2, [r0]
    cmp   r2, r1                                 @ == 2Gbit
    orreq r5, r5, #0x02
    moveq r6, r6, lsr #8
    beq   check_dram_size_end
    orr   r5, r5, #0x01                          @ == 1Gbit

check_dram_size_end:
    ldr   r0, =0x1e6e0004
    str   r5, [r0]
    ldr   r0, =0x1e6e0014
    ldr   r1, [r0]
    bic   r1, r1, #0x000000FF
    and   r6, r6, #0xFF
    orr   r1, r1, r6
    str   r1, [r0]

    ldr   r0, =0x1e6e0120                        @ VGA Compatible Mode
    ldr   r1, =0x000050C0                        @ 408 MHz
#if defined(CONFIG_DRAM_336)
    ldr   r1, =0x00004DC0
#endif
    str   r1, [r0]

/******************************************************************************
 Version Number
 *****************************************************************************/
    ldr   r0, =0x1e7200a8
    ldr   r1, =0x20150209                        @ released date
    str   r1, [r0]

    add   r0, r0, #4
    ldr   r1, =0x00000061                        @ released SDK version
    str   r1, [r0]

/******************************************************************************
 Calibration Code End
 ******************************************************************************/

set_scratch:
    /*Set Scratch register Bit 6 after ddr initial finished */
    ldr r0, =0x1e6e2040
    ldr r1, [r0]
    orr r1, r1, #0x41
    str r1, [r0]

/* Debug - UART console message */
    ldr r0, =0x1e784000
    mov r1, #0x44                                @ 'D'
    str r1, [r0]
    mov r1, #0x6F                                @ 'o'
    str r1, [r0]
    mov r1, #0x6E                                @ 'n'
    str r1, [r0]
    mov r1, #0x65                                @ 'e'
    str r1, [r0]
    mov r1, #0x0D                                @ '\r'
    str r1, [r0]
    mov r1, #0x0A                                @ '\n'
    str r1, [r0]
/* Debug - UART console message */

/******************************************************************************
 Solve PCIe ASPM issue, only applied to AST2300 series
 ******************************************************************************/
    ldr r0, =0x1e6e207c                          @ Check bounding for AST1150 existence
    ldr r1, [r0]
    mov r2, r1, lsr #24
    cmp r2, #0x01
    bne platform_exit                            @ not match AST2300
    bic r1, r1, #0xFFFFFCFF
    mov r1, r1, lsr #8
    cmp r1, #0x02
    beq platform_exit                            @ match AST1050

    ldr r0, =0x1e6e2004                          @ Disable I2C controller reset
    ldr r1, [r0]
    orr r1, r1, #0x04
    str r1, [r0]
    bic r1, r1, #0x04
    str r1, [r0]

    ldr r0, =0x1e78a054                          @ Check I2C bus state, if busy then quit
    ldr r1, [r0]
    mov r1, r1, lsr #17
    and r1, r1, #0x03
    cmp r1, #0x03
    bne platform_exit

    ldr r0, =0x1e78a040                          @ Init I2C1 controller
    mov r1, #0x01
    orr r1, r1, r1, lsl #16
    str r1, [r0]

    ldr r0, =0x1e78a044
    ldr r1, =0x77776704
    str r1, [r0]

    mov r1, #0x0
    ldr r0, =0x1e78a048
    str r1, [r0]
    ldr r0, =0x1e78a04c
    str r1, [r0]

    ldr r0, =0x1e78a050
    ldr r1, =0xFFFFFFFF
    str r1, [r0]

    ldr r0, =0x1e78a200                          @ Set AST1150 I2C password
    ldr r1, =0x00A88FA8
    str r1, [r0]

    ldr r0, =0x1e78a05c
    ldr r1, =0x00000200                          @ Enable buffer mode transfering 3 bytes
    str r1, [r0]

    ldr r0, =0x1e78a054
    ldr r1, =0x00000063                          @ Fire commmand
    str r1, [r0]

    ldr r0, =0x1e78a050
i2c_wait_cmddone_1:
    ldr r1, [r0]
    tst r1, #0x38
    beq i2c_wait_cmddone_1
    tst r1, #0x2A                                @ transmit error
    bne platform_exit2
    ldr r1, =0xFFFFFFFF
    str r1, [r0]

    ldr r0, =0x1e78a200                          @ Disable ASPM capability
    ldr r1, =0x04005DA8
    str r1, [r0]

    ldr r0, =0x1e78a204
    ldr r1, =0x00000024
    str r1, [r0]

    ldr r0, =0x1e78a05c
    ldr r1, =0x00000200                          @ Enable buffer mode transfering 3 bytes
    str r1, [r0]

    ldr r0, =0x1e78a054
    ldr r1, =0x00000063                          @ Fire commmand
    str r1, [r0]

    ldr r0, =0x1e78a050
i2c_wait_cmddone_2:
    ldr r1, [r0]
    tst r1, #0x38
    beq i2c_wait_cmddone_2
    tst r1, #0x2A                                @ transmit error
    bne platform_exit2
    ldr r1, =0xFFFFFFFF
    str r1, [r0]

platform_exit2:
    ldr r0, =0x1e78a040                          @ Disable I2C1 controller
    mov r1, #0x00
    str r1, [r0]

    b   platform_exit
.LTORG

platform_exit:
#ifdef      CONFIG_DRAM_ECC
    ldr r0, =0x1e6e0004
    ldr r1, [r0]
    orr r1, r1, #0x80
    str r1, [r0]

    ldr r0, =0x1e6e0054
    ldr r1, =0x05000000                          /* ECC protected memory size, default set at 80M   */
    str r1, [r0]

    ldr r0, =0x1e6e007C
    ldr r1, =0x00000000
    str r1, [r0]
    ldr r0, =0x1e6e0074
    str r1, [r0]

    ldr r0, =0x1e6e0070
    ldr r1, =0x00000221
    str r1, [r0]

    ldr r0, =0x1e6e0070
    ldr r2, =0x00001000
ECC_Init_Flag:
    ldr r1, [r0]
    tst r1, r2                                   @ D[12] = 1, Done
    beq ECC_Init_Flag

    ldr r0, =0x1e6e0070
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e0050
    ldr r1, =0x80000000
    str r1, [r0]

    ldr r0, =0x1e6e0050
    ldr r1, =0x00000000
    str r1, [r0]

    ldr r0, =0x1e6e0070
    ldr r1, =0x00000400
    str r1, [r0]
#endif
    ldr r0, =0x1e6e2008                          @ Set Video ECLK phase
    ldr r1, [r0]
    ldr r2, =0xfffffff3
    and r1, r1, r2
    orr r1, r1, #0x08
    str r1, [r0]

    ldr r0, =0x1e6e2004
    ldr r1, [r0]
    ldr r2, =0xFFBFFFFF                          @ Enable JTAG Master, solve ARM stucked by JTAG issue
    and r1, r1, r2
    str r1, [r0]

    ldr r0, =0x1e6e2048                          @ Set MAC interface delay timing
    ldr r1, =0x2255
    str r1, [r0]

    ldr r0, =0x1e6e2070                          @ Set MAC AHB bus clock
    ldr r1, [r0]
    mov r2, #0x04                                @ Default RMII, set MHCLK = HPLL/10
    tst r1, #0xC0
    movne r2, #0x02                              @ if RGMII,     set MHCLK = HPLL/6
    ldr r0, =0x1e6e2008
    ldr r1, [r0]
    bic r1, r1, #0x00070000
    orr r1, r1, r2, lsl #16
    str r1, [r0]

/* Test - DRAM initial time */
    ldr r0, =0x1e782040
    ldr r1, [r0]
    ldr r0, =0xFFFFFFFF
    sub r1, r0, r1
    ldr r0, =0x1e72009c
    str r1, [r0]
    ldr r0, =0x1e7200a4
    str r1, [r0]
    ldr r0, =0x1e782030
    ldr r1, [r0]
    bic r1, r1, #0x0000F000
    str r1, [r0]
/* Test - DRAM initial time */

/******************************************************************************
 Reset GPIO registers when watchdog reset
 ******************************************************************************/
    ldr r0, =0x1e6e207c                          @ Check Revision ID
    ldr r1, [r0]
    mov r1, r1, lsr #24
    cmp r1, #0x02
    bne platform_exit3                           @ not match AST2400

    ldr r0, =0x1e6e203c                          @ Check watchdog reset event
    ldr r1, [r0]
    and r1, r1, #0x06
    cmp r1, #0x0
    beq platform_exit3                           @ no watchdog reset event

    ldr r0, =0x1e6e209c                          @ Check watchdog GPIO selection
    ldr r1, [r0]
    mov r1, r1, lsr #21
    tst r1, #0x01
    beq platform_exit3                           @ no watchdog reset selection

    ldr r1, =0x00000000                          @ clear GPIO register reset by PRST_N
    ldr r2, =0xFFFFFFFF
    ldr r0, =0x1e780008
    str r1, [r0]
    ldr r0, =0x1e78000c
    str r1, [r0]
    ldr r0, =0x1e780010
    str r1, [r0]
    ldr r0, =0x1e780014
    str r1, [r0]
    ldr r0, =0x1e780018
    str r2, [r0]
    ldr r0, =0x1e780028
    str r1, [r0]
    ldr r0, =0x1e78002c
    str r1, [r0]
    ldr r0, =0x1e780030
    str r1, [r0]
    ldr r0, =0x1e780034
    str r1, [r0]
    ldr r0, =0x1e780038
    str r2, [r0]
    ldr r0, =0x1e780040
    str r1, [r0]
    ldr r0, =0x1e780044
    str r1, [r0]
    ldr r0, =0x1e780048
    str r1, [r0]
    ldr r0, =0x1e78004c
    str r1, [r0]
    ldr r0, =0x1e780050
    str r1, [r0]
    ldr r0, =0x1e780054
    str r1, [r0]
    ldr r0, =0x1e780058
    str r1, [r0]
    ldr r0, =0x1e780060
    str r1, [r0]
    ldr r0, =0x1e780064
    str r1, [r0]
    ldr r0, =0x1e780068
    str r1, [r0]
    ldr r0, =0x1e78006c
    str r1, [r0]
    ldr r0, =0x1e780090
    str r1, [r0]
    ldr r0, =0x1e780094
    str r1, [r0]
    ldr r0, =0x1e780098
    str r1, [r0]
    ldr r0, =0x1e78009c
    str r1, [r0]
    ldr r0, =0x1e7800a0
    str r1, [r0]
    ldr r0, =0x1e7800a4
    str r1, [r0]
    ldr r0, =0x1e7800a8
    str r2, [r0]
    ldr r0, =0x1e7800b0
    str r1, [r0]
    ldr r0, =0x1e7800b4
    str r1, [r0]
    ldr r0, =0x1e7800b8
    str r1, [r0]
    ldr r0, =0x1e7800e0
    str r1, [r0]
    ldr r0, =0x1e7800e4
    str r1, [r0]
    ldr r0, =0x1e7800e8
    str r1, [r0]
    ldr r0, =0x1e7800ec
    str r1, [r0]
    ldr r0, =0x1e7800f0
    str r1, [r0]
    ldr r0, =0x1e7800f4
    str r1, [r0]
    ldr r0, =0x1e7800f8
    str r2, [r0]
    ldr r0, =0x1e780100
    str r1, [r0]
    ldr r0, =0x1e780104
    str r1, [r0]
    ldr r0, =0x1e780108
    str r1, [r0]
    ldr r0, =0x1e780110
    str r1, [r0]
    ldr r0, =0x1e780114
    str r1, [r0]
    ldr r0, =0x1e780118
    str r1, [r0]
    ldr r0, =0x1e78011c
    str r1, [r0]
    ldr r0, =0x1e780120
    str r1, [r0]
    ldr r0, =0x1e780124
    str r1, [r0]
    ldr r0, =0x1e780128
    str r2, [r0]
    ldr r0, =0x1e780130
    str r1, [r0]
    ldr r0, =0x1e780134
    str r1, [r0]
    ldr r0, =0x1e780138
    str r1, [r0]
    ldr r0, =0x1e780140
    str r1, [r0]
    ldr r0, =0x1e780144
    str r1, [r0]
    ldr r0, =0x1e780148
    str r1, [r0]
    ldr r0, =0x1e78014c
    str r1, [r0]
    ldr r0, =0x1e780150
    str r1, [r0]
    ldr r0, =0x1e780154
    str r1, [r0]
    ldr r0, =0x1e780158
    str r2, [r0]
    ldr r0, =0x1e780160
    str r1, [r0]
    ldr r0, =0x1e780164
    str r1, [r0]
    ldr r0, =0x1e780168
    str r1, [r0]
    ldr r0, =0x1e780170
    str r1, [r0]
    ldr r0, =0x1e780174
    str r1, [r0]
    ldr r0, =0x1e780178
    str r1, [r0]
    ldr r0, =0x1e78017c
    str r1, [r0]
    ldr r0, =0x1e780180
    str r1, [r0]
    ldr r0, =0x1e780184
    str r1, [r0]
    ldr r0, =0x1e780188
    str r2, [r0]
    ldr r0, =0x1e780190
    str r1, [r0]
    ldr r0, =0x1e780194
    str r1, [r0]
    ldr r0, =0x1e780198
    str r1, [r0]
    ldr r0, =0x1e7801d0
    str r1, [r0]
    ldr r0, =0x1e7801d4
    str r1, [r0]

    ldr r0, =0x1e780204                          @ clear SGPIOM register reset by PRST_N
    str r1, [r0]
    ldr r0, =0x1e780208
    str r1, [r0]
    ldr r0, =0x1e78020c
    str r1, [r0]
    ldr r0, =0x1e780210
    str r1, [r0]
    ldr r0, =0x1e780214
    str r2, [r0]
    ldr r0, =0x1e780220
    str r1, [r0]
    ldr r0, =0x1e780224
    str r1, [r0]
    ldr r0, =0x1e780228
    str r1, [r0]
    ldr r0, =0x1e78022c
    str r1, [r0]
    ldr r0, =0x1e780230
    str r2, [r0]
    ldr r0, =0x1e78023c
    str r1, [r0]
    ldr r0, =0x1e780240
    str r1, [r0]
    ldr r0, =0x1e780244
    str r1, [r0]
    ldr r0, =0x1e780248
    str r1, [r0]
    ldr r0, =0x1e78024c
    str r2, [r0]
    ldr r0, =0x1e780254
    ldr r3, =0x01000040
    str r3, [r0]
    ldr r0, =0x1e780258
    str r1, [r0]
    ldr r0, =0x1e78025c
    str r1, [r0]
    ldr r0, =0x1e780260
    str r1, [r0]

    ldr r0, =0x1e780300                          @ clear SGPIOS register reset by PRST_N
    str r1, [r0]
    ldr r0, =0x1e780304
    str r1, [r0]
    ldr r0, =0x1e780308
    str r1, [r0]
    ldr r0, =0x1e78030c
    str r1, [r0]
    ldr r0, =0x1e780310
    str r1, [r0]
    ldr r0, =0x1e780314
    str r1, [r0]
    ldr r0, =0x1e780318
    str r2, [r0]
    ldr r0, =0x1e78031c
    str r2, [r0]
    ldr r0, =0x1e780320
    str r2, [r0]

platform_exit3:

/******************************************************************************
 SPI Timing Calibration, not applicable to AST2300 series
 ******************************************************************************/
    ldr r0, =0x1e6e207c                          @ Check Revision ID
    ldr r1, [r0]
    mov r1, r1, lsr #24
    cmp r1, #0x02
    blt platform_exit4                           @ not match AST2400 or later

    ldr r0, =0x1e6e2070                          @ Check SPI flash
    ldr r1, [r0]
    and r1, r1, #0x03
    cmp r1, #0x02
    bne platform_exit4

    mov r2, #0x0
    mov r6, #0x0
    mov r7, #0x0
    init_spi_checksum
spi_checksum_wait_0:
    ldr r1, [r0]
    tst r1, r2
    beq spi_checksum_wait_0
    ldr r0, =0x1e620090
    ldr r5, [r0]                                 @ record golden checksum
    ldr r0, =0x1e620080
    mov r1, #0x0
    str r1, [r0]

    ldr r0, =0x1e620010                          @ set to fast read mode
    ldr r1, =0x000B0041
    str r1, [r0]

    ldr r6, =0x00F7E6D0                          @ Init spiclk loop
    mov r8, #0x0                                 @ Init delay record

spi_cbr_next_clkrate:
    mov r6, r6, lsr #0x4
    cmp r6, #0x0
    beq spi_cbr_end

    mov r7, #0x0                                 @ Init delay loop
    mov r8, r8, lsl #4

spi_cbr_next_delay_s:
    mov r2, #0x8
    init_spi_checksum
spi_checksum_wait_1:
    ldr r1, [r0]
    tst r1, r2
    beq spi_checksum_wait_1
    ldr r0, =0x1e620090
    ldr r2, [r0]                                 @ read checksum
    ldr r0, =0x1e620080
    mov r1, #0x0
    str r1, [r0]
    cmp r2, r5
    bne spi_cbr_next_delay_e

    mov r2, #0x0
    init_spi_checksum
spi_checksum_wait_2:
    ldr r1, [r0]
    tst r1, r2
    beq spi_checksum_wait_2
    ldr r0, =0x1e620090
    ldr r2, [r0]                                 @ read checksum
    ldr r0, =0x1e620080
    mov r1, #0x0
    str r1, [r0]
    cmp r2, r5
    bne spi_cbr_next_delay_e

    orr r8, r8, r7                               @ record passed delay
    b   spi_cbr_next_clkrate

spi_cbr_next_delay_e:
    add r7, r7, #0x1
    cmp r7, #0x6
    blt spi_cbr_next_delay_s
    b   spi_cbr_next_clkrate

spi_cbr_end:
    ldr r0, =0x1e620094
    str r8, [r0]
    ldr r0, =0x1e620010
    mov r1, #0x0
    str r1, [r0]

platform_exit4:

    /* restore lr */
    mov lr, r4

    /* back to arch calling code */
    mov pc, lr
