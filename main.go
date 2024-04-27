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
	b := instruction >> 20
	m := 0xff000 & instruction

	if a == 0 {
		return m | b
	} else {
		return 0xfff00000 | m | b 
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

func main() {
	cpu := newCPU()

	cpu.mem_insert(0x123450b7, 0) // lui x1,0x12345
	cpu.mem_insert(0x10011b17, 1) // auipc x22,0x10001
	cpu.mem_insert(0x008002ef, 2) // jal x5,0x1c

	var input string
	for {	
        fmt.Scanln(&input)
		instruction := cpu.fetch()
		cpu.decode(instruction)
	}
}
