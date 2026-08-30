# Captain's Logbook — Task 01

**Challenge:** https://github.com/rogueone-x/Terminal-Voyage-User-Edition
**Final repo:** https://github.com/rogueone-x/Laugh-Tale-Merge-War
**Completed:** 6/6 levels

---

## What I Recovered

| # | Item | Value |
|---|------|-------|
| 1 | Awakening Signature | `ONE_PIECE{GITO_GITO_NO_AWAKENING}` |
| 2 | Executive Transmission Code | `BAROQUE_DIAL{SPLIT_TIMELINE_MISDIRECTION}` |
| 3 | Poneglyph Fragment I | `KjY2MjF4bW0lKzYqNyBsIS0vbTAtJTcnL` |
| 4 | Poneglyph Fragment II | `SwnbzptDiM3JSpvFiMuJ28PJzAlJ28VIzA=` |
| 5 | Decoded Poneglyph | `https://github.com/rogueone-x/Laugh-Tale-Merge-War` |
| 6 | Pirate King's Password | `TheGrandLineRemembers` |
| 7 | Final Flag | `FLAG{The_Grand_Line_Remembers_Your_Commit}` |

---

## Level 1 — Loguetown Reef

Forty files with near-identical names spread across four sectors, and I
had to find the one real Devil Fruit among them.

My first instinct was to just try one. It came back with "It's just
another Marine replica" — so the script was actually checking the file,
not accepting anything I handed it. That meant the files differed in some
way I wasn't seeing.

Going back to the riddle, two phrases stood out: the replicas were
"permanently sealed," but the real fruit "still possesses the power to
awaken itself." That's a description of permissions. A sealed file is
read-only. A file that can act on its own is executable.

```
$ find . -name "devil_fruit_*.txt" -perm +111
./GrandLine/Loguetown_Reef/sector_C/devil_fruit_6.txt
```

One file out of forty.

```
$ ./eat.sh sector_C/devil_fruit_6.txt
You have awakened the legendary... Gito Gito no Mi
AWAKENING_SIGNATURE: ONE_PIECE{GITO_GITO_NO_AWAKENING}
```

![Level 1]

---

## Level 2 — Whiskey Peak

The riddle promised "another version of this island," so I ran `ls -a`
expecting a hidden dotfile. Nothing. Just a decoy manifest listing sake
and sea king meat.

What I'd overlooked was something I'd already seen. When I checked the
Git log earlier, the branch was called `canonical-timeline`, not `main`.
That's a strange name unless there are other timelines — and the whole
premise of this Devil Fruit is walking parallel histories. A branch *is*
a parallel history.

```
$ git branch -a
  canonical-timeline
  remotes/origin/alternate_timeline
  remotes/origin/little_garden
  remotes/origin/whiskey_peak_investigation
```

Switching to `whiskey_peak_investigation` made a hidden folder appear
that simply doesn't exist on the default branch. Two layers of
concealment at once.

The vault inside refused me until I exported the Level 1 flag as an
environment variable — which is what "recognizes the aura" turns out to
mean quite literally. It reads your shell, not your arguments.

```
$ export AWAKENING_SIGNATURE="ONE_PIECE{GITO_GITO_NO_AWAKENING}"
$ ./unlock_vault.sh
[SIGNATURE MATCH] Devil Fruit aura detected.
```

That gave me `BAROQUE_DIAL{SPLIT_TIMELINE_MISDIRECTION}`.

![Level 2]

---

## Level 3 — Mr. 3's Labyrinth

Little Garden turned out to be 490 files buried in 60 nested
directories, four sectors deep with identical structure. Opening them by
hand was obviously not the plan.

The riddle said transmission codes get converted into their "broadcast
representation," so I tried searching for the base64 of my Level 2 code.
Nothing came back. Rather than keep guessing encodings, I decided to look
at what the files actually contained — all of them at once:

```
$ find GrandLine/Wax_Jungle -type f -exec cat {} \; | sort | uniq -c | sort -rn
  64 SYSTEM_DUMP: DEN DEN MUSHI OFFLINE
  62 SYSTEM_DUMP: FALSE ALARM
  ...
   1 PONEGLYPH_FRAGMENT_I = "KjY2MjF4bW0lKzYqNyBsIS0vbTAtJTcnL"
   1 BAROQUE WORKS EXECUTIVE REPORT
```

Decoy lines appear dozens of times. The real report appears once. Sorting
by frequency found the outlier without needing to know the encoding at
all, which I thought was a neater solution than what I'd been attempting.

It also explained the failed search: the stored base64 ended `Tn0K`
where mine ended `Tn0=`. That trailing `K` is an encoded newline — I'd
stripped it and they hadn't. One byte, and an exact-match search finds
nothing while telling you nothing about why.

![Level 3]

---

## Level 4 — Water 7

A file with no extension at all. The riddle said to ask it for its
nature rather than its name, which is exactly what `file` does — it reads
the first few bytes rather than trusting the filename.

```
$ file puffing_tom_blueprints
gzip compressed data, was "step2_blueprints.tar"
```

"step2" implied more layers, and there were four in total. Each one was
the same loop: run `file`, see what it is, unwrap it with the matching
tool, run `file` again.

| Layer | What it was | How I opened it |
|---|---|---|
| 1 | gzip | `gunzip` |
| 2 | tar archive | `tar -xvf` |
| 3 | zip archive | `unzip` |
| 4 | plain text | `cat` |

At the bottom were two files. One was a decoy about keel blueprints. The
other had what I came for:

```
PONEGLYPH_FRAGMENT_II="SwnbzptDiM3JSpvFiMuJ28PJzAlJ28VIzA="
```

![Level 4]

---

## Level 5 — Rebuilding the Poneglyph

Fragment I had no base64 padding and Fragment II ended in `=`. That's
what one base64 string looks like when you cut it in half, so I joined
them and decoded:

```
*6621xmm%+6*7 l!-/m0-%7'-,'o:m#7%*o#.'o'0%'o#0%
```

Readable characters, but not words — meaning there was another cipher
underneath.

What gave it away was the shape. `*6621xmm` is eight characters, and so
is `https://`. If the first eight characters of a hidden URL had been
transformed, they'd line up. XORing `h` against `*` gave 66, and every
pair after it gave 66 too. A single constant key across the whole string.

```
$ python3 -c "
import base64
d = base64.b64decode('KjY2MjF4bW0lKzYqNyBsIS0vbTAtJTcnLSwnbzptDiM3JSpvFiMuJ28PJzAlJ28VIzA=')
for k in range(1, 256):
    r = bytes(b ^ k for b in d)
    if all(32 <= c < 127 for c in r):
        print(k, r.decode())
"
66 https://github.com/rogueone-x/Laugh-Tale-Merge-War
```

Key 66 is `0x42`, the letter B — for Baroque Works, presumably.

One thing I got wrong first: I filtered the brute-force output to lines
containing `ONE_PIECE` or `{`, assuming this would be another flag. It
was a URL, so my filter threw away the right answer and showed me only
garbage. Filtering by what I expected instead of by what was readable
cost me a round.

![Level 5](screenshots/level-05-xor.png)

---

## Level 6 — The Merge War

The final repo had two branches that split from a common ancestor, each
having edited the same two files:

```
* 8835d14 (ancient_history) Recovered ancient history
| * 091591f (pirate_king_path) Current pirate records
|/
* 34b8f9a Initial Laugh Tale records
```

Comparing them showed why neither timeline could win:

| File | ancient_history | pirate_king_path | Together |
|---|---|---|---|
| key_part_1 | `Line` | `TheGrand` | `TheGrandLine` |
| key_part_2 | `bers` | `Remem` | `Remembers` |

Neither side holds a complete word. That's the point the README was
making — history isn't preserved by picking a side. The usual instinct in
a merge conflict is to choose one version and discard the other, and here
that would destroy the answer either way.

```
$ git merge origin/pirate_king_path
CONFLICT (content): Merge conflict in treasure/key_part_1.txt
CONFLICT (content): Merge conflict in treasure/key_part_2.txt
```

![Conflict markers](screenshots/level-06-conflict.png)

I resolved both by joining the two sides into one line and clearing out
the conflict markers, committed the merge, and gave the vault the
password:

```
$ ./victory.sh
Enter the Pirate King's Password: TheGrandLineRemembers
Timeline Integrity ............. OK
FLAG{The_Grand_Line_Remembers_Your_Commit}
```

![Victory]

---