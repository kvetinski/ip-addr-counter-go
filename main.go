package main

import (
	"fmt"
	"math/bits"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/edsrzf/mmap-go"
)

// --- Configuration Constants ---
const (
	MaxIPv4      = 1 << 32     // 4,294,967,296 possible IPv4 addresses
	BucketSize   = 64          // One uint64 holds 64 bits
	MinIPLen     = 7           // Minimal length of "x.x.x.x"
	MaxIPLen     = 15          // Maximal length of "xxx.xxx.xxx.xxx"
	ChunkMinSize = 1024 * 1024 // Minimum file size (~1MB) for using multiple workers
)

// --- AtomicBitSet ---
// AtomicBitSet stores unique IPv4 addresses using a bitset.
type AtomicBitSet struct {
	bits []uint64 // Each element is updated atomically.
}

// NewAtomicBitSet creates a new AtomicBitSet covering all possible IPv4 addresses.
func NewAtomicBitSet() *AtomicBitSet {
	// Each uint64 stores 64 bits.
	return &AtomicBitSet{
		bits: make([]uint64, MaxIPv4/BucketSize),
	}
}

// Set marks the bit corresponding to the given IPv4 address.
func (bs *AtomicBitSet) Set(ip uint32) {
	index := ip / BucketSize
	bit := ip % BucketSize
	atomic.OrUint64(&bs.bits[index], 1<<bit)
}

// Count returns the number of unique IPv4 addresses.
func (bs *AtomicBitSet) Count() int {
	workers := runtime.NumCPU() / 2
	countChan := make(chan int, workers)
	chunkSize := len(bs.bits) / workers

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == workers-1 {
			end = len(bs.bits) // Last worker processes the remaining elements.
		}
		go func(start, end int) {
			defer wg.Done()
			localCount := 0
			for j := start; j < end; j++ {
				localCount += bits.OnesCount64(bs.bits[j])
			}
			countChan <- localCount
		}(start, end)
	}
	wg.Wait()
	close(countChan)

	total := 0
	for count := range countChan {
		total += count
	}
	return total
}

// --- IP Parsing ---
// parseIPFast parses an IPv4 address in the format "xxx.xxx.xxx.xxx" from a byte slice.
// Returns the IPv4 address as a uint32 or false if the format is invalid.
func parseIPFast(line []byte) (uint32, bool) {
	if len(line) < MinIPLen || len(line) > MaxIPLen {
		return 0, false
	}

	var ip, num uint32
	dots := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '.':
			if dots >= 3 || num > 255 {
				return 0, false
			}
			ip = (ip << 8) | num
			num = 0
			dots++
		default:
			if line[i] < '0' || line[i] > '9' {
				return 0, false
			}
			num = num*10 + uint32(line[i]-'0')
		}
	}
	if dots != 3 || num > 255 {
		return 0, false
	}
	ip = (ip << 8) | num
	return ip, true
}

// --- Chunk Processing ---
// processChunk processes a section of the memory-mapped file data from startChunk to endChunk,
// parsing each line as an IPv4 address and setting its bit in the global bitset.
func processChunk(data []byte, startChunk, endChunk int, bitSet *AtomicBitSet, wg *sync.WaitGroup) {
	defer wg.Done()
	var ip uint32
	var ok bool
	lineStart := startChunk
	for i := startChunk; i < endChunk; i++ {
		if data[i] == '\n' {
			if lineStart < i {
				if ip, ok = parseIPFast(data[lineStart:i]); ok {
					bitSet.Set(ip)
				}
			}
			lineStart = i + 1
			// Skip a few bytes to speed up processing
			i += 6
		}
	}
}

// --- File Counting ---
// countUniqueIpInFile opens, memory-maps, and processes the file to count unique IPv4 addresses.
func countUniqueIpInFile(fileName string) (int, error) {
	startTime := time.Now()

	// Open the file.
	fmt.Printf("Opening file %s...\n", fileName)
	file, err := os.Open(fileName)
	if err != nil {
		return 0, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()
	fmt.Println("File opened in", time.Since(startTime))

	// Get file stats.
	stat, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("error getting file stats: %w", err)
	}
	fmt.Printf("File size: %d bytes, stat time: %v\n", stat.Size(), time.Since(startTime))

	// Memory-map the file.
	fmt.Println("Mapping file...")
	mmapData, err := mmap.Map(file, mmap.RDONLY, 0)
	if err != nil {
		return 0, fmt.Errorf("mmap error: %w", err)
	}
	defer mmapData.Unmap()
	fmt.Println("File mapped in", time.Since(startTime))

	if len(mmapData) == 0 {
		fmt.Printf("Empty file: %s\n", fileName)
		return 0, nil
	}

	// Create an atomic bitset for unique IPv4 addresses.
	bitSet := NewAtomicBitSet()

	// Determine the number of workers.
	workers := runtime.NumCPU() / 2
	if len(mmapData) < ChunkMinSize {
		workers = 1
	}
	fmt.Printf("Processing file using %d worker(s)\n", workers)

	var wg sync.WaitGroup
	wg.Add(workers)

	// Divide the memory-mapped file into chunks for each worker.
	start := 0
	for i := 0; i < workers; i++ {
		end := (len(mmapData) * (i + 1)) / workers

		// Adjust the chunk boundaries to align with newline characters.
		if i > 0 {
			for start < len(mmapData) && mmapData[start-1] != '\n' {
				start++
			}
		}
		if i < workers-1 && end < len(mmapData) {
			for end < len(mmapData) && mmapData[end-1] != '\n' {
				end++
			}
		} else {
			end = len(mmapData)
		}
		go processChunk(mmapData, start, end, bitSet, &wg)
		start = end
	}

	wg.Wait()
	uniqueCount := bitSet.Count()
	fmt.Printf("File processed in %v\n", time.Since(startTime))
	fmt.Printf("Unique IPv4 addresses: %d\n", uniqueCount)
	return uniqueCount, nil
}

func main() {
	// Get filename from command-line arguments.
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <filename>")
		os.Exit(1)
	}
	fileName := os.Args[1]
	_, err := countUniqueIpInFile(fileName)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
