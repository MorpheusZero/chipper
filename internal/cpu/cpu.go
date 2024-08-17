package cpu

import (
	"fmt"
	"math/rand"
	"os"
)

type CPUOptions struct {
	MAX_RAM_MEMORY_SIZE uint16
	MAX_STACK_SIZE      uint16
}

type CPU struct {
	display [32][64]uint8 // Display size 64x32 pixels

	ramMemory        []uint8   // RAM size by default of 4096 kb
	stack            []uint16  // STACK for 16 bit addresses
	variableRegister [16]uint8 // general purpose registers--numbered 0 through F hexadecimal (0 through 15 in decimal, called V0 through VF)
	key              [16]uint8 // input keys

	opCode         uint16 // Current opcode
	programCounter uint16 // PC - points at the current instruction in memory
	indexRegister  uint16 // I - point at locations in memory
	stackPointer   uint16 // current stack pointer

	delayTimer uint8 // 8 bit delay timer which decrements at a rate of 60hz (60 times per second) until it reaches 0.
	soundTimer uint8 // Sort of like the delay timer, but which also gives off a beeping sound as its not 0.

	shouldDraw bool
	beeper     func()
}

var fontSet = []uint8{
	0xF0, 0x90, 0x90, 0x90, 0xF0, //0
	0x20, 0x60, 0x20, 0x20, 0x70, //1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, //2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, //3
	0x90, 0x90, 0xF0, 0x10, 0x10, //4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, //5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, //6
	0xF0, 0x10, 0x20, 0x40, 0x40, //7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, //8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, //9
	0xF0, 0x90, 0xF0, 0x90, 0x90, //A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, //B
	0xF0, 0x80, 0x80, 0x80, 0xF0, //C
	0xE0, 0x90, 0x90, 0x90, 0xE0, //D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, //E
	0xF0, 0x80, 0xF0, 0x80, 0x80, //F
}

func (cpu CPU) InitCPU(options CPUOptions) CPU {
	cpu.ramMemory = make([]uint8, options.MAX_RAM_MEMORY_SIZE)
	cpu.stack = make([]uint16, options.MAX_STACK_SIZE)
	cpu.shouldDraw = true
	cpu.programCounter = 0x200
	cpu.beeper = func() {}

	// store fonts in ram memory
	copy(cpu.ramMemory[:len(fontSet)], fontSet)

	return cpu
}

func (cpu *CPU) Display() [32][64]uint8 {
	return cpu.display
}

func (cpu *CPU) Draw() bool {
	shouldDrawCurrent := cpu.shouldDraw
	cpu.shouldDraw = false
	return shouldDrawCurrent
}

func (cpu *CPU) AddBeep(fn func()) {
	cpu.beeper = fn
}

func (cpu *CPU) Key(num uint8, down bool) {
	if down {
		cpu.key[num] = 1
	} else {
		cpu.key[num] = 0
	}
}

func (cpu *CPU) Cycle() {
	cpu.opCode = (uint16(cpu.ramMemory[cpu.programCounter]) << 8) | uint16(cpu.ramMemory[cpu.programCounter+1])

	switch cpu.opCode & 0xF000 {
	case 0x0000:
		switch cpu.opCode & 0x000F {
		case 0x0000: // 0x00E0 Clears screen
			for i := 0; i < len(cpu.display); i++ {
				for j := 0; j < len(cpu.display[i]); j++ {
					cpu.display[i][j] = 0x0
				}
			}
			cpu.shouldDraw = true
			cpu.programCounter = cpu.programCounter + 2
		case 0x000E: // 0x00EE Returns from a subroutine
			cpu.stackPointer = cpu.stackPointer - 1
			cpu.programCounter = cpu.stack[cpu.stackPointer]
			cpu.programCounter = cpu.programCounter + 2
		default:
			fmt.Printf("Invalid opcode %X\n", cpu.opCode)
		}
	case 0x1000: // 0x1NNN Jump to address NNN
		cpu.programCounter = cpu.opCode & 0x0FFF
	case 0x2000: // 0x2NNN Calls subroutine at NNN
		cpu.stack[cpu.stackPointer] = cpu.programCounter // store current program counter
		cpu.stackPointer = cpu.stackPointer + 1          // increment stack pointer
		cpu.programCounter = cpu.opCode & 0x0FFF         // jump to address NNN
	case 0x3000: // 0x3XNN Skips the next instruction if VX equals NN
		if uint16(cpu.variableRegister[(cpu.opCode&0x0F00)>>8]) == cpu.opCode&0x00FF {
			cpu.programCounter = cpu.programCounter + 4
		} else {
			cpu.programCounter = cpu.programCounter + 2
		}
	case 0x4000: // 0x4XNN Skips the next instruction if VX doesn't equal NN
		if uint16(cpu.variableRegister[(cpu.opCode&0x0F00)>>8]) != cpu.opCode&0x00FF {
			cpu.programCounter = cpu.programCounter + 4
		} else {
			cpu.programCounter = cpu.programCounter + 2
		}
	case 0x5000: // 0x5XY0 Skips the next instruction if VX equals VY
		if cpu.variableRegister[(cpu.opCode&0x0F00)>>8] == cpu.variableRegister[(cpu.opCode&0x00F0)>>4] {
			cpu.programCounter = cpu.programCounter + 4
		} else {
			cpu.programCounter = cpu.programCounter + 2
		}
	case 0x6000: // 0x6XNN Sets VX to NN
		cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = uint8(cpu.opCode & 0x00FF)
		cpu.programCounter = cpu.programCounter + 2
	case 0x7000: // 0x7XNN Adds NN to VX
		cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] + uint8(cpu.opCode&0x00FF)
		cpu.programCounter = cpu.programCounter + 2
	case 0x8000:
		switch cpu.opCode & 0x000F {
		case 0x0000: // 0x8XY0 Sets VX to the value of VY
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x00F0)>>4]
			cpu.programCounter = cpu.programCounter + 2
		case 0x0001: // 0x8XY1 Sets VX to VX or VY
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] | cpu.variableRegister[(cpu.opCode&0x00F0)>>4]
			cpu.programCounter = cpu.programCounter + 2
		case 0x0002: // 0x8XY2 Sets VX to VX and VY
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] & cpu.variableRegister[(cpu.opCode&0x00F0)>>4]
			cpu.programCounter = cpu.programCounter + 2
		case 0x0003: // 0x8XY3 Sets VX to VX xor VY
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] ^ cpu.variableRegister[(cpu.opCode&0x00F0)>>4]
			cpu.programCounter = cpu.programCounter + 2
		case 0x0004: // 0x8XY4 Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't
			if cpu.variableRegister[(cpu.opCode&0x00F0)>>4] > 0xFF-cpu.variableRegister[(cpu.opCode&0x0F00)>>8] {
				cpu.variableRegister[0xF] = 1
			} else {
				cpu.variableRegister[0xF] = 0
			}
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] + cpu.variableRegister[(cpu.opCode&0x00F0)>>4]
			cpu.programCounter = cpu.programCounter + 2
		case 0x0005: // 0x8XY5 VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			if cpu.variableRegister[(cpu.opCode&0x00F0)>>4] > cpu.variableRegister[(cpu.opCode&0x0F00)>>8] {
				cpu.variableRegister[0xF] = 0
			} else {
				cpu.variableRegister[0xF] = 1
			}
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] - cpu.variableRegister[(cpu.opCode&0x00F0)>>4]
			cpu.programCounter = cpu.programCounter + 2
		case 0x0006: // 0x8XY6 Shifts VY right by one and stores the result to VX (VY remains unchanged). VF is set to the value of the least significant bit of VY before the shift
			cpu.variableRegister[0xF] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] & 0x1
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] >> 1
			cpu.programCounter = cpu.programCounter + 2
		case 0x0007: // 0x8XY7 Sets VX to VY minus VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			if cpu.variableRegister[(cpu.opCode&0x0F00)>>8] > cpu.variableRegister[(cpu.opCode&0x00F0)>>4] {
				cpu.variableRegister[0xF] = 0
			} else {
				cpu.variableRegister[0xF] = 1
			}
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x00F0)>>4] - cpu.variableRegister[(cpu.opCode&0x0F00)>>8]
			cpu.programCounter = cpu.programCounter + 2
		case 0x000E: // 0x8XYE Shifts VY left by one and copies the result to VX. VF is set to the value of the most significant bit of VY before the shift
			cpu.variableRegister[0xF] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] >> 7
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] << 1
			cpu.programCounter = cpu.programCounter + 2
		default:
			fmt.Printf("Invalid opcode %X\n", cpu.opCode)
		}
	case 0x9000: // 9XY0 Skips the next instruction if VX doesn't equal VY
		if cpu.variableRegister[(cpu.opCode&0x0F00)>>8] != cpu.variableRegister[(cpu.opCode&0x00F0)>>4] {
			cpu.programCounter = cpu.programCounter + 4
		} else {
			cpu.programCounter = cpu.programCounter + 2
		}
	case 0xA000: // 0xANNN Sets I to the address NNN
		cpu.indexRegister = cpu.opCode & 0x0FFF
		cpu.programCounter = cpu.programCounter + 2
	case 0xB000: // 0xBNNN Jumps to the address NNN plus V0
		cpu.programCounter = (cpu.opCode & 0x0FFF) + uint16(cpu.variableRegister[0x0])
	case 0xC000: // 0xCXNN Sets VX to the result of a bitwise and operation on a random number (Typically: 0 to 255) and NN
		cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = uint8(rand.Intn(256)) & uint8(cpu.opCode&0x00FF)
		cpu.programCounter = cpu.programCounter + 2
	case 0xD000: // 0xDXYN Draws a sprite at coordinate (VX, VY)
		x := cpu.variableRegister[(cpu.opCode&0x0F00)>>8]
		y := cpu.variableRegister[(cpu.opCode&0x00F0)>>4]
		h := cpu.opCode & 0x000F
		cpu.variableRegister[0xF] = 0
		var j uint16 = 0
		var i uint16 = 0
		for j = 0; j < h; j++ {
			pixel := cpu.ramMemory[cpu.indexRegister+j]
			for i = 0; i < 8; i++ {
				if (pixel & (0x80 >> i)) != 0 {
					if cpu.display[(y + uint8(j))][x+uint8(i)] == 1 {
						cpu.variableRegister[0xF] = 1
					}
					cpu.display[(y + uint8(j))][x+uint8(i)] ^= 1
				}
			}
		}
		cpu.shouldDraw = true
		cpu.programCounter = cpu.programCounter + 2
	case 0xE000:
		switch cpu.opCode & 0x00FF {
		case 0x009E: // 0xEX9E Skips the next instruction if the key stored in VX is pressed
			if cpu.key[cpu.variableRegister[(cpu.opCode&0x0F00)>>8]] == 1 {
				cpu.programCounter = cpu.programCounter + 4
			} else {
				cpu.programCounter = cpu.programCounter + 2
			}
		case 0x00A1: // 0xEXA1 Skips the next instruction if the key stored in VX isn't pressed
			if cpu.key[cpu.variableRegister[(cpu.opCode&0x0F00)>>8]] == 0 {
				cpu.programCounter = cpu.programCounter + 4
			} else {
				cpu.programCounter = cpu.programCounter + 2
			}
		default:
			fmt.Printf("Invalid opcode %X\n", cpu.opCode)
		}
	case 0xF000:
		switch cpu.opCode & 0x00FF {
		case 0x0007: // 0xFX07 Sets VX to the value of the delay timer
			cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = cpu.delayTimer
			cpu.programCounter = cpu.programCounter + 2
		case 0x000A: // 0xFX0A A key press is awaited, and then stored in VX
			pressed := false
			for i := 0; i < len(cpu.key); i++ {
				if cpu.key[i] != 0 {
					cpu.variableRegister[(cpu.opCode&0x0F00)>>8] = uint8(i)
					pressed = true
				}
			}
			if !pressed {
				return
			}
			cpu.programCounter = cpu.programCounter + 2
		case 0x0015: // 0xFX15 Sets the delay timer to VX
			cpu.delayTimer = cpu.variableRegister[(cpu.opCode&0x0F00)>>8]
			cpu.programCounter = cpu.programCounter + 2
		case 0x0018: // 0xFX18 Sets the sound timer to VX
			cpu.soundTimer = cpu.variableRegister[(cpu.opCode&0x0F00)>>8]
			cpu.programCounter = cpu.programCounter + 2
		case 0x001E: // 0xFX1E Adds VX to I
			if cpu.indexRegister+uint16(cpu.variableRegister[(cpu.opCode&0x0F00)>>8]) > 0xFFF {
				cpu.variableRegister[0xF] = 1
			} else {
				cpu.variableRegister[0xF] = 0
			}
			cpu.indexRegister = cpu.indexRegister + uint16(cpu.variableRegister[(cpu.opCode&0x0F00)>>8])
			cpu.programCounter = cpu.programCounter + 2
		case 0x0029: // 0xFX29 Sets I to the location of the sprite for the character in VX. Characters 0-F (in hexadecimal) are represented by a 4x5 font
			cpu.indexRegister = uint16(cpu.variableRegister[(cpu.opCode&0x0F00)>>8]) * 0x5
			cpu.programCounter = cpu.programCounter + 2
		case 0x0033: // 0xFX33 Stores the binary-coded decimal representation of VX, with the most significant of three digits at the address in I, the middle digit at I plus 1, and the least significant digit at I plus 2
			cpu.ramMemory[cpu.indexRegister] = cpu.variableRegister[(cpu.opCode&0x0F00)>>8] / 100
			cpu.ramMemory[cpu.indexRegister+1] = (cpu.variableRegister[(cpu.opCode&0x0F00)>>8] / 10) % 10
			cpu.ramMemory[cpu.indexRegister+2] = (cpu.variableRegister[(cpu.opCode&0x0F00)>>8] % 100) / 10
			cpu.programCounter = cpu.programCounter + 2
		case 0x0055: // 0xFX55 Stores V0 to VX (including VX) in memory starting at address I. I is increased by 1 for each value written
			for i := 0; i < int((cpu.opCode&0x0F00)>>8)+1; i++ {
				cpu.ramMemory[uint16(i)+cpu.indexRegister] = cpu.variableRegister[i]
			}
			cpu.indexRegister = ((cpu.opCode & 0x0F00) >> 8) + 1
			cpu.programCounter = cpu.programCounter + 2
		case 0x0065: // 0xFX65 Fills V0 to VX (including VX) with values from memory starting at address I. I is increased by 1 for each value written
			for i := 0; i < int((cpu.opCode&0x0F00)>>8)+1; i++ {
				cpu.variableRegister[i] = cpu.ramMemory[cpu.indexRegister+uint16(i)]
			}
			cpu.indexRegister = ((cpu.opCode & 0x0F00) >> 8) + 1
			cpu.programCounter = cpu.programCounter + 2
		default:
			fmt.Printf("Invalid opcode %X\n", cpu.opCode)
		}
	default:
		fmt.Printf("Invalid opcode %X\n", cpu.opCode)
	}

	if cpu.delayTimer > 0 {
		cpu.delayTimer = cpu.delayTimer - 1
	}
	if cpu.soundTimer > 0 {
		if cpu.soundTimer == 1 {
			cpu.beeper()
		}
		cpu.soundTimer = cpu.soundTimer - 1
	}
}

func (cpu *CPU) LoadProgram(fileName string) error {
	file, fileErr := os.OpenFile(fileName, os.O_RDONLY, 0777)
	if fileErr != nil {
		return fileErr
	}
	defer file.Close()

	fStat, fStatErr := file.Stat()
	if fStatErr != nil {
		return fStatErr
	}
	if int64(len(cpu.ramMemory)-512) < fStat.Size() { // program is loaded at 0x200
		return fmt.Errorf("Program size bigger than memory")
	}

	buffer := make([]byte, fStat.Size())
	if _, readErr := file.Read(buffer); readErr != nil {
		return readErr
	}

	for i := 0; i < len(buffer); i++ {
		cpu.ramMemory[i+512] = buffer[i]
	}

	return nil
}
