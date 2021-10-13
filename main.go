package main

import (
	"fmt"
	"math/big"
	"net"
	"os"
	"syscall"
	"unsafe"
)

const SIZE = 200000  // for simplicity pseudo-randomly set. This value should be optimized
const TMP_FILE_PATH = "/tmp/foo.dat"

type Tree struct {
	cdns []net.IP
	tree [SIZE]uint8
}

func (t*Tree) appendCdnIfNew(cdn *net.IP) uint8{
	for i, x := range t.cdns {
		if cdn.Equal(x) {
			return uint8(i)
		}
	}
	t.cdns = append(t.cdns, *cdn)
	return uint8(len(t.cdns) - 1)
}

func (t*Tree) leftChild(idx int) (int, uint8, uint8){
	nextIdx := (idx*2) + 2
	return nextIdx, t.tree[nextIdx], t.tree[nextIdx+1]
}

func (t*Tree) rightChild(idx int) (int, uint8, uint8){
	nextIdx := (idx*2) + 4
	return nextIdx, t.tree[nextIdx], t.tree[nextIdx+1]
}

func (t*Tree) createNode(idx int, numOfBits uint8, cdnIdx uint8) {
	t.tree[idx] = numOfBits
	t.tree[idx+1] = cdnIdx
}

func (t *Tree) Insert(newNet *net.IPNet, cdn *net.IP){
	cdnIdx := t.appendCdnIfNew(cdn) // This value shall be stored in new node as cdn server index

	ones, bits := newNet.Mask.Size()
	ipv6int := big.NewInt(0).SetBytes(newNet.IP.To16())

	offset, idx := 1, 0
	for {
		if offset > ones {
			panic(fmt.Sprintf("Error we did not find place for subnet %s\n", newNet))
		}

		curBit := ipv6int.Bit(bits-offset)
		var numOfNodeBits uint8 = 0
		if curBit == 0 {
			idx, numOfNodeBits, _ = t.leftChild(idx)
		} else {
			idx, numOfNodeBits, _ = t.rightChild(idx)
		}

		// how many same bits there are
		var sequenceOfBits uint8 = 0
		for curBit == ipv6int.Bit(bits-offset) && offset <= ones{
			offset += 1
			sequenceOfBits += 1
		}

		// Current node either does not exist or represents fewer same bits than we are dealing with
		if sequenceOfBits > numOfNodeBits {

			// This node does not exist yet, let's create it
			if numOfNodeBits == 0 {

				// We end here, whole prefix was traversed
				if offset > ones {
					t.createNode(idx, sequenceOfBits, cdnIdx)
					return

				// We create new node but there is still more prefix bits to go through
				} else {
					t.createNode(idx, sequenceOfBits, 0)
				}
			// Let's move back offset a little because current node does not consume all the same bits
			} else {
				offset -= int(sequenceOfBits - numOfNodeBits)
				continue
			}

		// Current node consumes exactly the same number of same bit we deal with in current part of prefix
		} else if sequenceOfBits == numOfNodeBits {
			// TODO add check: if we are at the end of prefix than we found duplicates in our routing data
			continue

		} else if sequenceOfBits < numOfNodeBits {
			// TODO
			// Current node should be split into 2, because we need to branch here cuz prefix bits change here
		}

	}

}

func (t *Tree) saveIntoFile(filepath string) {
	const sz = unsafe.Sizeof(*t)
	var asByteSlice = (*(*[1<<31 - 1]byte)(unsafe.Pointer(t)))[:sz]

	err := os.WriteFile(filepath, asByteSlice, 0644)
	checkErr(err)
	fmt.Printf("Saved tree as %d bytes into %s file\n", sz, filepath)
}

func checkErr(err error){
	if err != nil {
		panic(err)
	}
}

func loadTree(filepath string) *Tree {
	// Check size of file
	fstats, err := os.Stat(filepath)
	checkErr(err)

	// Open file
	f, err := os.OpenFile(filepath, os.O_RDWR, 0666)
	checkErr(err)

	// Map file into shared memory with mmap syscall
	mmap, err := syscall.Mmap(int(f.Fd()), 0, int(fstats.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
	checkErr(err)

	// Cast mapped memory into our tree struct
	root := (*Tree)(unsafe.Pointer(&mmap[0]))
	return root
}

func createAndSaveTreeScenario(){
	cdn1 := net.ParseIP("20.14.15.17") // example of cdn server
	_, ipv6subnet1, _ := net.ParseCIDR("2600:1700:1920:0000::/48") // example of routed subnet

	root := Tree{}
	root.Insert(ipv6subnet1, &cdn1)

	_, ipv6subnet2, _ := net.ParseCIDR("f600:1700:1920:0000::/48") // example of routed subnet
	root.Insert(ipv6subnet2, &cdn1)

	root.saveIntoFile(TMP_FILE_PATH)
}

func loadTreeScenario(){
	root := loadTree(TMP_FILE_PATH)
	fmt.Println(len(root.tree))
}

func main() {
	createAndSaveTreeScenario()
	//loadTreeScenario()
}
