package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------- data types

// Process represents one pirate crew waiting for the CPU.
type Process struct {
	ID         string
	Arrival    int
	Burst      int
	Remaining  int
	Start      int
	Completion int
	Waiting    int
	Turnaround int
	started    bool
}

// Block is one continuous stretch of the timeline: a crew running, or idle sea.
type Block struct {
	Label string
	Start int
	End   int
}

const idleLabel = "IDLE"

// ------------------------------------------------------------ shared helpers

// clone gives each algorithm a private copy so runs never corrupt each other.
func clone(src []Process) []Process {
	out := make([]Process, len(src))
	copy(out, src)
	for i := range out {
		out[i].Remaining = out[i].Burst
		out[i].started = false
		out[i].Start = 0
		out[i].Completion = 0
		out[i].Waiting = 0
		out[i].Turnaround = 0
	}
	return out
}

// sortByArrival orders crews by arrival time, keeping entry order on ties.
func sortByArrival(procs []Process) {
	sort.SliceStable(procs, func(a, b int) bool {
		return procs[a].Arrival < procs[b].Arrival
	})
}

// addBlock records a stretch of the timeline, merging with the previous block
// when the same label runs back to back.
func addBlock(timeline []Block, label string, start, end int) []Block {
	if end <= start {
		return timeline
	}
	if n := len(timeline); n > 0 && timeline[n-1].Label == label && timeline[n-1].End == start {
		timeline[n-1].End = end
		return timeline
	}
	return append(timeline, Block{Label: label, Start: start, End: end})
}

// finalize derives the metrics once completion times are known.
func finalize(procs []Process) {
	for i := range procs {
		p := &procs[i]
		p.Turnaround = p.Completion - p.Arrival
		p.Waiting = p.Turnaround - p.Burst
	}
}

// ------------------------------------------------------------- gantt drawing

func center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	left := (width - len(s)) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-len(s)-left)
}

// drawGantt renders the timeline as an ASCII bar with time markers underneath.
func drawGantt(timeline []Block) string {
	if len(timeline) == 0 {
		return "(nothing was scheduled)"
	}

	// Each cell must fit its label and the boundary number below it.
	widths := make([]int, len(timeline))
	for i, b := range timeline {
		w := len(b.Label) + 4
		if need := len(strconv.Itoa(b.End)) + 2; w < need {
			w = need
		}
		widths[i] = w
	}

	// Column of the '+' separator preceding each cell.
	cols := make([]int, len(timeline)+1)
	for i, w := range widths {
		cols[i+1] = cols[i] + w + 1
	}

	var border, body strings.Builder
	border.WriteString("+")
	body.WriteString("|")
	for i, b := range timeline {
		border.WriteString(strings.Repeat("-", widths[i]) + "+")
		body.WriteString(center(b.Label, widths[i]) + "|")
	}

	// Stamp each boundary number at its exact column.
	last := strconv.Itoa(timeline[len(timeline)-1].End)
	buf := []byte(strings.Repeat(" ", cols[len(cols)-1]+len(last)+1))
	for i, b := range timeline {
		copy(buf[cols[i]:], strconv.Itoa(b.Start))
	}
	copy(buf[cols[len(cols)-1]:], last)

	return border.String() + "\n" + body.String() + "\n" +
		border.String() + "\n" + strings.TrimRight(string(buf), " ")
}

// ------------------------------------------------------------------- results

func averages(procs []Process) (float64, float64) {
	totalWait, totalTat := 0, 0
	for _, p := range procs {
		totalWait += p.Waiting
		totalTat += p.Turnaround
	}
	n := float64(len(procs))
	return float64(totalWait) / n, float64(totalTat) / n
}

func printResults(name string, procs []Process, timeline []Block) {
	fmt.Printf("\n=== %s ===\n\n", name)

	fmt.Println("Gantt Chart:")
	fmt.Println(drawGantt(timeline))

	order := make([]string, 0, len(timeline))
	for _, b := range timeline {
		order = append(order, fmt.Sprintf("%s(%d-%d)", b.Label, b.Start, b.End))
	}
	fmt.Println("\nExecution order:")
	fmt.Println("  " + strings.Join(order, " -> "))

	fmt.Printf("\n%-8s %8s %8s %10s %10s %12s\n",
		"Crew", "Arrival", "Burst", "Complete", "Waiting", "Turnaround")
	fmt.Println(strings.Repeat("-", 60))
	for _, p := range procs {
		fmt.Printf("%-8s %8d %8d %10d %10d %12d\n",
			p.ID, p.Arrival, p.Burst, p.Completion, p.Waiting, p.Turnaround)
	}
	fmt.Println(strings.Repeat("-", 60))

	avgWait, avgTat := averages(procs)
	fmt.Printf("Average Waiting Time    : %.2f\n", avgWait)
	fmt.Printf("Average Turnaround Time : %.2f\n", avgTat)
}

// ---------------------------------------------------------------- algorithms

// FCFS runs crews strictly in the order they arrive.
func FCFS(input []Process) ([]Process, []Block) {
	procs := clone(input)
	sortByArrival(procs)

	var timeline []Block
	clock := 0

	for i := range procs {
		p := &procs[i]

		if clock < p.Arrival {
			timeline = addBlock(timeline, idleLabel, clock, p.Arrival)
			clock = p.Arrival
		}

		p.Start = clock
		clock += p.Burst
		p.Completion = clock
		timeline = addBlock(timeline, p.ID, p.Start, clock)
	}

	finalize(procs)
	return procs, timeline
}

// SJF picks the shortest job among those that have already arrived,
// then runs it to completion (non-preemptive).
func SJF(input []Process) ([]Process, []Block) {
	procs := clone(input)
	done := make([]bool, len(procs))

	var timeline []Block
	clock, completed := 0, 0

	for completed < len(procs) {
		best := -1
		for i := range procs {
			if done[i] || procs[i].Arrival > clock {
				continue
			}
			if best == -1 ||
				procs[i].Burst < procs[best].Burst ||
				(procs[i].Burst == procs[best].Burst && procs[i].Arrival < procs[best].Arrival) {
				best = i
			}
		}

		// Nobody has arrived yet: drift forward to the next arrival.
		if best == -1 {
			next := -1
			for i := range procs {
				if !done[i] && (next == -1 || procs[i].Arrival < procs[next].Arrival) {
					next = i
				}
			}
			timeline = addBlock(timeline, idleLabel, clock, procs[next].Arrival)
			clock = procs[next].Arrival
			continue
		}

		p := &procs[best]
		p.Start = clock
		clock += p.Burst
		p.Completion = clock
		timeline = addBlock(timeline, p.ID, p.Start, clock)

		done[best] = true
		completed++
	}

	finalize(procs)
	return procs, timeline
}

// RoundRobin gives every crew a fixed slice of CPU time in circular order.
func RoundRobin(input []Process, quantum int) ([]Process, []Block) {
	procs := clone(input)
	sortByArrival(procs)

	var timeline []Block
	var queue []int
	clock, next, completed := 0, 0, 0

	for completed < len(procs) {
		// Admit everyone who has arrived by now.
		for next < len(procs) && procs[next].Arrival <= clock {
			queue = append(queue, next)
			next++
		}

		// Nobody is waiting: jump the clock to the next arrival.
		if len(queue) == 0 {
			timeline = addBlock(timeline, idleLabel, clock, procs[next].Arrival)
			clock = procs[next].Arrival
			continue
		}

		// Take the crew at the front of the queue.
		i := queue[0]
		queue = queue[1:]
		p := &procs[i]

		if !p.started {
			p.Start = clock
			p.started = true
		}

		// Run for the quantum, or less if that is all the work left.
		run := quantum
		if p.Remaining < run {
			run = p.Remaining
		}
		timeline = addBlock(timeline, p.ID, clock, clock+run)
		clock += run
		p.Remaining -= run

		// Crews that arrived DURING this slice board before the
		// preempted crew rejoins the back of the queue.
		for next < len(procs) && procs[next].Arrival <= clock {
			queue = append(queue, next)
			next++
		}

		if p.Remaining > 0 {
			queue = append(queue, i)
		} else {
			p.Completion = clock
			completed++
		}
	}

	finalize(procs)
	return procs, timeline
}

// --------------------------------------------------------------- user input

var reader = bufio.NewReader(os.Stdin)

// prompt prints a label and returns the trimmed line the user typed.
func prompt(label string) string {
	fmt.Print(label)
	line, err := reader.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		fmt.Println("\nInput closed. Setting sail.")
		os.Exit(0)
	}
	return strings.TrimSpace(line)
}

// readInt keeps asking until it gets a valid integer of at least min.
func readInt(label string, min int) int {
	for {
		v, err := strconv.Atoi(prompt(label))
		if err != nil {
			fmt.Println("  -> that is not a number, try again.")
			continue
		}
		if v < min {
			fmt.Printf("  -> value must be at least %d.\n", min)
			continue
		}
		return v
	}
}

// readProcesses collects the crew manifest from the user.
func readProcesses() []Process {
	n := readInt("How many pirate crews (processes)? ", 1)
	procs := make([]Process, 0, n)
	seen := make(map[string]bool)

	for i := 0; i < n; i++ {
		fmt.Printf("\n-- Crew %d --\n", i+1)

		var id string
		for {
			id = prompt("  Process ID (blank = auto): ")
			if id == "" {
				id = fmt.Sprintf("P%d", i+1)
			}
			if seen[id] {
				fmt.Println("  -> that ID is already taken.")
				continue
			}
			break
		}
		seen[id] = true

		arrival := readInt("  Arrival Time: ", 0)
		burst := readInt("  Burst Time: ", 1)

		procs = append(procs, Process{ID: id, Arrival: arrival, Burst: burst, Remaining: burst})
	}
	return procs
}

func showManifest(procs []Process) {
	fmt.Println("\nCrew manifest:")
	fmt.Printf("%-8s %8s %8s\n", "Crew", "Arrival", "Burst")
	fmt.Println(strings.Repeat("-", 26))
	for _, p := range procs {
		fmt.Printf("%-8s %8d %8d\n", p.ID, p.Arrival, p.Burst)
	}
}

// sampleData is kept for quick testing without typing a manifest each run.
func sampleData() []Process {
	return []Process{
		{ID: "P1", Arrival: 0, Burst: 5, Remaining: 5},
		{ID: "P2", Arrival: 1, Burst: 3, Remaining: 3},
		{ID: "P3", Arrival: 2, Burst: 8, Remaining: 8},
		{ID: "P4", Arrival: 3, Burst: 6, Remaining: 6},
	}
}

// --------------------------------------------------------------------- main

func main() {
	fmt.Println("=========================================")
	fmt.Println("     THE PIRATE KING'S SCHEDULER")
	fmt.Println("  CPU Scheduling Simulator (Grand Line)")
	fmt.Println("=========================================")

	procs := readProcesses()
	showManifest(procs)

	for {
		fmt.Println("\nChoose a scheduling strategy:")
		fmt.Println("  1) First Come First Serve (FCFS)")
		fmt.Println("  2) Shortest Job First (SJF, non-preemptive)")
		fmt.Println("  3) Round Robin (RR)")
		fmt.Println("  4) Run all three and compare")
		fmt.Println("  5) Re-enter crew details")
		fmt.Println("  6) Exit")

		switch prompt("Selection: ") {
		case "1":
			r, t := FCFS(procs)
			printResults("First Come First Serve (FCFS)", r, t)

		case "2":
			r, t := SJF(procs)
			printResults("Shortest Job First (SJF, Non-Preemptive)", r, t)

		case "3":
			q := readInt("Time Quantum: ", 1)
			r, t := RoundRobin(procs, q)
			printResults(fmt.Sprintf("Round Robin (RR), Quantum = %d", q), r, t)

		case "4":
			q := readInt("Time Quantum (for Round Robin): ", 1)

			r1, t1 := FCFS(procs)
			printResults("First Come First Serve (FCFS)", r1, t1)
			r2, t2 := SJF(procs)
			printResults("Shortest Job First (SJF, Non-Preemptive)", r2, t2)
			r3, t3 := RoundRobin(procs, q)
			printResults(fmt.Sprintf("Round Robin (RR), Quantum = %d", q), r3, t3)

			fmt.Println("\n=== Comparison ===")
			fmt.Printf("%-42s %13s %16s\n", "Algorithm", "Avg Waiting", "Avg Turnaround")
			fmt.Println(strings.Repeat("-", 73))
			names := []string{"First Come First Serve (FCFS)",
				"Shortest Job First (SJF, Non-Preemptive)",
				fmt.Sprintf("Round Robin (RR), Quantum = %d", q)}
			for i, set := range [][]Process{r1, r2, r3} {
				w, t := averages(set)
				fmt.Printf("%-42s %13.2f %16.2f\n", names[i], w, t)
			}

		case "5":
			procs = readProcesses()
			showManifest(procs)

		case "6":
			fmt.Println("\nFair winds. The One Piece is out there somewhere.")
			return

		default:
			fmt.Println("  -> pick a number from 1 to 6.")
		}
	}
}
