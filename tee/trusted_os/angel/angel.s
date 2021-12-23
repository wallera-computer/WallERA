/*
register int reg0 asm("r0");
register int reg1 asm("r1");

reg0 = 0x18;    // angel_SWIreason_ReportException
reg1 = 0x20026; // ADP_Stopped_ApplicationExit

asm("svc 0x00123456");  // make semihosting call
*/

#include "go_asm.h"

// func SemihostingShutdown()
TEXT ·SemihostingShutdown(SB),$0-1
	MOVW	$const_targetSysExit, R0
	MOVW	$const_targetSysExitArgument, R1

	WORD	$0xEF123456 // svc 0x00123456

	RET

