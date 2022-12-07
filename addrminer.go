package main

import (
	"os"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
	"strings"
	"runtime"
	"time"
)

func sanitize(s string) string {
	if strings.HasPrefix(s, "0x") {
		return s[2:]
	} else {
		return s
	}
}

func toBytes(s string) []byte {
	myBytes, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return myBytes
}

func computeCreate2Address(deployer []byte, salt []byte, bytecodeHash []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte{0xff})
	h.Write(deployer)
	h.Write(salt)
	h.Write(bytecodeHash)
	// Get the resulting hash, take the last 20 bytes
	return h.Sum(nil)[12:]
}

func computeAddressDeployedByProxy(proxyContractAddress []byte) []byte {
	nonce1 := []byte{0x01}
	encoded, err := rlp.EncodeToBytes([][]byte{proxyContractAddress, nonce1})
	if err != nil {
		fmt.Println(err)
	}
	h := sha3.NewLegacyKeccak256()
	h.Write(encoded)
	// Get the resulting hash, take the last 20 bytes
	return h.Sum(nil)[12:]
}

// Using the bytecode of the proxy provided by reference create3 factories
func proxyByteCodeHash() []byte {
	proxyByteCode := []byte{0x67, 0x36, 0x3d, 0x3d, 0x37, 0x36, 0x3d, 0x34, 0xf0, 0x3d, 0x52, 0x60, 0x08, 0x60, 0x18, 0xf3}
	h := sha3.NewLegacyKeccak256()
	h.Write(proxyByteCode)
	return h.Sum(nil)
}

func hash(i int) []byte {
	buff := new(bytes.Buffer)
	bigOrLittleEndian := binary.BigEndian
	err := binary.Write(buff, bigOrLittleEndian, uint64(i))
	if err != nil {
		fmt.Println(err)
	}
	h := sha3.NewLegacyKeccak256()
	h.Write(buff.Bytes())
	return h.Sum(nil)
}

func toInt(b []byte) int {
	n := binary.LittleEndian.Uint64(b)
	return int(n)
}

func outputFile() *os.File {
	file, err := os.OpenFile("./addresses.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Could not open addresses.csv")
	}
	return file
}

func computeOutputFromSalt(
	factoryAddressBytes []byte, 
	proxyByteCodeHash []byte, 
	desiredPrefix []byte,
	c chan int,
	printChannel chan string) {
	for {
		select {
		case saltCandidate, _ := <- c:
			salt := hash(saltCandidate)
			create2ProxyAddress := computeCreate2Address(factoryAddressBytes, salt, proxyByteCodeHash)
			create3Address := computeAddressDeployedByProxy(create2ProxyAddress)
			if bytes.HasPrefix(create3Address, desiredPrefix) {
				printChannel <- "0x" + hex.EncodeToString(salt) + ",0x" + hex.EncodeToString(create3Address) + "\n"
			}
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func printFromChannel(printChannel chan string) {
	file := outputFile()
	defer file.Close() // close file later
	for {
		select {
		case printStatement, _ := <- printChannel:
			fmt.Print(printStatement)
			file.WriteString(printStatement)
			if _, err := file.WriteString(printStatement); err != nil {
				fmt.Println(err)
			}
		default:
			time.Sleep(1000 * time.Millisecond)
		}
	}
}

func main() {
	givenFactoryAddress := flag.String("factoryAddress", "", "Address of the create3 factory being used")
	prefix := flag.String("prefix", "00", "hexstring prefix that you want, eg. '00'. Even number of characters.")
	minOccur := flag.Int("minOccur", 1, "minimum number of times the `prefix` must have appeared")
	miningSalt := flag.Int("miningSalt", 1, "Salt to use for starting to search for salts")
	step := flag.Int("step", 1, "Step of increment for each try")
	flag.Parse()

	// Prepare parameters
	if *givenFactoryAddress == "" {
		fmt.Println("Error: you must specify a value for the 'factoryAddress' flag")
		return
	}
	factoryAddress := sanitize(*givenFactoryAddress)
	factoryAddressBytes := toBytes(factoryAddress)
	desiredPrefix := toBytes(strings.Repeat(*prefix, *minOccur))
	proxyByteCodeHash := proxyByteCodeHash()

	// Create mining threads
    numCpu := runtime.NumCPU()
	runtime.GOMAXPROCS(numCpu)
	c := make(chan int, numCpu * 500) // Use channeled buffer to cap memory footprint. Large cap to reduce wait-time in goroutines
	printChannel := make(chan string, numCpu * 500)
	for i := 0; i < numCpu; i++ {
		go computeOutputFromSalt(factoryAddressBytes, proxyByteCodeHash, desiredPrefix, c, printChannel)
	}

	// Create printing thread
	go printFromChannel(printChannel)

	// Start generating hashes to be tried
	hashOfMiningSalt := hash(*miningSalt)
	saltCandidate := toInt(hashOfMiningSalt)
	for {
		c <- saltCandidate
		saltCandidate += *step
	}
}
