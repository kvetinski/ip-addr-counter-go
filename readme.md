Below is an example of a README.md file for your application:

---

# Unique IP Counter

A fast and efficient Go application that counts the number of unique IPv4 addresses from a file using memory-mapped I/O and an atomic bitset. The application is optimized for large files (potentially gigabytes in size) and leverages concurrency to maximize performance.

## Features

- **Fast IP Parsing:** Optimized function for parsing IPv4 addresses in the format `xxx.xxx.xxx.xxx`.
- **Memory Mapping:** Uses mmap to efficiently access large files without loading the entire file into memory.
- **Atomic BitSet:** Uses an atomic bitset to store unique IPv4 addresses in a memory‑efficient manner.
- **Concurrent Processing:** Splits the input file into chunks processed in parallel using multiple goroutines.
- **Configurable Constants:** All hardcoded values (IP length limits, bucket size, etc.) are defined as constants for easy configuration.
- **Command‑Line Arguments:** Reads the input filename from the command line.

## Installation

1. Clone the repository:

   ```sh
   git clone https://github.com/kvetinski/ip-addr-counter-go.git
   cd ip-addr-counter-go
   ```

2. Install dependencies:

   ```sh
   go get github.com/edsrzf/mmap-go
   ```

## Usage

Build the application:

```sh
go build -o ipcounter
```

Run the application with a filename as an argument:

```sh
./ipcounter <path_your_ip_addresses_file>
```

The application will output the total number of unique IPv4 addresses along with processing time.

## How It Works

1. **File Mapping:**  
   The application opens the specified file and uses memory mapping to efficiently access its contents.

2. **Concurrent Processing:**  
   The file is divided into chunks based on the number of available CPU cores (with a minimum chunk size threshold). Each chunk is processed concurrently by separate goroutines.

3. **Fast IP Parsing:**  
   A custom IP parsing function (`parseIPFast`) quickly converts each IPv4 address (in "xxx.xxx.xxx.xxx" format) to a `uint32` value.

4. **Atomic BitSet:**  
   The unique IPv4 addresses are stored in an atomic bitset. Each IP is represented as a bit in the bitset; if the bit is not already set, it is marked and the unique IP count is incremented.

5. **Results:**  
   After all chunks are processed, the application counts the total set bits in the bitset to determine the number of unique IP addresses.

## Contributing

Contributions are welcome! Please fork the repository and submit your pull requests. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
