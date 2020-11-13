package vector

// number of clients sending/receiving messages
const VECSIZE = 3

type Message struct {
    Message string
    VecTime [VECSIZE]int
}

// returns true if vec1 <= vec2, false otherwise
func LTE(vec1 [VECSIZE]int, vec2 [VECSIZE]int) bool{
	// condition 1:
	//  - there exists an i s.t. vec1[i] != vec2[i]
	cond1 := false
	// condition 2:
	//  - for all i vec1[i] <= vec2[i]
	cond2 := 0

	for i := 0; i < VECSIZE; i++ {
		if vec1[i] != vec2[i] {
			cond1 = true
		}

		if vec1[i] <= vec2[i] {
			cond2++
		}
	}

	if cond1 && cond2 == VECSIZE {
		return true
	}

	return false
}