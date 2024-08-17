package config

import (
	"flag"
	"fmt"
	"log"
	"strconv"
)

type ChipperConfig struct {
	DEBUG               bool
	MAX_RAM_MEMORY_SIZE uint16
	MAX_STACK_SIZE      uint16
}

func (c ChipperConfig) InitConfig() ChipperConfig {
	debug := flag.Bool("debug", true, "when TRUE, will output more verbose info to the console")
	maxRamMemorySizeStr := flag.String("max_ram_memory_size", "4096", "the maximum amount of ram memory size to use (in kilobytes)")
	maxStackSizeStr := flag.String("max_stack_size", "16", "the maximum amount of stack size to use (in bits)")

	flag.Parse()

	maxRamMemorySizeInt, err := strconv.ParseUint(*maxRamMemorySizeStr, 10, 16)
	if err != nil {
		fmt.Println()
		log.Panicf("an error occured when reading the max_ram_memory_size", err)
	}

	maxStackSizeInt, err := strconv.ParseUint(*maxStackSizeStr, 10, 16)
	if err != nil {
		fmt.Println()
		log.Panicf("an error occured when reading the max_stack_size", err)
	}

	c.MAX_RAM_MEMORY_SIZE = uint16(maxRamMemorySizeInt)
	c.MAX_STACK_SIZE = uint16(maxStackSizeInt)
	c.DEBUG = *debug

	return c
}
