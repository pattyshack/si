package x64

import (
	"encoding/binary"

	arch "github.com/pattyshack/chickadee/architecture"
)

// Resources:
//
// https://www.felixcloutier.com/x86/
// http://x86asm.net/articles/x86-64-tour-of-intel-manuals/
// https://wiki.osdev.org/X86-64_Instruction_Encoding

const (
	prefix16BitOperand = byte(0x66)
	rexPrefix          = byte(0x40)
	rexWBit            = byte(0x08)
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

	// https://www.felixcloutier.com/x86/syscall
	syscall = []byte{0x0f, 0x05}

	// https://www.felixcloutier.com/x86/ret
	ret = []byte{0xc3}
)

// https://www.felixcloutier.com/x86/nop
func nop(length int) []byte {
	remaining := length
	result := make([]byte, 0, length)
	for remaining > 0 {
		switch remaining {
		case 1:
			result = append(result, 0x90)
			return result
		case 2:
			result = append(result, 0x66, 0x90)
			return result
		case 3:
			result = append(result, 0x0f, 0x1f, 0x00)
			return result
		case 4:
			result = append(result, 0x0f, 0x1f, 0x40, 0x00)
			return result
		case 5:
			result = append(result, 0x0f, 0x1f, 0x44, 0x00, 0x00)
			return result
		case 6:
			result = append(result, 0x66, 0x0f, 0x1f, 0x44, 0x00, 0x00)
			return result
		case 7:
			result = append(result, 0x0f, 0x1f, 0x80, 0x00, 0x00, 0x00, 0x00)
			return result
		case 8:
			result = append(result, 0x0f, 0x1f, 0x84, 0x00, 0x00, 0x00, 0x00, 0x00)
			return result
		default: // 9 or longer
			result = append(
				result,
				0x66, 0x0f, 0x1f, 0x84, 0x00, 0x00, 0x00, 0x00, 0x00)
			remaining -= 9
		}
	}

	return result
}

func directAddressInstruction(
	operandSize int,
	opCode []byte,
	regXReg int, // could also be op code extension
	rmXReg int,
	immediate interface{}, // int/uint
) []byte {
	immMaxBytes := 0
	if immediate != nil {
		immMaxBytes = 8 // mov can support imm64
	}

	result := make([]byte, 3+len(opCode)+immMaxBytes)
	idx := 0

	rex := rexPrefix

	switch operandSize {
	case 8:
	case 16:
		result[0] = prefix16BitOperand
		idx = 1
	case 32:
	case 64:
		rex |= rexWBit
	default:
		panic("should never happen")
	}

	// reg's rex extension bit (R-bit) and modR/M reg bits
	rexRegX := (regXReg & 0x08) >> 1
	modRMReg := (regXReg & 0x07) << 3

	// rm's rex extension bit (B-bit) and modR/M rm bits
	rexRmX := (rmXReg & 0x08) >> 3
	modRMRm := rmXReg & 0x07

	rex |= byte(rexRegX | rexRmX)

	// NOTE: rex makes AH / CH / DH / BH inaccessible for 8-bit operand
	if operandSize == 8 || rex != rexPrefix {
		result[idx] = rex
		idx++
	}

	for _, opByte := range opCode {
		result[idx] = opByte
		idx++
	}

	modRMPrefix := 0xc0 // Register-direct addressing mode
	result[idx] = byte(modRMPrefix | modRMReg | modRMRm)
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

func opCode(operandSize int, opCode []byte, opCode8Bit []byte) []byte {
	if operandSize == 8 {
		return opCode8Bit
	}
	return opCode
}

func immediate(operandSize int, val uint64, allow64 bool) interface{} {
	var imm interface{} = val
	switch operandSize {
	case 8:
		imm = uint8(val)
	case 16:
		imm = uint16(val)
	case 32:
		imm = uint32(val)
	case 64:
		if !allow64 {
			imm = uint32(val)
		}
	default:
		panic("should never happen")
	}

	return imm
}

// <int dest> = -<int dest>
//
// https://www.felixcloutier.com/x86/neg
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: F7 /3
// 64-bit:      REX.W + F7 /3
func negSignedInt(
	operandSize int,
	dest *arch.Register,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0xf7},
		3,
		xRegMapping[dest],
		nil)
}

// <uint/int dest> = ~<uint/int dest>
//
// https://www.felixcloutier.com/x86/not
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: F7 /2
// 64-bit:      REX.W + F7 /2
func bitwiseNotInt(
	operandSize int,
	dest *arch.Register,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0xf7},
		2,
		xRegMapping[dest],
		nil)
}

// <sign-extended int dest> = <int src>
//
// https://www.felixcloutier.com/x86/movsx
//
// 8-bit src operand:  (REX.W +) 0F BE /r
// 16-bit src operand: (REX.W +) 0F BF /r
// 32-bit src operand: REX.W + 63 /r
func extendSignedInt(
	destOperandSize int,
	dest *arch.Register,
	srcOperandSize int,
	src *arch.Register,
) []byte {
	if destOperandSize <= srcOperandSize {
		panic("should never happen")
	}

	if destOperandSize != 64 {
		destOperandSize = 32
	}

	switch srcOperandSize {
	case 8:
		return directAddressInstruction(
			destOperandSize,
			[]byte{0x0f, 0xbe},
			xRegMapping[dest],
			xRegMapping[src],
			nil)
	case 16:
		return directAddressInstruction(
			destOperandSize,
			[]byte{0x0f, 0xbf},
			xRegMapping[dest],
			xRegMapping[src],
			nil)
	case 32:
		return directAddressInstruction(
			destOperandSize,
			[]byte{0x63},
			xRegMapping[dest],
			xRegMapping[src],
			nil)
	default:
		panic("should never happen")
	}
}

// <zero-extended uint dest> = <uint src>
//
// https://www.felixcloutier.com/x86/movzx
// https://www.felixcloutier.com/x86/mov (for uint32 -> uint64)
//
// 8-bit src operand:
//
//	movzx r32, r/m8: 0F B6 /r
//
// 16-bit src operand:
//
//	movzx r32, r/m16: 0F B7 /r
//
// 32-bit src operand:
//
//	mov r32, r/m32: 8B /r
//
// NOTE: the upper 32 bits are automatically zero-ed when a 32-bit operand
// instruction is used (see Intel manual, Volume 1, Section 3.4.1.1
// General-Purpose Registers in 64-Bit Mode).
func extendUnsignedInt(
	dest *arch.Register,
	srcOperandSize int,
	src *arch.Register,
) []byte {
	switch srcOperandSize {
	case 8:
		return directAddressInstruction(
			32,
			[]byte{0x0f, 0xb6},
			xRegMapping[dest],
			xRegMapping[src],
			nil)
	case 16:
		return directAddressInstruction(
			32,
			[]byte{0x0f, 0xb7},
			xRegMapping[dest],
			xRegMapping[src],
			nil)
	case 32:
		return directAddressInstruction(
			32,
			[]byte{0x8b},
			xRegMapping[dest],
			xRegMapping[src],
			nil)
	default:
		panic("should never happen")
	}
}

// <int/uint dest> += <int/uint src>
//
// https://www.felixcloutier.com/x86/add
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 03 /r
// 64-bit:      REX.W + 03 /r
//
// NOTE: For now, we'll only use 2-address code ADD (0x01) for adding.
//
// TODO: test 3-address code form via LEA (0x8d).
func addInt(
	operandSize int,
	dest *arch.Register,
	src *arch.Register,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x03},
		xRegMapping[dest],
		xRegMapping[src],
		nil)
}

// <int/uint dest> += <int/uint immediate>
//
// https://www.felixcloutier.com/x86/add
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 81 /0 id
// 64-bit:      REX.W + 81 /0 id
//
// NOTE: For now, we'll only use 2-address code ADD (0x81 /0) for adding.
//
// TODO: test 3-address code form via LEA (0x8d).
func addIntImmediate(
	operandSize int,
	dest *arch.Register,
	value uint64, // sign-extended to 64-bit
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x81},
		0,
		xRegMapping[dest],
		immediate(operandSize, value, false))
}

// <int/uint dest> -= <int/uint src>
//
// https://www.felixcloutier.com/x86/sub
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 2B /r
// 64-bit:      REX.W + 2B /r
//
// NOTE: For now, we'll only use 2-address code SUB (0x29) for subtracting.
//
// TODO: test 3-address code form via LEA (0x8d).
func subInt(
	operandSize int,
	dest *arch.Register,
	src *arch.Register,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x2b},
		xRegMapping[dest],
		xRegMapping[src],
		nil)
}

// <int/uint dest> -= <int/uint immediate>
//
// https://www.felixcloutier.com/x86/sub
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 81 /5 id
// 64-bit:      REX.W + 81 /5 id
//
// NOTE: For now, we'll only use 2-address code SUB (0x81 /5) for subtracting.
//
// TODO: test 3-address code form via LEA (0x8d).
func subIntImmediate(
	operandSize int,
	dest *arch.Register,
	value uint64, // sign-extended to 64-bit
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x81},
		5,
		xRegMapping[dest],
		immediate(operandSize, value, false))
}

// <int/uint dest> *= <int/uint src>
//
// https://www.felixcloutier.com/x86/imul
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 0F AF /r
// 64-bit:      REX.W + 0F AF /r
func mulInt(
	operandSize int,
	dest *arch.Register,
	src *arch.Register,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x0f, 0xaf},
		xRegMapping[dest],
		xRegMapping[src],
		nil)
}

// <int/uint dest> *= <int/uint immediate>
//
// https://www.felixcloutier.com/x86/mul
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 69 /r id
// 64-bit:      REX.W + 69 /r id
//
// NOTE: Even though imul's 3-operand form supports separate src/dest registers,
// i.e., <dest> = <src> * <imm>, for consistency/simplicity, we'll restrict
// dest to reuse the src register.
func mulIntImmediate(
	operandSize int,
	dest *arch.Register,
	value uint64, // sign-extended to 64-bit
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	xReg := xRegMapping[dest]
	return directAddressInstruction(
		operandSize,
		[]byte{0x69},
		xReg,
		xReg,
		immediate(operandSize, value, false))
}

// (<uint quotient RAX>, <uint remainder RDX>) =
//
//	<uint upper RDX>:<uint lower RAX> / <uint divisor>
//
// https://www.felixcloutier.com/x86/movzx
// https://www.felixcloutier.com/x86/xor
// https://www.felixcloutier.com/x86/div
//
// (Sign extension sensitive)
//
// 8-bit (uses 32-bit div):
//
//	movzx eax, al:                0F B6 /r
//	movzx <divisor>d, <divisor>b: 0F B6 /r
//	xor edx, edx:                 33 /r
//	div <divisor>d:               F7 /6
//
// 16-bit:
//
//	xor edx, edx:   33 /r
//	div <divisor>w: F7 /6
//
// 32-bit:
//
//	xor edx, edx:   33 /r
//	div <divisor>d: F7 /6
//
// 64-bit:
//
//	xor rdx, rdx:   REX.W + 33 /r
//	div <divisor>q: REX.W + F7 /6
func divRemUnsignedInt(
	operandSize int,
	divisor *arch.Register,
) []byte {
	instructions := make([]byte, 0, 12)

	if operandSize == 8 {
		operandSize = 32

		instructions = append(
			instructions,
			extendUnsignedInt(rax, 8, rax)...)

		if divisor != rax {
			instructions = append(
				instructions,
				extendUnsignedInt(divisor, 8, divisor)...)
		}
	}

	instructions = append(instructions, bitwiseXorInt(operandSize, rdx, rdx)...)

	instructions = append(
		instructions,
		directAddressInstruction(
			operandSize,
			[]byte{0xf7},
			6,
			xRegMapping[divisor],
			nil)...)

	return instructions
}

// (<uint quotient RAX>, <uint remainder RDX>) =
//
//	<uint upper RDX>:<uint lower RAX> / <uint divisor>
//
// https://www.felixcloutier.com/x86/movsx
// https://www.felixcloutier.com/x86/cwd:cdq:cqo
// https://www.felixcloutier.com/x86/div
//
// (Sign extension sensitive)
//
// 8-bit (uses 32-bit idiv):
//
//	movsx eax, al:                0F BE /r
//	movsx <divisor>d, <divisor>b: 0F BE /r
//	cdq:                          99
//	idiv <divisor>d:              F7 /7
//
// 16-bit:
//
//	cwd:             99
//	idiv <divisor>w: F7 /7
//
// 32-bit:
//
//	cdq:             99
//	idiv <divisor>d: F7 /7
//
// 64-bit:
//
//	cqo:             REX.W + 99
//	idiv <divisor>q: REX.W + F7 /7
func divRemSignedInt(
	operandSize int,
	divisor *arch.Register,
) []byte {
	instructions := make([]byte, 0, 11)

	if operandSize == 8 {
		operandSize = 32

		instructions = append(
			instructions,
			extendSignedInt(32, rax, 8, rax)...)

		if divisor != rax {
			instructions = append(
				instructions,
				extendSignedInt(32, divisor, 8, divisor)...)
		}
	}

	// cwd / cdq / cqo (signed-extend rax to rdx)
	switch operandSize {
	case 16:
		instructions = append(instructions, prefix16BitOperand, 0x99)
	case 32:
		instructions = append(instructions, 0x99)
	case 64:
		instructions = append(instructions, rexPrefix|rexWBit, 0x99)
	default:
		panic("should never happen")
	}

	instructions = append(
		instructions,
		directAddressInstruction(
			operandSize,
			[]byte{0xf7},
			7,
			xRegMapping[divisor],
			nil)...)

	return instructions
}

// <int/uint dest> ^= <int/uint src>
//
// https://www.felixcloutier.com/x86/xor
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 33 /r
// 64-bit:      REX.W + 33 /r
func bitwiseXorInt(
	operandSize int,
	dest *arch.Register,
	src *arch.Register,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x33},
		xRegMapping[dest],
		xRegMapping[src],
		nil)
}

// <int/uint dest> ^= <int/uint immediate>
//
// https://www.felixcloutier.com/x86/xor
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 81 /6 ib
// 64-bit:      REX.W + 81 /6 id
func bitwiseXorIntImmediate(
	operandSize int,
	dest *arch.Register,
	value uint64, // sign-extended to 64-bit
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x81},
		6,
		xRegMapping[dest],
		immediate(operandSize, value, false))
}

// <int/uint dest> |= <int/uint src>
//
// https://www.felixcloutier.com/x86/or
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 0B /r
// 64-bit:      REX.W + 0B /r
func bitwiseOrIntRegister(
	operandSize int,
	dest *arch.Register,
	src *arch.Register,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x0b},
		xRegMapping[dest],
		xRegMapping[src],
		nil)
}

// <int/uint dest> |= <int/uint immediate>
//
// https://www.felixcloutier.com/x86/or
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 81 /1 id
// 64-bit:      REX.W + 81 /1 id
func bitwiseOrIntImmediate(
	operandSize int,
	dest *arch.Register,
	value uint64, // sign-extended to 64-bit
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x81},
		1,
		xRegMapping[dest],
		immediate(operandSize, value, false))
}

// <int/uint dest> &= <int/uint src>
//
// https://www.felixcloutier.com/x86/and
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 23 /r
// 64-bit:      REX.W + 23 /r
func bitwiseAndInt(
	operandSize int,
	dest *arch.Register,
	src *arch.Register,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x23},
		xRegMapping[dest],
		xRegMapping[src],
		nil)
}

// <int/uint dest> &= <int/uint immediate>
//
// https://www.felixcloutier.com/x86/and
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: 81 /4 id
// 64-bit:      REX.W + 81 /4 id
func bitwiseAndIntImmediate(
	operandSize int,
	dest *arch.Register,
	value uint64, // sign-extended to 64-bit
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0x81},
		4,
		xRegMapping[dest],
		immediate(operandSize, value, false))
}

// <int/uint dest> <<= <uint8 RCX>
//
// https://www.felixcloutier.com/x86/sal:sar:shl:shr
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: D3 /4
// 64-bit: REX.W + D3 /4
func shiftLeftInt(
	operandSize int,
	dest *arch.Register,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0xd3},
		4,
		xRegMapping[dest],
		nil)
}

// <int/uint dest> <<= <uint8 immediate>
//
// https://www.felixcloutier.com/x86/sal:sar:shl:shr
//
// (Not sign extension sensitive)
//
// 8/16/32-bit: C1 /4 ib
// 64-bit:      REX.W + C1 /4 ib
func shiftLeftIntImmediate(
	operandSize int,
	dest *arch.Register,
	immediate uint8,
) []byte {
	if operandSize != 64 {
		operandSize = 32
	}

	return directAddressInstruction(
		operandSize,
		[]byte{0xc1},
		4,
		xRegMapping[dest],
		immediate)
}

// <int dest> >>= <uint8 RCX> (aka sar)
//
// https://www.felixcloutier.com/x86/sal:sar:shl:shr
//
// (Sign extension sensitive)
//
// 8-bit:  REX + D2 /7
// 16-bit: D3 /7
// 32-bit: D3 /7
// 64-bit: REX.W + D3 /7
func shiftRightSignedInt(
	operandSize int,
	dest *arch.Register,
) []byte {
	return directAddressInstruction(
		operandSize,
		opCode(operandSize, []byte{0xd3}, []byte{0xd2}),
		7,
		xRegMapping[dest],
		nil)
}

// <int dest> >>= <uint8 immediate> (aka sar)
//
// https://www.felixcloutier.com/x86/sal:sar:shl:shr
//
// (Sign extension sensitive)
//
// 8-bit:  REX + C0 /7 ib
// 16-bit: C1 /7 ib
// 32-bit: C1 /7 ib
// 64-bit: REX.W + C1 /7 ib
func shiftRightSignedIntImmediate(
	operandSize int,
	dest *arch.Register,
	immediate uint8,
) []byte {
	return directAddressInstruction(
		operandSize,
		opCode(operandSize, []byte{0xc1}, []byte{0xc0}),
		7,
		xRegMapping[dest],
		immediate)
}

// <uint dest> >>= <uint8 RCX> (aka shl)
//
// https://www.felixcloutier.com/x86/sal:sar:shl:shr
//
// (Sign extension sensitive)
//
// 8-bit:  REX + D2 /5
// 16-bit: D3 /5
// 32-bit: D3 /5
// 64-bit: REX.W + D3 /5
func shiftRightUnsignedInt(
	operandSize int,
	dest *arch.Register,
) []byte {
	return directAddressInstruction(
		operandSize,
		opCode(operandSize, []byte{0xd3}, []byte{0xd2}),
		5,
		xRegMapping[dest],
		nil)
}

// <int dest> >>= <uint8 immediate> (aka shl)
//
// https://www.felixcloutier.com/x86/sal:sar:shl:shr
//
// (Sign extension sensitive)
//
// 8-bit:  REX + C0 /5 ib
// 16-bit: C1 /5 ib
// 32-bit: C1 /5 ib
// 64-bit: REX.W + C1 /5 ib
func shiftRightUnsignedIntImmediate(
	operandSize int,
	dest *arch.Register,
	immediate uint8,
) []byte {
	return directAddressInstruction(
		operandSize,
		opCode(operandSize, []byte{0xc1}, []byte{0xc0}),
		5,
		xRegMapping[dest],
		immediate)
}
