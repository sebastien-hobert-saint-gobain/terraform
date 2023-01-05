package list

import (
	"reflect"
)

type ProcessChild func(beforeIx, afterIx int)

func Process[T any](before, after []T, isObjType func(item T) bool, process ProcessChild) {
	lcs := lcs(before, after)
	var beforeIx, afterIx, lcsIx int
	for beforeIx < len(before) || afterIx < len(after) || lcsIx < len(lcs) {
		// Step through all the before values until we hit the next item in the
		// longest common subsequence. We are going to just say that all of
		// these have been deleted.
		for beforeIx < len(before) && (lcsIx >= len(lcs) || !reflect.DeepEqual(before[beforeIx], lcs[lcsIx])) {
			isObjectDiff := isObjType(before[beforeIx]) && afterIx < len(after) && isObjType(after[afterIx]) && (lcsIx >= len(lcs) || !reflect.DeepEqual(after[afterIx], lcs[lcsIx]))
			if isObjectDiff {
				process(beforeIx, afterIx)
				beforeIx++
				afterIx++
				continue
			}

			process(beforeIx, len(after))
			beforeIx++
		}

		// Now, step through all the after values until hit the next item in the
		// LCS. We are going to say that all of these have been created.
		for afterIx < len(after) && (lcsIx >= len(lcs) || !reflect.DeepEqual(after[afterIx], lcs[lcsIx])) {
			process(len(before), afterIx)
			afterIx++
		}

		// Finally, add the item in common as unchanged.
		if lcsIx < len(lcs) {
			process(beforeIx, afterIx)
			beforeIx++
			afterIx++
			lcsIx++
		}
	}
}

func lcs[T any](xs, ys []T) []T {
	if len(xs) == 0 || len(ys) == 0 {
		return make([]T, 0)
	}

	c := make([]int, len(xs)*len(ys))
	eqs := make([]bool, len(xs)*len(ys))
	w := len(xs)

	for y := 0; y < len(ys); y++ {
		for x := 0; x < len(xs); x++ {
			eq := false
			if reflect.DeepEqual(xs[x], ys[y]) {
				eq = true
				eqs[(w*y)+x] = true // equality tests can be expensive, so cache it
			}
			if eq {
				// Sequence gets one longer than for the cell at top left,
				// since we'd append a new item to the sequence here.
				if x == 0 || y == 0 {
					c[(w*y)+x] = 1
				} else {
					c[(w*y)+x] = c[(w*(y-1))+(x-1)] + 1
				}
			} else {
				// We follow the longest of the sequence above and the sequence
				// to the left of us in the matrix.
				l := 0
				u := 0
				if x > 0 {
					l = c[(w*y)+(x-1)]
				}
				if y > 0 {
					u = c[(w*(y-1))+x]
				}
				if l > u {
					c[(w*y)+x] = l
				} else {
					c[(w*y)+x] = u
				}
			}
		}
	}

	// The bottom right cell tells us how long our longest sequence will be
	seq := make([]T, c[len(c)-1])

	// Now we will walk back from the bottom right cell, finding again all
	// of the equal pairs to construct our sequence.
	x := len(xs) - 1
	y := len(ys) - 1
	i := len(seq) - 1

	for x > -1 && y > -1 {
		if eqs[(w*y)+x] {
			// Add the value to our result list and then walk diagonally
			// up and to the left.
			seq[i] = xs[x]
			x--
			y--
			i--
		} else {
			// Take the path with the greatest sequence length in the matrix.
			l := 0
			u := 0
			if x > 0 {
				l = c[(w*y)+(x-1)]
			}
			if y > 0 {
				u = c[(w*(y-1))+x]
			}
			if l > u {
				x--
			} else {
				y--
			}
		}
	}

	if i > -1 {
		// should never happen if the matrix was constructed properly
		panic("not enough elements in sequence")
	}

	return seq
}
