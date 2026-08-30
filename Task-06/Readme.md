# Task 06 — The Pirate King's Scheduler

A CPU scheduling simulator in Go. Every pirate crew arriving on the Grand Line is a process
waiting for the CPU, and the program decides who sails first under three classic scheduling
policies:

- First Come First Serve (FCFS)
- Shortest Job First (SJF), non-preemptive
- Round Robin (RR), with a time quantum you choose

Runs entirely in the terminal. Standard library only, no external packages.

## Running it

```bash
cd pirate-scheduler
go run .
```

The program asks how many crews you have, then for each one: a Process ID (leave it blank
and it names them P1, P2, ... for you), an Arrival Time and a Burst Time. After that you
pick an algorithm from the menu. Round Robin asks for the quantum separately, since the
other two don't need it.

Option 4 runs all three on the same crew list and prints a comparison table at the end,
which is the most interesting thing to look at.

## What it prints

For each algorithm: a Gantt chart, the execution order, a per-process table, and the two
averages.

```
Gantt Chart:
+------+------+------+------+
|  P1  |  P2  |  P3  |  P4  |
+------+------+------+------+
0      5      8      16     22

Crew      Arrival    Burst   Complete    Waiting   Turnaround
------------------------------------------------------------
P1              0        5          5          0            5
P2              1        3          8          4            7
P3              2        8         16          6           14
P4              3        6         22         13           19
------------------------------------------------------------
Average Waiting Time    : 5.75
Average Turnaround Time : 11.25
```

Running the same four crews through all three, with a quantum of 2:

| Algorithm | Avg Waiting | Avg Turnaround |
|---|---|---|
| FCFS | 5.75 | 11.25 |
| SJF | 5.25 | 10.75 |
| Round Robin (q=2) | 9.75 | 15.25 |

Round Robin comes out worst on both averages, which surprised me at first. It isn't a bug.
RR is optimising for something the averages don't show: under FCFS, P4 sits untouched for
13 time units, while under RR every crew has had CPU time by t=8. You trade average
finishing time for responsiveness. That's why interactive systems use it.

## How it works

### Files

Everything is in `main.go`, split into sections: data types, shared helpers, the Gantt
renderer, the three algorithms, and the input handling.

### The core idea

I only track two things during a simulation: when each process finishes, and a list of
blocks saying who occupied the CPU between which times. Everything else is arithmetic
afterwards:

```
Turnaround = Completion - Arrival
Waiting    = Turnaround - Burst
```

I tried accumulating waiting time as the clock advanced at first and kept getting it wrong
at the boundaries. Deriving it at the end from two subtractions made the whole thing
simpler, and it meant all three algorithms could share one `finalize()` function instead of
each having its own metric logic.

The same goes for the timeline. Because every algorithm produces the same `[]Block` shape,
the Gantt renderer and the results table are written once and reused three times.

### FCFS

Sort by arrival, run each crew to completion. If the clock reaches a point where nobody has
arrived, record an `IDLE` block and jump the clock forward to the next arrival.

### SJF

Can't precompute an order here, because the choice depends on what has arrived. So the loop
runs until everyone is done, and each pass scans for the smallest burst among crews that
have already arrived and aren't finished. Ties go to the earlier arrival so the output is
the same every run.

The `arrival > clock` check is the whole algorithm. It's shortest job among those *waiting*,
not shortest overall — a tiny job arriving at t=50 doesn't get to jump the queue at t=10.

### Round Robin

A FIFO queue of process indices, a quantum, and a `Remaining` field counting down. The crew
at the front runs for `min(quantum, remaining)` — the quantum is a ceiling, not a fixed
amount, and forgetting that inflates every completion time after the first short slice.

The part that cost me the most time: crews arriving *during* a time slice have to be
enqueued **before** the preempted crew goes back. With the sample data and q=2, the correct
order starts P1, P2, P3, P1. If you re-queue the preempted crew first you get P1, P1, P2, P3
and every number afterwards is wrong — but it still looks plausible, which is what made it
hard to spot. That's why the "admit arrivals" loop appears twice in the function.

### The Gantt chart

Aligning it was fiddlier than expected, because labels vary in width (`P1` vs `IDLE`) and so
do the boundary numbers (`8` vs `22`). Building the number line by concatenation drifted out
of alignment. What worked: compute a width per cell, record the exact column of every `+`
separator, then allocate a buffer of spaces and stamp each number into its known column.
Alignment is correct by construction that way.

### Edge cases handled

- Idle CPU when there are gaps between arrivals
- All crews arriving at t=0
- Quantum larger than every burst — RR collapses into FCFS, which is a good sanity check
- Quantum of 1
- A single crew
- Non-numeric input, negative values, duplicate process IDs

## Resources used

- A Tour of Go — https://go.dev/tour/list — for slices, structs and the basics
- Effective Go — https://go.dev/doc/effective_go
- Go package docs for `sort`, `strings`, `bufio`, `strconv`
- Silberschatz, Galvin & Gagne, *Operating System Concepts*, Chapter 5 on CPU scheduling
- GeeksforGeeks CPU scheduling articles, mainly to check the Round Robin queue ordering
  convention and to verify my numbers against worked examples

## What I learned

**Go**

- Slices are views over a shared array, not copies. Passing one into a function and changing
  an element changes the caller's data too. I found this out when running all three
  algorithms in sequence gave wrong numbers for the second and third — Round Robin had
  already decremented `Remaining` to zero for everyone. Hence the `clone()` function.
- `for _, p := range procs` hands you a copy of each element, so writing to `p` does nothing.
  You need `p := &procs[i]` or to index directly. This one is silent — no compiler error,
  the writes just vanish.
- `sort.SliceStable` vs `sort.Slice`. With plain `sort.Slice`, two processes with the same
  arrival time could come out in either order, so FCFS gave different charts on different
  runs. Stable sorting fixed it.
- A queue out of a slice: `append` to add, `queue[0]` to peek, `queue = queue[1:]` to pop.
- Go handles errors as return values rather than exceptions, so `strconv.Atoi` returns both
  the number and an error you check with `if err != nil`.

**Scheduling**

- The convoy effect is very visible in the numbers. P4 waits 13 units under FCFS purely
  because it queued behind an 8-unit job. SJF reorders around that and drops the average.
- Any "loop until everything is done" needs a guard for the case where nothing can progress.
  Both SJF and RR hang forever without the branch that jumps the clock to the next arrival
  when nobody has arrived yet.
- Idle CPU time is part of the schedule, not the absence of one. Once I modelled it as a
  real block on the timeline, the sparse-arrival cases stopped being special.
- The three algorithms aren't ranked. SJF wins on averages, RR wins on responsiveness, FCFS
  wins on being predictable and starvation-free. Which one is "best" depends entirely on
  what the system is for.