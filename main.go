package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"runtime/pprof"
)

func init() {
	go drawCounter()
}

const (
	numNodes  = 0x1000000
	nodesFile = "nodes.gob"
	loopsFile = "loops.gob"
)

type Node struct {
	Num  int
	Dst  int
	Srcs []int
}

func (n Node) String() string {
	return fmt.Sprintf("{%06X -> %06X (%d srcs)}", n.Num, n.Dst, len(n.Srcs))
}

type Nodes []Node

func (n Nodes) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("[")
	if len(n) > 0 {
		buf.WriteString("\n")
	}
	for i := range n {
		buf.WriteString(fmt.Sprintf("\t%v\n", n[i]))
	}
	buf.WriteString("]")
	return buf.String()
}

var charToDec = map[rune]int{
	'0': 0,
	'1': 1,
	'2': 2,
	'3': 3,
	'4': 4,
	'5': 5,
	'6': 6,
	'7': 7,
	'8': 8,
	'9': 9,
	'A': 10,
	'B': 11,
	'C': 12,
	'D': 13,
	'E': 14,
	'F': 15,
}

func happify(i int) int {
	s := fmt.Sprintf("%X", i)
	dst := 0
	for _, r := range s {
		ri := charToDec[r]
		dst += ri * ri
	}
	return dst
}

func happifyColor(i int) int {
	r := happify(i & 0xFF0000)
	g := happify(i & 0x00FF00)
	b := happify(i & 0x0000FF)
	return ((r & 0xFF) << 16) | ((g & 0xFF) << 8) | (b & 0xFF)
}

func countSrcs(n Nodes, nn Node) int {
	c := 1
	for _, si := range nn.Srcs {
		c += countSrcs(n, n[si])
	}
	return c
}

func isInSet(n Nodes, i int) bool {
	for _, nn := range n {
		if nn.Num == i {
			return true
		}
	}
	return false
}

func createNodes() Nodes {
	n := make(Nodes, numNodes)
	for i := range n {
		dst := happifyColor(i)
		n[i].Num = i
		n[i].Dst = dst
		n[dst].Srcs = append(n[dst].Srcs, i)
	}
	return n
}

func store(n interface{}, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	return enc.Encode(n)
}

func load(n interface{}, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	return dec.Decode(n)
}

func findLoops(n Nodes) []Nodes {
	var loops []Nodes
	loop := make(Nodes, 0, 16)
outerLoop:
	for i := 0; i < numNodes; i++ {
		// If i is part of any of the loops found so far, don't bother
		for i := range loops {
			if isInSet(loop, i) {
				continue outerLoop
			}
		}

		if rloop := maybeLoop(n, i, loop); len(rloop) > 0 {
			loops = append(loops, rloop)
			loop = make(Nodes, 0, 16)
		}
	}
	return loops
}

func maybeLoop(n Nodes, i int, loop Nodes) Nodes {
	origI := i
	for {
		loop = append(loop, n[i])

		dst := n[i].Dst
		if dst == origI {
			break
		}

		for _, ln := range loop {
			if ln.Num == dst {
				return nil
			}
		}

		i = dst
	}

	return loop
}

func dedupLoops(loops []Nodes) []Nodes {
	found := map[int]bool{}
	ret := make([]Nodes, 0, len(loops))
outer:
	for _, loop := range loops {
		for _, n := range loop {
			if found[n.Num] {
				continue outer
			}
			found[n.Num] = true
		}
		ret = append(ret, loop)
	}
	return ret
}

func nodeLevels(n Nodes, nn Node, excluding Nodes) int {
	max := 0
outerLoop:
	for _, sni := range nn.Srcs {
		for _, en := range excluding {
			if en.Num == sni {
				continue outerLoop
			}
		}
		if c := nodeLevels(n, n[sni], nil); c > max {
			max = c
		}
	}

	// Return +1 to include this level
	return max + 1
}

func loopLevels(n Nodes, loop Nodes) int {
	max := 0
	for _, ln := range loop {
		if c := nodeLevels(n, ln, loop); c > max {
			max = c
		}
	}
	return max
}

func totalLevels(n Nodes, loops []Nodes) int {
	levels := 0
	for _, loop := range loops {
		levels += loopLevels(n, loop)
	}
	return levels
}

var drawCountCh = make(chan bool)

// this is started in its own go-routine in init
func drawCounter() {
	total := 0
	for _ = range drawCountCh {
		total++
		if total%0x10000 == 0 {
			log.Printf("drawn: %06X", total)
		}
	}
}

func drawNode(n Nodes, i img, nn Node, excluding Nodes, level int, start, end float64) {
	c := curve{
		level: level,
		color: nn.Num,
		start: start,
		end:   end,
	}
	i.drawCurve(c)
	drawCountCh <- true

	srcCounts := make([]int, len(nn.Srcs))
	srcTotal := 0
	for j, sni := range nn.Srcs {
		if isInSet(excluding, sni) {
			continue
		}
		c := countSrcs(n, n[sni])
		srcCounts[j] = c
		srcTotal += c
	}

	diff := end - start
	for j, sni := range nn.Srcs {
		sn := n[sni]

		if isInSet(excluding, sni) {
			continue
		}

		fract := (float64(srcCounts[j]) / float64(srcTotal)) * diff

		drawNode(n, i, sn, nil, level+1, start, start+fract)
		start += fract
	}
}

type drawNodeCmd struct {
	n          Nodes
	i          img
	nn         Node
	excluding  Nodes
	level      int
	start, end float64
	done       chan struct{}
}

func drawLoop(n Nodes, i img, loop Nodes, level int) {
	// We do this this way instead of just doing a countSrcs on each loop node
	// directly because we don't want to actually include the count from one of
	// the loop nodes
	srcTotal := 0
	srcCounts := make([]int, len(loop))
	for j, ln := range loop {
		for _, sni := range ln.Srcs {
			if isInSet(loop, sni) {
				continue
			}
			c := countSrcs(n, n[sni])
			srcCounts[j] += c
			srcTotal += c
		}
	}

	start := float64(0)
	for j, ln := range loop {
		fract := float64(srcCounts[j]) / float64(srcTotal)
		if math.IsNaN(fract) {
			fract = 1
		}
		drawNode(n, i, ln, loop, level, start, start+fract)
		start += fract
	}
}

func profileCPU() {
	cpuf, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(cpuf)
	ctrlcCh := make(chan os.Signal, 1)
	signal.Notify(ctrlcCh, os.Interrupt)
	go func() {
		<-ctrlcCh
		log.Printf("got ctrlc, writing out cpu profile and exiting")
		pprof.StopCPUProfile()
		cpuf.Close()
		os.Exit(0)
	}()

}

func main() {
	//j := newImg("test.png", 1000, 1000, 6)
	//j.drawCurve(curve{
	//	level: 5,
	//	color: 0xFF00ff,
	//	start: 0, end: 1,
	//})
	//j.drawCurve(curve{
	//	level: 4,
	//	color: 0xFF0000,
	//	start: 0, end: 0.5,
	//})
	//j.drawCurve(curve{
	//	level: 3,
	//	color: 0x0000FF,
	//	start: 0.5, end: 1,
	//})
	//j.drawCurve(curve{
	//	level: 2,
	//	color: 0x00FF00,
	//	start: 0.25, end: 0.75,
	//})
	//j.drawCurve(curve{
	//	level: 1,
	//	color: 0xFFFF00,
	//	start: 0, end: 0.66,
	//})
	//j.save()

	//return

	//log.Print("creating nodes")
	//nodes := createNodes()
	//log.Printf("total nodes: %X", len(nodes))

	//log.Print("storing nodes")
	//if err := store(&nodes, nodesFile); err != nil {
	//	log.Fatal(err)
	//}

	log.Print("loading in nodes")
	var nodes Nodes
	if err := load(&nodes, nodesFile); err != nil {
		log.Fatal(err)
	}

	//log.Print("finding loops")
	//loops := findLoops(nodes)
	//log.Printf("total loops (pr-dedup): %d", len(loops))

	//log.Printf("deduplicating loops")
	//loops = dedupLoops(loops)

	//log.Printf("storing loops")
	//if err := store(&loops, loopsFile); err != nil {
	//	log.Fatal(err)
	//}

	log.Print("loading in loops")
	var loops []Nodes
	if err := load(&loops, loopsFile); err != nil {
		log.Fatal(err)
	}

	log.Printf("loops: %v", loops)
	log.Printf("total loops: %d", len(loops))

	levels := totalLevels(nodes, loops) + 1 // plus 1 because we start on level 1
	log.Printf("totalLevels: %d", levels)

	profileCPU()

	i := newImg("happy-tree.png", 5000, 5000, levels)
	level := 1
	for _, loop := range loops {
		drawLoop(nodes, i, loop, level)
		level += loopLevels(nodes, loop)
		level++
	}

	if err := i.save(); err != nil {
		log.Fatal(err)
	}
}
