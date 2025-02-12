package x64

import (
	"encoding/binary"

	arch "github.com/pattyshack/chickadee/architecture"
)

const (
	immediateMaxBytes = 8 // mov can support up to imm64
)

var (
	// x64's encoded X.Reg mapping.  The lowest 3 bits are encoded in mod r/m
	// while the 4th bit is encoded in rex
	//
	// https://wiki.osdev.org/X86-64_Instruction_Encoding#Registers
	xRegMapping = map[*arch.Register]int{
		rax:   0,
		rcx:   1,
		rdx:   2,
		rbx:   3,
		rsp:   4,
		rbp:   5,
		rsi:   6,
		rdi:   7,
		r8:    8,
		r9:    9,
		r10:   10,
		r11:   11,
		r12:   12,
		r13:   13,
		r14:   14,
		r15:   15,
		xmm0:  0,
		xmm1:  1,
		xmm2:  2,
		xmm3:  3,
		xmm4:  4,
		xmm5:  5,
		xmm6:  6,
		xmm7:  7,
		xmm8:  8,
		xmm9:  9,
		xmm10: 10,
		xmm11: 11,
		xmm12: 12,
		xmm13: 13,
		xmm14: 14,
		xmm15: 15,
	}
)

func directAddressInstruction(
	opCode []byte,
	regXReg int, // could also be op code extension
	rmXReg int,
	immediate interface{}, // int/uint
) []byte {
	// 0x40 is a fixed bit pattern, 0x08 (W-bit) indicates 64-bit operand size.
	rexPrefix := 0x48
	// Register-direct addressing mode
	modRMPrefix := 0xc0

	// reg's rex extension bit (R-bit) and modR/M reg bits
	rexRegX := (regXReg & 0x08) >> 1
	modRMReg := (regXReg & 0x07) << 3

	// rm's rex extension bit (B-bit) and modR/M rm bits
	rexRmX := (rmXReg & 0x08) >> 3
	modRMRm := rmXReg & 0x07

	rex := byte(rexPrefix | rexRegX | rexRmX)
	modRM := byte(modRMPrefix | modRMReg | modRMRm)

	immMaxBytes := 0
	if immediate != nil {
		immMaxBytes = immediateMaxBytes
	}

	result := make([]byte, 2+len(opCode)+immMaxBytes)

	result[0] = rex

	idx := 1
	for _, opByte := range opCode {
		result[idx] = opByte
		idx++
	}

	result[idx] = modRM
	idx++

	if immediate != nil {
		size, err := binary.Encode(result[idx:], binary.LittleEndian, immediate)
		if err != nil {
			panic("cannot encode immediate: " + err.Error())
		}
		idx += size
	}

	return result[:idx]
}

// <int/uint dest> += <int/uint src>
//
// NOTE: For now, we'll only use 2-address code ADD (0x01) for adding.
//
// TODO: test 3-address code form via LEA (0x8d).
func addIntRegister(
	dest *arch.Register,
	src *arch.Register,
) []byte {
	return directAddressInstruction(
		[]byte{0x01},
		xRegMapping[src],
		xRegMapping[dest],
		nil)
}

// <int/uint dest> += <int/uint immediate>
//
// NOTE: For now, we'll only use 2-address code ADD (0x81 /0) for adding.
//
// TODO: test 3-address code form via LEA (0x8d).
func addIntImmediate(
	dest *arch.Register,
	immediate int32, // sign-extended to 64-bit
) []byte {
	return directAddressInstruction(
		[]byte{0x81},
		0,
		xRegMapping[dest],
		immediate)
}

// <int/uint dest> -= <int/uint src>
//
// NOTE: For now, we'll only use 2-address code SUB (0x29) for subtracting.
//
// TODO: test 3-address code form via LEA (0x8d).
func subIntRegister(
	dest *arch.Register,
	src *arch.Register,
) []byte {
	return directAddressInstruction(
		[]byte{0x29},
		xRegMapping[src],
		xRegMapping[dest],
		nil)
}

// <int/uint dest> -= <int/uint immediate>
//
// NOTE: For now, we'll only use 2-address code SUB (0x81 /5) for subtracting.
//
// TODO: test 3-address code form via LEA (0x8d).
func subIntImmediate(
	dest *arch.Register,
	immediate int32, // sign-extended to 64-bit
) []byte {
	return directAddressInstruction(
		[]byte{0x81},
		5,
		xRegMapping[dest],
		immediate)
}

// <int/uint dest> ^= <int/uint src>
func xorIntRegister(
	dest *arch.Register,
	src *arch.Register,
) []byte {
	return directAddressInstruction(
		[]byte{0x31},
		xRegMapping[src],
		xRegMapping[dest],
		nil)
}

// <int/uint dest> ^= <int/uint immediate>
func xorIntImmediate(
	dest *arch.Register,
	immediate int32, // sign-extended to 64-bit
) []byte {
	return directAddressInstruction(
		[]byte{0x81},
		6,
		xRegMapping[dest],
		immediate)
}

// <int/uint dest> |= <int/uint src>
func orIntRegister(
	dest *arch.Register,
	src *arch.Register,
) []byte {
	return directAddressInstruction(
		[]byte{0x09},
		xRegMapping[src],
		xRegMapping[dest],
		nil)
}

// <int/uint dest> |= <int/uint immediate>
func orIntImmediate(
	dest *arch.Register,
	immediate int32, // sign-extended to 64-bit
) []byte {
	return directAddressInstruction(
		[]byte{0x81},
		1,
		xRegMapping[dest],
		immediate)
}

// <int/uint dest> &= <int/uint src>
func andIntRegister(
	dest *arch.Register,
	src *arch.Register,
) []byte {
	return directAddressInstruction(
		[]byte{0x21},
		xRegMapping[src],
		xRegMapping[dest],
		nil)
}

// <int/uint dest> &= <int/uint immediate>
func andIntImmediate(
	dest *arch.Register,
	immediate int32, // sign-extended to 64-bit
) []byte {
	return directAddressInstruction(
		[]byte{0x81},
		4,
		xRegMapping[dest],
		immediate)
}

// TODO MUL / DIV / REM / SHL / SHR
