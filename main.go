package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU() + 1)
	gob.Register(Node{})
	go drawCounter()
	for i := 0; i < runtime.NumCPU(); i++ {
		go drawNodeSpin()
	}
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

func intToDst(i int) int {
	s := fmt.Sprintf("%06X", i)
	dst := 0
	for _, r := range s {
		ri := charToDec[r]
		dst += ri * ri
	}
	return dst
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
		dst := intToDst(i)
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
		for _, loop := range loops {
			for _, ln := range loop {
				if ln.Num == i {
					continue outerLoop
				}
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
	if i%0x1000 == 0 {
		log.Printf("maybeLoop: %06X", i)
	}
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
		levels++
	}
	return levels
}

var drawCountCh = make(chan int)

// this is started in its own go-routine in init
func drawCounter() {
	maxLevel := 0
	levelm := map[int]int{}
	total := 0
	for level := range drawCountCh {
		if level > maxLevel {
			maxLevel = level
		}
		levelm[level]++
		total++
		if total%0x1000 == 0 {
			log.Printf("drawn: %06X", total)
		}
	}

	for i := 1; i <= maxLevel; i++ {
		log.Printf("level %d -> %d", i, levelm[i])
	}
}

func drawNode(cmd drawNodeCmd) {
	c := curve{
		level: cmd.level,
		color: cmd.nn.Num,
		start: cmd.start,
		end:   cmd.end,
	}
	cmd.i.drawCurve(c)
	drawCountCh <- cmd.level

	srcCounts := make([]int, len(cmd.nn.Srcs))
	srcTotal := 0
	for i, sni := range cmd.nn.Srcs {
		if isInSet(cmd.excluding, sni) {
			continue
		}
		c := countSrcs(cmd.n, cmd.n[sni])
		srcCounts[i] = c
		srcTotal += c
	}

	diff := cmd.end - cmd.start
	for i, sni := range cmd.nn.Srcs {
		sn := cmd.n[sni]

		if isInSet(cmd.excluding, sni) {
			continue
		}

		fract := (float64(srcCounts[i]) / float64(srcTotal)) * diff

		drawNode(drawNodeCmd{
			n:         cmd.n,
			i:         cmd.i,
			nn:        sn,
			excluding: nil,
			level:     cmd.level + 1,
			start:     cmd.start,
			end:       cmd.start + fract,
		})
		cmd.start += fract
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

var drawNodeCh = make(chan drawNodeCmd)

// multiple of these will be started by init
func drawNodeSpin() {
	for cmd := range drawNodeCh {
		log.Printf("drawNodeSpin got %v", cmd.nn)
		drawNode(cmd)
		close(cmd.done)
	}
}

func drawLoop(n Nodes, i img, loop Nodes, level int) []drawNodeCmd {
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

	ret := make([]drawNodeCmd, 0, len(loop))
	start := float64(0)
	for j, ln := range loop {
		fract := float64(srcCounts[j]) / float64(srcTotal)
		if math.IsNaN(fract) {
			fract = 1
		}
		cmd := drawNodeCmd{
			n:         n,
			i:         i.copyBlank(),
			nn:        ln,
			excluding: loop,
			level:     level,
			start:     start,
			end:       start + fract,
			done:      make(chan struct{}),
		}
		drawNodeCh <- cmd
		ret = append(ret, cmd)
		start += fract
	}
	return ret
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

	//j := newImg("test.png", 400, 400, 50)
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

	//log.Printf("storing loops")
	//if err := store(&loops, loopsFile); err != nil {
	//	log.Fatal(err)
	//}

	log.Print("loading in loops")
	var loops []Nodes
	if err := load(&loops, loopsFile); err != nil {
		log.Fatal(err)
	}

	levels := totalLevels(nodes, loops) + 1 // plus 1 because we start on level 1
	log.Printf("totalLevels: %d", levels)

	profileCPU()

	w, h := 1000, 1000
	imgName := fmt.Sprintf("happy-tree.png")

	i := newImg(imgName, w, h, levels)
	level := 1
	var promises []drawNodeCmd
	for _, loop := range loops {
		promises = append(promises, drawLoop(nodes, i, loop, level)...)
		level += loopLevels(nodes, loop)
		level++
	}

	for _, cmd := range promises {
		<-cmd.done
		i.cat(cmd.i)
	}
	close(drawCountCh)

	if err := i.save(); err != nil {
		log.Fatal(err)
	}

	// Wait a second for other threads to finish logging and stuff
	time.Sleep(1 * time.Second)
}
