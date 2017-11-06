package main

import (
	"encoding/binary"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
)

// Output directory.
var outDir string

func main() {
	var cpuprofile = flag.String("c", "", "Write cpu profile to file")
	var mpqFile = flag.String("m", "diabdat.mpq", "Path to Diablo 1 mpq")
	flag.StringVar(&outDir, "dir", "diabdat", "Output directory")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	file, err := ioutil.ReadFile(*mpqFile)
	if err != nil {
		log.Fatalln("\n", err, "\nPlese use -h for help\n")
	}

	mpq, err := NewMpq(file)
	if err != nil {
		log.Fatalln(err)
	}

	// Best run:
	//test$ time mpq
	//real	0m1.998s
	//user	0m4.678s
	//sys	0m1.660s
	var wg sync.WaitGroup
	for _, hv := range mpq.HashTable {
		// Get File Name
		name, ok := mpq.PrecalcList[FilePrecalc{hv.Name1, hv.Name2}]
		if ok {
			wg.Add(1)
			go func(name string, index uint32) {
				defer wg.Done()
				mpq.ExtractFile(name, index)
			}(name, hv.BlockIndex)
		}
	}
	wg.Wait()

	//Version with worker goroutines
	//test$ time mpq
	//real	0m2.110s
	//user	0m4.830s
	//sys	0m1.620s
	//ch := make(chan fileEntry)
	//var wg sync.WaitGroup
	//for i := 0; i < runtime.NumCPU(); i++ {
	//	wg.Add(1)
	//	// Worket to extract files
	//	go func(c chan fileEntry, w *sync.WaitGroup) {
	//		defer w.Done()
	//		for fe := range c {
	//			mpq.ExtractFile(fe.name, fe.index)
	//		}
	//	}(ch, &wg)
	//}
	//for _, hv := range mpq.HashTable {
	//	// Get File Names
	//	name, ok := mpq.PrecalcList[FilePrecalc{hv.Name1, hv.Name2}]
	//	if ok {
	//		ch <- fileEntry{hv.BlockIndex, name}
	//	}
	//}
	//close(ch)
	//wg.Wait()

}

func (mpq Mpq) extractSingelFile(fileName string, index uint32) {
	block := mpq.BlockTable[index]

	fileKey := DecryptFileKey(fileName, block.FilePos, block.FileSize, block.Flags)
	data := mpq.RawData[block.FilePos : block.FilePos+block.FileSize]

	file := createFile(fileName)
	defer file.Close()

	parts := uint32(len(data) / 4096)
	for i := uint32(0); i < parts+1; i++ {
		if block.Flags&MPQ_FILE_ENCRYPTED != 0 {
			if i == parts {
				DecryptBlock(data[i*4096:], fileKey+i)
			} else {
				DecryptBlock(data[i*4096:i*4096+4096], fileKey+i)
			}
		}
	}
	file.Write(data)
}

// Method ExtractFile extracts the file fileName with the blocktable index index
func (mpq Mpq) ExtractFile(fileName string, index uint32) {
	if filepath.Ext(fileName) == ".wav" ||
		filepath.Ext(fileName) == ".smk" ||
		filepath.Ext(fileName) == ".mpq" {
		mpq.extractSingelFile(fileName, index)
		return
	}

	block := mpq.BlockTable[index]
	sectorSize := ((block.FileSize + 4095) / 4096) + 1
	sectorTable := mpq.RawData[block.FilePos : block.FilePos+sectorSize*4]
	fileKey := DecryptFileKey(fileName, block.FilePos, block.FileSize, block.Flags) - 1

	if block.Flags&MPQ_FILE_ENCRYPTED != 0 {
		DecryptBlock(sectorTable, fileKey)
	}

	file := createFile(fileName)
	defer file.Close()

	for i := 0; i <= len(sectorTable)-8; i += 4 {
		cur := binary.LittleEndian.Uint32(sectorTable[i : i+4])
		next := binary.LittleEndian.Uint32(sectorTable[i+4 : i+8])

		from := block.FilePos + cur
		to := block.FilePos + next

		sector := mpq.RawData[from:to]

		if block.Flags&MPQ_FILE_ENCRYPTED != 0 {
			DecryptBlock(sector, fileKey+uint32(i/4)+1)
		}

		if i == len(sectorTable)-8 {
			if block.Flags&MPQ_FILE_IMPLODE != 0 && sector[0] == 0 && sector[1] == 6 {
				decompSector := decompress(sector)[:block.FileSize%4096]
				file.Write(decompSector)
				return
			}
			file.Write(sector)
			return
		}

		// Dont decompress if sector is 4096 or if it is not imploded
		if len(sector) < 4096 && block.Flags&MPQ_FILE_IMPLODE != 0 {
			file.Write(decompress(sector))
		} else {
			file.Write(sector)
		}
	}
}

func DecryptFileKey(FileName string, MpqPos, FileSize, Flags uint32) uint32 {
	FileName = strings.Replace(FileName, "\\", "/", -1)
	FileName = filepath.Base(FileName)

	FileKey := HashString(FileName, MPQ_HASH_FILE_KEY)

	if Flags&MPQ_FILE_FIX_KEY != 0 {
		FileKey = (FileKey + MpqPos) ^ FileSize
	}
	return FileKey
}

func DecryptBlock(block []byte, key uint32) {
	const SIZEOF_UINT32 = 4
	var seed uint32 = 0xEEEEEEEE

	for i := 0; i < len(block)/SIZEOF_UINT32; i++ {
		tmpval := binary.LittleEndian.Uint32(block[i*SIZEOF_UINT32 : (i+1)*SIZEOF_UINT32])
		seed += StormBuffer[MPQ_HASH_KEY2_MIX+(key&0xFF)]
		tmpval = tmpval ^ (seed + key)
		value32 := tmpval
		key = (((key ^ 0xFFFFFFFF) << 0x15) + 0x11111111) | (key >> 0x0B)
		seed = value32 + seed + (seed << 5) + 3

		block[i*SIZEOF_UINT32+0] = byte(tmpval & 0xff)
		block[i*SIZEOF_UINT32+1] = byte((tmpval >> 8) & 0xff)
		block[i*SIZEOF_UINT32+2] = byte((tmpval >> 16) & 0xff)
		block[i*SIZEOF_UINT32+3] = byte((tmpval >> 24) & 0xff)
	}
	return
}

func GetDecryptBlock(block []byte, key uint32) (decrypted []uint32) {
	const SIZEOF_UINT32 = 4
	var seed uint32 = 0xEEEEEEEE
	decrypted = make([]uint32, len(block)/SIZEOF_UINT32)

	for i := range decrypted {
		decrypted[i] = binary.LittleEndian.Uint32(block[i*SIZEOF_UINT32 : (i+1)*SIZEOF_UINT32])
	}

	for i := 0; i < len(decrypted); i++ {
		seed += StormBuffer[MPQ_HASH_KEY2_MIX+(key&0xFF)]
		decrypted[i] = decrypted[i] ^ (seed + key)
		value32 := decrypted[i]
		key = (((key ^ 0xFFFFFFFF) << 0x15) + 0x11111111) | (key >> 0x0B)
		seed = value32 + seed + (seed << 5) + 3
	}
	return decrypted

}

// Tested with the following:
//#define MPQ_KEY_HASH_TABLE          0xC3AF3770  // Obtained by HashString("(hash table)", MPQ_HASH_FILE_KEY)
//#define MPQ_KEY_BLOCK_TABLE         0xEC83B3A3  // Obtained by HashString("(block table)", MPQ_HASH_FILE_KEY)
func HashString(FileName string, HashType uint32) uint32 {
	var seed1 uint32 = 0x7FED7FED
	var seed2 uint32 = 0xEEEEEEEE

	for _, v := range FileName {
		// Convert the input character to uppercase
		// Convert slash (0x2F) to backslash (0x5C)
		var ch = AsciiToUpperTable[int(v)]

		seed1 = StormBuffer[HashType+uint32(ch)] ^ (seed1 + seed2)
		seed2 = uint32(ch) + seed1 + seed2 + (seed2 << 5) + 3
	}
	return seed1
}

func createFile(fileName string) (file *os.File) {
	// Make fileName in to a path that works on the current OS
	fileName = strings.Replace(".\\"+fileName, "\\", string(os.PathSeparator), -1)
	path := filepath.Join(outDir, fileName)
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}

	file, err = os.Create(path)
	if err != nil {
		log.Fatalln(err)
	}
	return file
}
