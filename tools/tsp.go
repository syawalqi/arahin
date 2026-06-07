package tools

// OptimizeRoute finds the optimal ordering of waypoints using brute-force
// permutation of all middle waypoints. First and last waypoints are fixed
// (user's start and destination).
//
// For n ≤ 5 the brute-force search is optimal and instant.
// For n > 5 it falls back to the original order (thesis scope is 2-5).
// Returns the optimal order as indices and the total cost in seconds.
func OptimizeRoute(matrix *DistanceMatrix, startIdx, endIdx int) ([]int, float64) {
	n := len(matrix.Durations)
	if n <= 2 {
		return []int{0, 1}, matrix.Durations[0][1]
	}

	// Collect middle indices (everything except start and end)
	middle := make([]int, 0)
	for i := 0; i < n; i++ {
		if i != startIdx && i != endIdx {
			middle = append(middle, i)
		}
	}

	// For n > 5, return original order (brute-force would be slow)
	if n > 5 {
		order := make([]int, 0)
		order = append(order, startIdx)
		order = append(order, middle...)
		order = append(order, endIdx)
		var cost float64
		for i := 0; i < len(order)-1; i++ {
			cost += matrix.Durations[order[i]][order[i+1]]
		}
		return order, cost
	}

	// Brute-force all permutations of middle indices
	bestOrder := make([]int, 0)
	bestCost := -1.0

	// Generate permutations using Heap's algorithm
	permute(middle, func(perm []int) {
		order := make([]int, 0, 2+len(perm))
		order = append(order, startIdx)
		order = append(order, perm...)
		order = append(order, endIdx)

		var cost float64
		for i := 0; i < len(order)-1; i++ {
			cost += matrix.Durations[order[i]][order[i+1]]
		}

		if bestCost < 0 || cost < bestCost {
			bestCost = cost
			bestOrder = append([]int{}, order...)
		}
	})

	return bestOrder, bestCost
}

// permute generates all permutations of the input slice, calling f for each.
// Uses Heap's algorithm (non-recursive, efficient).
func permute(a []int, f func([]int)) {
	n := len(a)
	if n == 0 {
		f(a)
		return
	}

	c := make([]int, n)
	f(a)

	i := 0
	for i < n {
		if c[i] < i {
			if i%2 == 0 {
				a[0], a[i] = a[i], a[0]
			} else {
				a[c[i]], a[i] = a[i], a[c[i]]
			}
			f(a)
			c[i]++
			i = 0
		} else {
			c[i] = 0
			i++
		}
	}
}
