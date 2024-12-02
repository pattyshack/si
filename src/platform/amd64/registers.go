package amd64

import (
	"github.com/pattyshack/chickadee/platform"
)

var (
	rsp = platform.NewStackPointerRegister("rsp")

	rbp = platform.NewGeneralRegister("rbp", false)
	rax = platform.NewGeneralRegister("rax", false)
	rbx = platform.NewGeneralRegister("rbx", false)
	rcx = platform.NewGeneralRegister("rcx", false)
	rdx = platform.NewGeneralRegister("rdx", false)
	rsi = platform.NewGeneralRegister("rsi", false)
	rdi = platform.NewGeneralRegister("rdi", false)
	r8  = platform.NewGeneralRegister("r8", false)
	r9  = platform.NewGeneralRegister("r9", false)
	r10 = platform.NewGeneralRegister("r10", false)
	r11 = platform.NewGeneralRegister("r11", false)
	r12 = platform.NewGeneralRegister("r12", false)
	r13 = platform.NewGeneralRegister("r13", false)
	r14 = platform.NewGeneralRegister("r14", false)
	r15 = platform.NewGeneralRegister("r15", false)

	xmm0  = platform.NewFloatRegister("xmm0")
	xmm1  = platform.NewFloatRegister("xmm1")
	xmm2  = platform.NewFloatRegister("xmm2")
	xmm3  = platform.NewFloatRegister("xmm3")
	xmm4  = platform.NewFloatRegister("xmm4")
	xmm5  = platform.NewFloatRegister("xmm5")
	xmm6  = platform.NewFloatRegister("xmm6")
	xmm7  = platform.NewFloatRegister("xmm7")
	xmm8  = platform.NewFloatRegister("xmm8")
	xmm9  = platform.NewFloatRegister("xmm9")
	xmm10 = platform.NewFloatRegister("xmm10")
	xmm11 = platform.NewFloatRegister("xmm11")
	xmm12 = platform.NewFloatRegister("xmm12")
	xmm13 = platform.NewFloatRegister("xmm13")
	xmm14 = platform.NewFloatRegister("xmm14")
	xmm15 = platform.NewFloatRegister("xmm15")

	ArchitectureRegisters = platform.NewArchitectureRegisters(
		rsp,
		rbp, rax, rbx, rcx, rdx, rsi, rdi, r8, r9, r10, r11, r12, r13, r14, r15,
		xmm0, xmm1, xmm2, xmm3, xmm4, xmm5, xmm6, xmm7,
		xmm8, xmm9, xmm10, xmm11, xmm12, xmm13, xmm14, xmm15)
)
