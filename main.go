package main

import "fmt"

const DEBUG_PRINT = true
const MEM_SIZE = 64*1024
type Registers [32]uint32

type Memory [MEM_SIZE]byte 

type CPU struct {
	reg Registers
	pc uint32
	mem Memory
}

func newCPU() *CPU {
	return &CPU {
		reg: [32]uint32{},
		pc: 0,
		mem: [MEM_SIZE]byte{},
	}
}

func (cpu *CPU) fetch() uint32 {
	return uint32(cpu.mem[cpu.pc]) << 24 | uint32(cpu.mem[cpu.pc + 1]) << 16 | uint32(cpu.mem[cpu.pc + 2]) << 8 | uint32(cpu.mem[cpu.pc + 3]) << 0
}

func (cpu *CPU) jtype_decode(instruction uint32) uint32 {
	a := 0x80000000 & instruction
	b := (instruction >> 21) & 0b11111111110
	l := ((instruction >> 20) & 0x1) << 11
	m := 0xff000 & instruction

	if a == 0 {
		return m | l | b 
	} else {
		return 0xfff00000 | m | l | b 
	}
}

func (cpu *CPU) btype_decode(instruction uint32) uint32 {
	a := 0x80000000 & instruction
	y := ((instruction >> 7) & 0x1) << 11
	u := (instruction >> 7) & 0b11110
	b := (instruction >> 20) & 0b11111100000

	if a == 0 {
		return y | b | u 
	} else {
		return 0xfffff000 | y | b | u
	}
}

func (cpu *CPU) stype_decode(instruction uint32) uint32 {
	a := 0x80000000 & instruction
	b := (instruction >> 20) & 0b111111100000
	u := (instruction >> 7) & 0b11111
	if a == 0 {
		return b | u 
	} else {
		return 0xfffff000 | b | u
	}
}

func (cpu *CPU) itype_decode(instruction uint32) uint32 {
	a := 0x80000000 & instruction
	if a == 0 {
		return (instruction >> 20)
	} else {
		return 0xfffff000 | (instruction >> 20)
	}
}

func (cpu *CPU) decode(instruction uint32) {
	cpu.reg[0] = 0

	opcode := 0x7f & instruction // select first 6bits 0b00000000000000000000000001111111
	rd := (0x7c0 & instruction) >> 7 // select next 4 bits 0b000000000000000000000111110000000	

	switch opcode {
	case 0x37: // lui U-Type
		imm := instruction & 0xfffff000
		cpu.reg[rd] = imm 
		cpu.pc += 4

	case 0x17: // auipc U-Type
		imm := instruction & 0xfffff000
		cpu.reg[rd] = cpu.pc + imm
		cpu.pc += 4

	case 0x6f: // jal J-Type
		cpu.reg[rd] = cpu.pc + 4
		imm_j := cpu.jtype_decode(instruction)
		cpu.pc = cpu.pc + imm_j

	case 0x63: // Btype
		imm_b := cpu.btype_decode(instruction)
		rs1 := (instruction >> 15) & 0b11111
		rs2 := (instruction >> 20) & 0b11111

		ins := (instruction >> 12) & 0b111
		switch ins {
		case 0b000: // beq
			if cpu.reg[rs1] == cpu.reg[rs2] {
				cpu.pc += imm_b
			} else {
				cpu.pc += 4
			}
		case 0b001: // bne
			if cpu.reg[rs1] != cpu.reg[rs2] {
				cpu.pc += imm_b
			} else {
				cpu.pc += 4
			}
		case 0b100: // blt
			if int32(cpu.reg[rs1]) < int32(cpu.reg[rs2]) {
				cpu.pc += imm_b
			} else {
				cpu.pc += 4
			}
		case 0b101: // bge
			if int32(cpu.reg[rs1]) >= int32(cpu.reg[rs2]) {
				cpu.pc += imm_b
			} else {
				cpu.pc += 4
			}	
		case 0b110: // bltu
			if cpu.reg[rs1] < cpu.reg[rs2] {
				cpu.pc += imm_b
			} else {
				cpu.pc += 4
			}
		case 0b111: // bgeu
			if cpu.reg[rs1] >= cpu.reg[rs2] {
				cpu.pc += imm_b
			} else {
				cpu.pc += 4
			}
		}
	case 0x23: // SType 
		imm_s := cpu.stype_decode(instruction)
		rs1 := (instruction >> 15) & 0b11111
		rs2 := (instruction >> 20) & 0b11111

		ins := (instruction >> 12) & 0b111
		switch ins {
			case 0b000: // sb
				addr := rs1 + imm_s
				cpu.mem_insert((cpu.mem_get(addr) & 0x00) | (0xFFFFFF00 & rs2), addr)
				cpu.pc += 4
			case 0b001: // sh 
				addr := rs1 + imm_s
				cpu.mem_insert((cpu.mem_get(addr) & 0x0000) | (0xFFFF0000 & rs2), addr)
				cpu.pc += 4
			case 0b010: // sw
				addr := rs1 + imm_s
				cpu.mem_insert(addr, rs2)
				cpu.pc += 4
		}
	case 0x33: // RType
		rs1 := (instruction >> 15) & 0b11111
		rs2 := (instruction >> 20) & 0b11111
		rd := (instruction >> 7) & 0b11111

		func3 := (instruction >> 12) & 0b111
		func7 := (instruction >> 25) & 0b1111111

		switch func3 {
			case 0b000: // add / sub
				if (func7 == 0) {
					cpu.reg[rd] = cpu.reg[rs1] + cpu.reg[rs2]
				} else {
					cpu.reg[rd] = cpu.reg[rs1] - cpu.reg[rs2]
				}
			case 0b001: // sll
				cpu.reg[rd] = cpu.reg[rs1] << (0b11111 & cpu.reg[rs2])
			case 0b010:
				if (int32(cpu.reg[rs1]) < int32(cpu.reg[rs2])) {
					cpu.reg[rd] = 1	
				} else {
					cpu.reg[rd] = 0
				}
			case 0b011: // stlu
				if (cpu.reg[rs1] < cpu.reg[rs2]) {
					cpu.reg[rd] = 1	
				} else {
					cpu.reg[rd] = 0
				}
			case 0b100: // xor
				cpu.reg[rd] = cpu.reg[rs1] ^ cpu.reg[rs2]
			case 0b101: // srl / sra
				if (func7 == 0) {
					cpu.reg[rd] = cpu.reg[rs1] >> (0b11111 & cpu.reg[rs2])
				} else {
					shiftAmount := uint32(0b11111 & cpu.reg[rs2])
					signBit := cpu.reg[rs1] & 0x80000000
					logicalShifted := cpu.reg[rs1] >> shiftAmount
					if signBit != 0 {
						mask := uint32(0xFFFFFFFF) << (32 - shiftAmount)
						cpu.reg[rd] = logicalShifted | mask
					} else {
						cpu.reg[rd] = logicalShifted
					}
				}
			case 0b110: // or
				cpu.reg[rd] = cpu.reg[rs1] | cpu.reg[rs2]
			case 0b111: // and
				cpu.reg[rd] = cpu.reg[rs1] & cpu.reg[rs2]
		}
		cpu.pc += 4
	case 0x3: // IType
		rs1 := (instruction >> 15) & 0b11111
		rd := (instruction >> 7) & 0b11111

		ins := (instruction >> 12) & 0b111
		imm_i := cpu.itype_decode(instruction)

		switch ins {
			case 0b000: // lb
				address := cpu.mem_get(rs1) + imm_i
				loadedByte := int32(cpu.mem_get(address))
				signExtendedByte := uint32(int8(loadedByte))
				cpu.reg[rd] = signExtendedByte
			case 0b001: // lh
				address := cpu.mem_get(rs1) + imm_i
				loadedHalfWord := int32(uint16(cpu.mem_get(address)) | uint16(cpu.mem_get(address+1))<<8)
				signExtendedHalfWord := uint32(int16(loadedHalfWord))
				cpu.reg[rd] = signExtendedHalfWord
			case 0b010: // lw
				address := cpu.reg[rs1] + imm_i
				loadedWord := uint32(cpu.mem_get(address) | uint32(cpu.mem_get(address+1))<<8 | uint32(cpu.mem_get(address+2))<<16 | uint32(cpu.mem_get(address+3))<<24)
				cpu.reg[rd] = loadedWord
			case 0b100: // lbu
				address := cpu.reg[rs1] + imm_i
				loadedByte := cpu.mem_get(address)
				cpu.reg[rd] = loadedByte 
			case 0b101: // lhu
				address := cpu.reg[rs1] + imm_i 
				loadedHalfWord := cpu.mem_get(address) | cpu.mem_get(address+1)<<8
				cpu.reg[rd] = loadedHalfWord 
		}
		cpu.pc += 4
	case 0x13: // IType
		rs1 := (instruction >> 15) & 0b11111
		rd := (instruction >> 7) & 0b11111

		ins := (instruction >> 12) & 0b111
		imm_i := cpu.itype_decode(instruction)

		switch ins {
			case 0b000: // addi
				cpu.reg[rd] = cpu.reg[rs1] + imm_i
			case 0b010: // slti
				if (int32(cpu.reg[rs1]) < int32(imm_i)) {
					cpu.reg[rd] = 1 
				} else {
					cpu.reg[rd] = 0
				}
			case 0b011: // sltiu
				if (cpu.reg[rs1] < imm_i) {
					cpu.reg[rd] = 1 
				} else {
					cpu.reg[rd] = 0
				}
			case 0b100: // xori
				cpu.reg[rd] = cpu.reg[rs1] ^ imm_i
			case 0b110: // ori
				cpu.reg[rd] = cpu.reg[rs1] | imm_i
			case 0b111: //andi
				cpu.reg[rd] = cpu.reg[rs1] & imm_i
			
			case 0b001: // slli
				shamt := imm_i & 0b11111
				cpu.reg[rd] = cpu.reg[rs1] << shamt
			case 0b101: // srli / srai
				shamt := imm_i & 0b11111
				if ((imm_i >> 5) > 0) {
					shiftAmount := uint32(0b11111 & imm_i)
					signBit := cpu.reg[rs1] & 0x80000000
					logicalShifted := cpu.reg[rs1] >> shiftAmount
					if signBit != 0 {
						mask := uint32(0xFFFFFFFF) << (32 - shiftAmount)
						cpu.reg[rd] = logicalShifted | mask
					} else {
						cpu.reg[rd] = logicalShifted
					}
				} else {
					cpu.reg[rd] = cpu.reg[rs1] >> shamt
				}
		}
		cpu.pc += 4


	default:
		cpu.pc += 4 // Change to err later
	}

	cpu.debug_print(instruction, opcode, rd)
}

func (cpu *CPU) debug_print(instruction uint32, opcode uint32, rd uint32) {
	if DEBUG_PRINT {
		fmt.Printf(" %X %d\n", opcode, rd)
		fmt.Printf(" 0x%08X \n", instruction)
		for idx, reg := range cpu.reg {
			fmt.Printf("x%d: %08X ", idx, reg)
			if (idx + 1) % 4 == 0 {
				fmt.Println()
			}
		}

		fmt.Printf("pc: %08X\n", cpu.pc)
	}
}

func (cpu *CPU) mem_insert(instruction uint32, index uint32) {
	cpu.mem[0 + index*4] = byte(instruction >> 24) & 0xFF
	cpu.mem[1 + index*4] = byte(instruction >> 16) & 0xFF
	cpu.mem[2 + index*4] = byte(instruction >> 8)  & 0xFF
	cpu.mem[3 + index*4] = byte(instruction >> 0)  & 0xFF
}

func (cpu *CPU) mem_get(index uint32) uint32 {
	return uint32(cpu.mem[index]) << 24 | uint32(cpu.mem[index + 1]) << 16 | uint32(cpu.mem[index + 2]) << 8 | uint32(cpu.mem[index + 3]) << 0
}

func main() {
	cpu := newCPU()

	cpu.mem_insert(0x123450b7, 0) // lui x1,0x12345
	cpu.mem_insert(0x10011b17, 1) // auipc x22,0x10001
	cpu.mem_insert(0x008002ef, 2) // jal x5,0
	cpu.mem_insert(0x00520463, 3)

	var input string
	for {	
        fmt.Scanln(&input)
		instruction := cpu.fetch()
		cpu.decode(instruction)
	}
}
