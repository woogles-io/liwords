package matching

import (
	"errors"
	"github.com/domino14/liwords/pkg/utilities"
)

type Edge struct {
	i int
	j int
	w int
}

type MaxWeightMatching struct {
	edges            []*Edge
	numberOfEdges    int
	numberOfVertexes int
	maxEdgeWeight    int
	endpoints        []int
	neighbend        [][]int
	mate             []int
	label            []int
	labelEnd         []int
	inblossoms       []int
	blossomParents   []int
	blossomChildren  [][]int
	blossomBase      []int
	blossomEndpoints [][]int
	bestEdge         []int
	blossomBestEdges [][]int
	unusedBlossoms   []int
	dualVar          []int
	allowEdge        []bool
	queue            []int
	maxCardinality   bool
}

func NewEdge(i int, j int, w int) *Edge {
	return &Edge{i: i, j: j, w: w}
}

func MinWeightMatching(edges []*Edge, maxCardinality bool) ([]int, error) {
	maxEdgeWeight := -1
	for _, edge := range edges {
		if edge.w > maxEdgeWeight {
			maxEdgeWeight = edge.w
		}
	}
	for _, edge := range edges {
		edge.w = maxEdgeWeight - edge.w
	}
	return maxWeightMatching(edges, maxCardinality)
}

func maxWeightMatching(edges []*Edge, maxCardinality bool) ([]int, error) {

	/*
	   Compute a maximum-weighted matching in the general undirected
	   weighted graph given by "edges".  If "maxcardinality" is true,
	   only maximum-cardinality matchings are considered as solutions.

	   Edges is a sequence of tuples (i, j, wt) describing an undirected
	   edge between vertex i and vertex j with weight wt.  There is at most
	   one edge between any two vertices; no vertex has an edge to itself.
	   Vertices are identified by consecutive, non-negative integers.

	   Return a list "mate", such that mate[i] == j if vertex i is
	   matched to vertex j, and mate[i] == -1 if vertex i is not matched.

	   This function takes time O(n ** 3).

	   Vertices are numbered 0 .. (nvertex-1).
	   Non-trivial blossoms are numbered nvertex .. (2*nvertex-1)

	   Edges are numbered 0 .. (nedge-1).
	   Edge endpoints are numbered 0 .. (2*nedge-1), such that endpoints
	   (2*k) and (2*k+1) both belong to edge k.
	*/

	// Count edges
	numberOfEdges := len(edges)
	// Count vertexes and find maximum edge weight
	numberOfVertexes := 0
	maxEdgeWeight := -1
	for _, edge := range edges {
		if !(edge.i >= 0 && edge.j >= 0 && edge.i != edge.j) {
			return nil, errors.New("ERROR 1")
		}
		if edge.i >= numberOfVertexes {
			numberOfVertexes = edge.i + 1
		}
		if edge.j >= numberOfVertexes {
			numberOfVertexes = edge.j + 1
		}
		if edge.w > maxEdgeWeight {
			maxEdgeWeight = edge.w
		}
	}

	// Create the endpoints array
	// If p is an edge endpoint,
	// endpoint[p] is the vertex to which endpoint p is attached.
	// Not modified by the algorithm.
	endpoints := []int{}
	for _, edge := range edges {
		endpoints = append(endpoints, edge.i)
		endpoints = append(endpoints, edge.j)
	}

	// Create neighbors array
	// If v is a vertex,
	// neighbend[v] is the list of remote endpoints of the edges attached to v.
	// Not modified by the algorithm.
	neighbend := [][]int{}
	for i := 0; i < numberOfVertexes; i++ {
		neighbend = append(neighbend, []int{})
	}
	neighbendLength := len(neighbend)
	for k, edge := range edges {
		if !(edge.i < neighbendLength && edge.j < neighbendLength) {
			return nil, errors.New("ERROR 2")
		}
		neighbend[edge.i] = append(neighbend[edge.i], 2*k+1)
		neighbend[edge.j] = append(neighbend[edge.j], 2*k)
	}

	// If v is a vertex,
	// mate[v] is the remote endpoint of its matched edge, or -1 if it is single
	// (i.e. endpoint[mate[v]] is v's partner vertex).
	// Initially all vertices are single; updated during augmentation.
	mate := []int{}
	for i := 0; i < numberOfVertexes; i++ {
		mate = append(mate, -1)
	}

	// If b is a top-level blossom,
	// label[b] is 0 if b is unlabeled (free);
	//             1 if b is an S-vertex/blossom;
	//             2 if b is a T-vertex/blossom.
	// The label of a vertex is found by looking at the label of its
	// top-level containing blossom.
	// If v is a vertex inside a T-blossom,
	// label[v] is 2 iff v is reachable from an S-vertex outside the blossom.
	// Labels are assigned during a stage and reset after each augmentation.
	label := []int{}
	for i := 0; i < numberOfVertexes*2; i++ {
		label = append(label, 0)
	}

	// If b is a labeled top-level blossom,
	// labelEnd[b] is the remote endpoint of the edge through which b obtained
	// its label, or -1 if b's base vertex is single.
	// If v is a vertex inside a T-blossom and label[v] == 2,
	// labelEnd[v] is the remote endpoint of the edge through which v is
	// reachable from outside the blossom.
	labelEnd := []int{}
	for i := 0; i < numberOfVertexes*2; i++ {
		labelEnd = append(labelEnd, -1)
	}

	// If v is a vertex,
	// inblossoms[v] is the top-level blossom to which v belongs.
	// If v is a top-level vertex, v is itself a blossom (a trivial blossom)
	// and inblossoms[v] == v.
	// Initially all vertices are top-level trivial blossoms.
	inblossoms := []int{}
	for i := 0; i < numberOfVertexes; i++ {
		inblossoms = append(inblossoms, i)
	}

	// If b is a sub-blossom,
	// blossomParents[b] is its immediate parent (sub-)blossom.
	// If b is a top-level blossom, blossomParents[b] is -1.
	blossomParents := []int{}
	for i := 0; i < numberOfVertexes*2; i++ {
		blossomParents = append(blossomParents, -1)
	}

	// If b is a non-trivial (sub-)blossom,
	// blossomChildren[b] is an ordered list of its sub-blossoms, starting with
	// the base and going round the blossom.
	blossomChildren := [][]int{}
	for i := 0; i < numberOfVertexes*2; i++ {
		blossomChildren = append(blossomChildren, []int{})
	}

	// If b is a (sub-)blossom,
	// blossomBase[b] is its base VERTEX (i.e. recursive sub-blossom).
	blossomBase := []int{}
	for i := 0; i < numberOfVertexes; i++ {
		blossomBase = append(blossomBase, i)
	}
	for i := 0; i < numberOfVertexes; i++ {
		blossomBase = append(blossomBase, -1)
	}

	// If b is a non-trivial (sub-)blossom,
	// blossomEndpoints[b] is a list of endpoints on its connecting edges,
	// such that blossomEndpoints[b][i] is the local endpoint of blossomChildren[b][i]
	// on the edge that connects it to blossomChildren[b][wrap(i+1)].
	blossomEndpoints := [][]int{}
	for i := 0; i < numberOfVertexes*2; i++ {
		blossomEndpoints = append(blossomEndpoints, []int{})
	}

	// If v is a free vertex (or an unreached vertex inside a T-blossom),
	// bestEdge[v] is the edge to an S-vertex with least slack,
	// or -1 if there is no such edge.
	// If b is a (possibly trivial) top-level S-blossom,
	// bestEdge[b] is the least-slack edge to a different S-blossom,
	// or -1 if there is no such edge.
	// This is used for efficient computation of delta2 and delta3.
	bestEdge := []int{}
	for i := 0; i < numberOfVertexes*2; i++ {
		bestEdge = append(bestEdge, -1)
	}

	// If b is a non-trivial top-level S-blossom,
	// blossomBestEdges[b] is a list of least-slack edges to neighbouring
	// S-blossoms, or None if no such list has been computed yet.
	// This is used for efficient computation of delta3.
	blossomBestEdges := [][]int{}
	for i := 0; i < numberOfVertexes*2; i++ {
		blossomBestEdges = append(blossomBestEdges, nil)
	}

	// List of currently unused blossom numbers.
	unusedBlossoms := []int{}
	for i := numberOfVertexes; i < numberOfVertexes*2; i++ {
		unusedBlossoms = append(unusedBlossoms, i)
	}

	// If v is a vertex,
	// dualVar[v] = 2 * u(v) where u(v) is the v's variable in the dual
	// optimization problem (multiplication by two ensures integer values
	// throughout the algorithm if all edge weights are integers).
	// If b is a non-trivial blossom,
	// dualVar[b] = z(b) where z(b) is b's variable in the dual optimization
	// problem.
	dualVar := []int{}
	for i := 0; i < numberOfVertexes; i++ {
		dualVar = append(dualVar, maxEdgeWeight)
	}
	for i := 0; i < numberOfVertexes; i++ {
		dualVar = append(dualVar, 0)
	}
	// If allowEdge[k] is true, edge k has zero slack in the optimization
	// problem; if allowEdge[k] is false, the edge's slack may or may not
	// be zero.
	allowEdge := []bool{}
	for i := 0; i < numberOfEdges; i++ {
		allowEdge = append(allowEdge, false)
	}

	// Queue of newly discovered S-vertexes
	queue := []int{}

	wm := &MaxWeightMatching{edges: edges,
		numberOfEdges:    numberOfEdges,
		numberOfVertexes: numberOfVertexes,
		maxEdgeWeight:    maxEdgeWeight,
		endpoints:        endpoints,
		neighbend:        neighbend,
		mate:             mate,
		label:            label,
		labelEnd:         labelEnd,
		inblossoms:       inblossoms,
		blossomParents:   blossomParents,
		blossomChildren:  blossomChildren,
		blossomBase:      blossomBase,
		blossomEndpoints: blossomEndpoints,
		bestEdge:         bestEdge,
		blossomBestEdges: blossomBestEdges,
		unusedBlossoms:   unusedBlossoms,
		dualVar:          dualVar,
		allowEdge:        allowEdge,
		queue:            queue,
		maxCardinality:   maxCardinality}

	err := wm.solveMaxWeightMatching()
	if err != nil {
		return nil, err
	}
	return wm.mate, nil
}

func (wm *MaxWeightMatching) slack(k int) (int, error) {
	if !(k < wm.numberOfEdges) {
		return 0, errors.New("ERROR 3")
	}
	edge := wm.edges[k]
	if !(edge.i < wm.numberOfVertexes*2 && edge.j < wm.numberOfVertexes*2) {
		return 0, errors.New("ERROR 4")
	}
	return wm.dualVar[edge.i] + wm.dualVar[edge.j] - 2*edge.w, nil
}

func (wm *MaxWeightMatching) blossomLeaves(b int, d int) ([]int, error) {
	if d > 1000 {
		return nil, errors.New("ERROR 5")
	}
	leaves := []int{}
	if b < wm.numberOfVertexes {
		leaves = append(leaves, b)
	} else {
		for _, t := range wm.blossomChildren[b] {
			if t < wm.numberOfVertexes {
				leaves = append(leaves, t)
			} else {
				moreLeaves, err := wm.blossomLeaves(t, d+1)
				if err != nil {
					return nil, errors.New("ERROR 6")
				}
				leaves = append(leaves, moreLeaves...)
			}
		}
	}
	return leaves, nil
}

// Assign label t to the top-level blossom containing vertex w
// and record the fact that w was reached through the edge with
// remote endpoint p.
func (wm *MaxWeightMatching) assignLabel(w int, t int, p int, d int) error {
	if d > 1000 {
		return errors.New("ERROR 7")
	}
	if !(w < wm.numberOfVertexes) {
		return errors.New("ERROR 8")
	}
	b := wm.inblossoms[w]
	if !(b < wm.numberOfVertexes*2) {
		return errors.New("ERROR 9")
	}
	if !(wm.label[w] == 0 && wm.label[b] == 0) {
		return errors.New("ERROR 10")
	}
	wm.label[w] = t
	wm.label[b] = t
	wm.labelEnd[w] = p
	wm.labelEnd[b] = p
	wm.bestEdge[w] = -1
	wm.bestEdge[b] = -1
	if t == 1 {
		moreLeaves, err := wm.blossomLeaves(b, 0)
		if err != nil {
			return err
		}
		wm.queue = append(wm.queue, moreLeaves...)
	} else if t == 2 {
		base := wm.blossomBase[b]
		if !(base < wm.numberOfVertexes && wm.mate[base] >= 0 && wm.mate[base] < wm.numberOfEdges*2) {
			return errors.New("ERROR 11")
		}
		err := wm.assignLabel(wm.endpoints[wm.mate[base]], 1, wm.mate[base]^1, d+1)
		if err != nil {
			return err
		}
	}
	return nil
}

// Trace back from vertices v and w to discover either a new blossom
// or an augmenting path. Return the base vertex of the new blossom or -1.
func (wm *MaxWeightMatching) scanBlossom(v int, w int) (int, error) {
	path := []int{}
	base := -1
	for v != -1 || w != -1 {
		// Look for a breadcrumb in v's blossom or put a new breadcrumb.
		if !(v < wm.numberOfVertexes) {
			return 0, errors.New("ERROR 12")
		}
		b := wm.inblossoms[v]
		if !(b < wm.numberOfVertexes*2) {
			return 0, errors.New("ERROR 13")
		}
		if wm.label[b]%8 >= 4 {
			base = wm.blossomBase[b]
			break
		}
		if !(wm.label[b] == 1) {
			return 0, errors.New("ERROR 14")
		}
		path = append(path, b)
		wm.label[b] = 5
		// Trace one step back
		if !(wm.labelEnd[b] == wm.mate[wm.blossomBase[b]]) {
			return 0, errors.New("ERROR 15")
		}
		if wm.labelEnd[b] == -1 {
			// The base of blossom b is single; stop tracing this path.
			v = -1
		} else {
			v = wm.endpoints[wm.labelEnd[b]]
			b = wm.inblossoms[v]
			if !(wm.label[b] == 2) {
				return 0, errors.New("ERROR 16")
			}
			// b is a T-blossom; trace one more step back.
			if !(wm.label[b] >= 0) {
				return 0, errors.New("ERROR 17")
			}
			v = wm.endpoints[wm.labelEnd[b]]
		}
		// Swap v and w so that we alternate between both paths.
		if w != -1 {
			v, w = w, v
		}
	}
	// Remove breadcrumbs
	for _, index := range path {
		wm.label[index] = 1
	}
	return base, nil
}

// Construct a new blossom with given base, containing edge k which
// connects a pair of S vertices. Label the new blossom as S; set its dual
// variable to zero; relabel its T-vertices to S and add them to the queue.
func (wm *MaxWeightMatching) addBlossom(base int, k int) error {
	if !(k < wm.numberOfEdges) {
		return errors.New("ERROR 18")
	}
	edge := wm.edges[k]
	v := edge.i
	w := edge.j
	if !(base < wm.numberOfVertexes && v < wm.numberOfVertexes && w < wm.numberOfVertexes) {
		return errors.New("ERROR 19")
	}
	bb := wm.inblossoms[base]
	bv := wm.inblossoms[v]
	bw := wm.inblossoms[w]
	// Create blossom
	b := wm.unusedBlossoms[len(wm.unusedBlossoms)-1]
	if !(b < wm.numberOfVertexes*2 &&
		bb < wm.numberOfVertexes*2 &&
		bw < wm.numberOfVertexes*2 &&
		bv < wm.numberOfVertexes*2) {
		return errors.New("ERROR 20")
	}
	wm.unusedBlossoms = wm.unusedBlossoms[:len(wm.unusedBlossoms)-1]
	wm.blossomBase[b] = base
	wm.blossomParents[b] = -1
	wm.blossomParents[bb] = b
	// Make list of sub-blossoms and their interconnecting edge endpoints.
	wm.blossomChildren[b] = []int{}
	wm.blossomEndpoints[b] = []int{}
	// Trace back from v to base
	for bv != bb {
		if !(b < wm.numberOfVertexes*2 &&
			bv < wm.numberOfVertexes*2) {
			return errors.New("ERROR 21")
		}
		// Add bv to the new blossom.
		wm.blossomParents[bv] = b
		wm.blossomChildren[b] = append(wm.blossomChildren[b], bv)
		wm.blossomEndpoints[b] = append(wm.blossomEndpoints[b], wm.labelEnd[bv])
		// Trace one step back
		if !(wm.labelEnd[bv] >= 0) {
			return errors.New("ERROR 22")
		}
		if !(wm.labelEnd[bv] < wm.numberOfEdges*2 && v < wm.numberOfVertexes) {
			return errors.New("ERROR 23")
		}
		v = wm.endpoints[wm.labelEnd[bv]]
		bv = wm.inblossoms[v]
	}
	// Reverse lists, add endpoint that connects the pair of S vertices.
	wm.blossomChildren[b] = append(wm.blossomChildren[b], bb)
	utilities.Reverse(wm.blossomChildren[b])
	utilities.Reverse(wm.blossomEndpoints[b])
	wm.blossomEndpoints[b] = append(wm.blossomEndpoints[b], 2*k)
	// Trace back from w to base
	for bw != bb {
		if !(b < wm.numberOfVertexes*2 &&
			bw < wm.numberOfVertexes*2) {
			return errors.New("ERROR 24")
		}
		// Add bw to the new blossom.
		wm.blossomParents[bw] = b
		wm.blossomChildren[b] = append(wm.blossomChildren[b], bw)
		wm.blossomEndpoints[b] = append(wm.blossomEndpoints[b], wm.labelEnd[bw]^1)
		// Trace one step back
		if !(wm.labelEnd[bw] >= 0) {
			return errors.New("ERROR 25")
		}
		if !(wm.labelEnd[bw] < wm.numberOfEdges*2 && w < wm.numberOfVertexes) {
			return errors.New("ERROR 26")
		}
		w = wm.endpoints[wm.labelEnd[bw]]
		bw = wm.inblossoms[w]
	}
	// Set label to S.
	wm.label[b] = 1
	wm.labelEnd[b] = wm.labelEnd[bb]
	// Set dual variable to zero
	wm.dualVar[b] = 0
	// Relabel vertexes
	moreLeaves, err := wm.blossomLeaves(b, 0)
	if err != nil {
		return err
	}
	for _, leaf := range moreLeaves {
		if !(wm.inblossoms[leaf] < wm.numberOfVertexes*2) {
			return errors.New("ERROR 27")
		}
		if wm.label[wm.inblossoms[leaf]] == 2 {
			// This T-vertex now turns into an S-vertex because it becomes
			// part of an S-blossom; add it to the queue.
			wm.queue = append(wm.queue, leaf)
		}
		wm.inblossoms[leaf] = b
	}
	// Compute the best edges
	bestEdgeTo := []int{}
	for i := 0; i < 2*wm.numberOfVertexes; i++ {
		bestEdgeTo = append(bestEdgeTo, -1)
	}
	for _, step := range wm.blossomChildren[b] {
		var nblists [][]int
		if wm.blossomBestEdges[step] == nil {
			// This subblossom does not have a list of least-slack edges;
			// get the information from the vertexes.
			wm.blossomBestEdges[step] = []int{}
			leaves, err := wm.blossomLeaves(step, 0)
			if err != nil {
				return err
			}
			for _, leaf := range leaves {
				neighbors := []int{}
				for _, neighbor := range wm.neighbend[leaf] {
					neighbors = append(neighbors, neighbor/2)
				}
				nblists = append(nblists, neighbors)
			}
		} else {
			nblists = [][]int{wm.blossomBestEdges[step]}
		}
		for _, nblist := range nblists {
			for _, edgeIndex := range nblist {
				edge := wm.edges[edgeIndex]
				i := edge.i
				j := edge.j
				if !(j < wm.numberOfVertexes) {
					return errors.New("ERROR 28")
				}
				if wm.inblossoms[j] == b {
					i, j = j, i
				}
				bj := wm.inblossoms[j]
				if !(bj < wm.numberOfVertexes*2) {
					return errors.New("ERROR 29")
				}
				if bj != b && wm.label[bj] == 1 {
					if bestEdgeTo[bj] == -1 {
						bestEdgeTo[bj] = edgeIndex
					} else {
						edgeSlack, errEdge := wm.slack(edgeIndex)
						if errEdge != nil {
							return errors.New("ERROR 30")
						}
						bestSlack, errBest := wm.slack(bestEdgeTo[bj])
						if errBest != nil {
							return errors.New("ERROR 31")
						}
						if edgeSlack < bestSlack {
							bestEdgeTo[bj] = edgeIndex
						}
					}
				}
			}
		}
		// Forget about least-slack edges of the subblossom
		wm.blossomBestEdges[step] = nil
		wm.bestEdge[step] = -1
	}
	newBestEdges := []int{}
	for _, edgeInt := range bestEdgeTo {
		if edgeInt != -1 {
			newBestEdges = append(newBestEdges, edgeInt)
		}
	}
	wm.blossomBestEdges[b] = newBestEdges
	// Select bestEdge[b]
	for _, edgeInt := range newBestEdges {
		if wm.bestEdge[b] == -1 {
			wm.bestEdge[b] = edgeInt
		} else {
			edgeSlack, errEdge := wm.slack(edgeInt)
			if errEdge != nil {
				return errors.New("ERROR 32")
			}
			bestSlack, errBest := wm.slack(wm.bestEdge[b])
			if errBest != nil {
				return errors.New("ERROR 33")
			}
			if edgeSlack < bestSlack {
				wm.bestEdge[b] = edgeInt
			}
		}
	}
	return nil
}

// Expand the given top-level blossom.
func (wm *MaxWeightMatching) expandBlossom(b int, endStage bool, d int) error {
	if d > 1000 {
		return errors.New("ERROR expandBlossom recursion failure")
	}
	// Convert subblossoms into toplevel blossoms.
	if !(b < wm.numberOfVertexes*2) {
		return errors.New("ERROR 34")
	}
	for _, s := range wm.blossomChildren[b] {
		if !(s < wm.numberOfVertexes*2) {
			return errors.New("ERROR 35")
		}
		wm.blossomParents[s] = -1
		if s < wm.numberOfVertexes {
			wm.inblossoms[s] = s
		} else if endStage && wm.dualVar[s] == 0 {
			// Recursively expand this subblossom
			err := wm.expandBlossom(s, endStage, d+1)
			if err != nil {
				return err
			}
		} else {
			leaves, err := wm.blossomLeaves(s, 0)
			if err != nil {
				return err
			}
			for _, v := range leaves {
				if !(v < wm.numberOfVertexes) {
					return errors.New("ERROR 36")
				}
				wm.inblossoms[v] = s
			}
		}
	}
	// If we expand a T-blossom during a stage, its subblossoms must be
	// relabeled
	if !endStage && wm.label[b] == 2 {
		// Start at the sub-blossom through which the expanding
		// blossom obtained its label, and relabel sub-blossoms untili
		// we reach the base.
		// Figure out through which sub-blossom the expanding blossom
		// obtained its label initially.
		if !(wm.labelEnd[b] >= 0) {
			return errors.New("ERROR 37")
		}
		entryChild := wm.inblossoms[wm.endpoints[wm.labelEnd[b]^1]]
		// Decide in which direction we will go round the blossom.
		j := utilities.IndexOf(entryChild, &wm.blossomChildren[b])
		var jstep int
		var endptrick int
		if j%2 == 1 {
			// Start index is odd; go forward and wrap.
			j -= len(wm.blossomChildren[b])
			jstep = 1
			endptrick = 0
		} else {
			// Start index is even; go backward.
			jstep = -1
			endptrick = 1
		}
		// Move along the blossom until we get to the base.
		p := wm.labelEnd[b]
		for j != 0 {
			// Relabel the T-subblossom.

			jendindex := j - endptrick
			if jendindex < 0 {
				jendindex += len(wm.blossomChildren[b])
			}
			if !(jendindex >= 0 && jendindex < len(wm.blossomEndpoints[b])) {
				return errors.New("ERROR 38")
			}
			wm.label[wm.endpoints[p^1]] = 0
			wm.label[wm.endpoints[wm.blossomEndpoints[b][jendindex]^endptrick^1]] = 0
			err := wm.assignLabel(wm.endpoints[p^1], 2, p, 0)
			if err != nil {
				return err
			}
			// Step to the next S-subblossom and note its forward endpoint.
			wm.allowEdge[wm.blossomEndpoints[b][jendindex]/2] = true

			j += jstep

			jendindex = j - endptrick
			if jendindex < 0 {
				jendindex += len(wm.blossomChildren[b])
			}

			if !(jendindex >= 0 && jendindex < len(wm.blossomEndpoints[b])) {
				return errors.New("ERROR 39")
			}

			p = wm.blossomEndpoints[b][jendindex] ^ endptrick
			// Step to the next T-subblossom
			wm.allowEdge[p/2] = true
			j += jstep
		}
		// Relabel the base T-sub-blossom WITHOUT stepping through to
		// its mate (so don't call assignLabel).
		bv := wm.blossomChildren[b][j]
		wm.label[wm.endpoints[p^1]] = 2
		wm.label[bv] = 2
		wm.labelEnd[wm.endpoints[p^1]] = p
		wm.labelEnd[bv] = p
		wm.bestEdge[bv] = -1
		// Continue along the blossom until we get back to entrychild.
		j += jstep
		jindex := j
		if jindex < 0 {
			jindex += len(wm.blossomChildren[b])
		}

		if !(jindex >= 0 && jindex < len(wm.blossomChildren[b])) {
			return errors.New("ERROR 40")
		}

		for wm.blossomChildren[b][jindex] != entryChild {

			// Examine the vertexes of the sub-blossom to see whether
			// it is reachable from a neighbouring S-vertex outside the
			// expanding blossom.
			bv = wm.blossomChildren[b][jindex]
			if wm.label[bv] == 1 {
				// This subblossom just got label S through one of its
				// neighbors, leave it.
				j += jstep
				jindex = j
				if jindex < 0 {
					jindex += len(wm.blossomChildren[b])
				}
				continue
			}
			reachableVertex := -1
			leaves, err := wm.blossomLeaves(bv, 0)
			if err != nil {
				return err
			}
			for _, v := range leaves {
				if wm.label[v] != 0 {
					reachableVertex = v
					break
				}
			}
			// If the subblossom contains a reachable vertex, assign
			// label T to the subblossom.
			if reachableVertex > 0 {
				if !(wm.label[reachableVertex] == 2 && wm.inblossoms[reachableVertex] == bv) {
					return errors.New("ERROR 41")
				}
				wm.label[reachableVertex] = 0
				if !(bv < wm.numberOfVertexes*2 &&
					wm.blossomBase[bv] < wm.numberOfVertexes &&
					wm.endpoints[wm.mate[wm.blossomBase[bv]]] < wm.numberOfVertexes*2) {
					return errors.New("ERROR 42")
				}
				wm.label[wm.endpoints[wm.mate[wm.blossomBase[bv]]]] = 0
				err := wm.assignLabel(reachableVertex, 2, wm.labelEnd[reachableVertex], 0)
				if err != nil {
					return err
				}
			}
			j += jstep
			jindex = j
			if jindex < 0 {
				jindex += len(wm.blossomChildren[b])
			}
		}
	}
	// Recycle the blossom number.
	wm.label[b] = -1
	wm.labelEnd[b] = -1
	wm.blossomChildren[b] = nil
	wm.blossomEndpoints[b] = nil
	wm.blossomBase[b] = -1
	wm.blossomBestEdges[b] = nil
	wm.bestEdge[b] = -1
	wm.unusedBlossoms = append(wm.unusedBlossoms, b)
	return nil

}

func (wm *MaxWeightMatching) augmentBlossom(b int, v int, d int) error {
	// Bubble up through the blossom tree from vertex v to an immediate
	// subblossom of b.
	if d > 1000 {
		return errors.New("ERROR 43")
	}
	if !(v < wm.numberOfVertexes*2 && b < wm.numberOfVertexes*2) {
		return errors.New("ERROR 44")
	}
	t := v
	for wm.blossomParents[t] != b {
		t = wm.blossomParents[t]
	}
	// Recursively deal with the first subblossom
	if t >= wm.numberOfVertexes {
		err := wm.augmentBlossom(t, v, 0)
		if err != nil {
			return err
		}
	}
	// Decide in which direction we will go round the blossom.
	i := utilities.IndexOf(t, &wm.blossomChildren[b])
	j := i
	var jstep int
	var endptrick int
	var p int
	if i%2 == 1 {
		// Start index is odd; go forward and wrap.
		j -= len(wm.blossomChildren[b])
		jstep = 1
		endptrick = 0
	} else {
		// Start index is even; go backward
		jstep = -1
		endptrick = 1
	}
	// Move along the blossom until we get to the base.
	for j != 0 {
		// Step to the next subblossom and augment it recursively
		j += jstep

		jindex := j
		if jindex < 0 {
			jindex += len(wm.blossomChildren[b])
		}

		jendindex := j - endptrick
		if jendindex < 0 {
			jendindex += len(wm.blossomChildren[b])
		}

		if !(jindex >= 0 && jindex < len(wm.blossomChildren[b]) && jendindex >= 0 && jendindex < len(wm.blossomEndpoints[b])) {
			return errors.New("ERROR 45")
		}

		t = wm.blossomChildren[b][jindex]
		p = wm.blossomEndpoints[b][jendindex] ^ endptrick
		if t >= wm.numberOfVertexes {
			err := wm.augmentBlossom(t, wm.endpoints[p], d+1)
			if err != nil {
				return err
			}
		}
		// Step to the next subblossom and augment it recursively
		j += jstep

		jindex = j
		if jindex < 0 {
			jindex += len(wm.blossomChildren[b])
		}

		t = wm.blossomChildren[b][jindex]
		if t >= wm.numberOfVertexes {
			err := wm.augmentBlossom(t, wm.endpoints[p^1], d+1)
			if err != nil {
				return err
			}
		}
		// Match the edge connecting those subblossoms.
		wm.mate[wm.endpoints[p]] = p ^ 1
		wm.mate[wm.endpoints[p^1]] = p

	}
	wm.blossomChildren[b] = append(wm.blossomChildren[b][i:], wm.blossomChildren[b][:i]...)
	wm.blossomEndpoints[b] = append(wm.blossomEndpoints[b][i:], wm.blossomEndpoints[b][:i]...)
	wm.blossomBase[b] = wm.blossomBase[wm.blossomChildren[b][0]]
	if !(wm.blossomBase[b] == v) {
		return errors.New("ERROR 46")
	}
	return nil
}

func (wm *MaxWeightMatching) augmentMatching(k int) error {
	if !(k < wm.numberOfEdges) {
		return errors.New("ERROR 47")
	}
	edge := wm.edges[k]
	v := edge.i
	w := edge.j
	for _, pair := range [2][2]int{{v, 2*k + 1}, {w, 2 * k}} {
		s := pair[0]
		p := pair[1]

		for {
			if !(s < wm.numberOfVertexes) {
				return errors.New("ERROR 48")
			}
			bs := wm.inblossoms[s]

			if !(wm.label[bs] == 1) {
				return errors.New("ERROR 49")
			}
			if !(wm.labelEnd[bs] == wm.mate[wm.blossomBase[bs]]) {
				return errors.New("ERROR 50")
			}
			// Augment through the S-blossom from s to base.
			if bs >= wm.numberOfVertexes {
				err := wm.augmentBlossom(bs, s, 0)
				if err != nil {
					return err
				}
			}
			// Update wm.mate[s]
			wm.mate[s] = p
			// Trace one step back.
			if wm.labelEnd[bs] == -1 {
				// Reached single vertex; stop.
				break
			}
			if !(wm.labelEnd[bs] < wm.numberOfEdges*2) {
				return errors.New("ERROR 51")
			}
			t := wm.endpoints[wm.labelEnd[bs]]
			bt := wm.inblossoms[t]
			if !(wm.label[bt] == 2) {
				return errors.New("ERROR 52")
			}
			// Trace one step back.
			if !(wm.labelEnd[bt] >= 0) {
				return errors.New("ERROR 53")
			}
			s = wm.endpoints[wm.labelEnd[bt]]
			j := wm.endpoints[wm.labelEnd[bt]^1]
			// Augment through the T-blossom from j to base.
			if !(wm.blossomBase[bt] == t) {
				return errors.New("ERROR 54")
			}
			if bt >= wm.numberOfVertexes {
				err := wm.augmentBlossom(bt, j, 0)
				if err != nil {
					return err
				}
			}
			// Update mate[j]
			wm.mate[j] = wm.labelEnd[bt]
			// Keep the opposite endpoint;
			// it will be assigned to mate[s] in the next step.
			p = wm.labelEnd[bt] ^ 1
		}
	}
	return nil
}

func (wm *MaxWeightMatching) verifyOptimum() error {
	var vdualOffset int
	minDualVarFirstHalf := utilities.MinArr(wm.dualVar[:wm.numberOfVertexes])
	minDualVarSecondHalf := utilities.MinArr(wm.dualVar[wm.numberOfVertexes:])
	if wm.maxCardinality {
		// Vertexes may have negative dual;
		// find a constant non-negative number to add to all vertex duals.
		vdualOffset = utilities.Max(0, -minDualVarFirstHalf)
	} else {
		vdualOffset = 0
	}
	// 0. all dual variables are non-negative
	if !(minDualVarFirstHalf+vdualOffset >= 0) {
		return errors.New("ERROR 55")
	}
	if !(minDualVarSecondHalf >= 0) {
		return errors.New("ERROR 56")
	}
	for k := 0; k < wm.numberOfEdges; k++ {
		edge := wm.edges[k]
		i := edge.i
		j := edge.j
		wt := edge.w
		s := wm.dualVar[i] + wm.dualVar[j] - (2 * wt)
		iblossoms := []int{i}
		jblossoms := []int{j}
		for wm.blossomParents[iblossoms[len(iblossoms)-1]] != -1 {
			iblossoms = append(iblossoms, wm.blossomParents[iblossoms[len(iblossoms)-1]])
		}
		for wm.blossomParents[jblossoms[len(jblossoms)-1]] != -1 {
			jblossoms = append(jblossoms, wm.blossomParents[jblossoms[len(jblossoms)-1]])
		}
		utilities.Reverse(iblossoms)
		utilities.Reverse(jblossoms)
		ilen := len(iblossoms)
		jlen := len(jblossoms)
		ziplen := ilen
		if jlen < ziplen {
			ziplen = jlen
		}
		for l := 0; l < ziplen; l++ {
			bi := iblossoms[l]
			bj := jblossoms[l]
			if bi != bj {
				break
			}
			s += 2 * wm.dualVar[bi]
		}
		if s < 0 {
			return errors.New("ERROR 57")
		}
		if (wm.mate[i] >= 0 && wm.mate[i]/2 == k) || (wm.mate[j] >= 0 && wm.mate[j]/2 == k) {
			if !(wm.mate[i]/2 == k && wm.mate[j]/2 == k) {
				return errors.New("ERROR 58")
			}
			if s != 0 {
				return errors.New("ERROR 59")
			}
		}
	}
	for v := 0; v < wm.numberOfVertexes; v++ {
		if !(wm.mate[v] >= 0 || wm.dualVar[v]+vdualOffset == 0) {
			return errors.New("ERROR 60")
		}
	}
	for b := wm.numberOfVertexes; b < 2*wm.numberOfVertexes; b++ {
		if wm.blossomBase[b] > 0 && wm.dualVar[b] > 0 {
			if !(len(wm.blossomEndpoints[b])%2 == 1) {
				return errors.New("ERROR 61")
			}
			p := wm.blossomEndpoints[b][1]
			if !(wm.mate[wm.endpoints[p]] == p^1) {
				return errors.New("ERROR 62")
			}
			if !(wm.mate[wm.endpoints[p^1]] == p) {
				return errors.New("ERROR 63")
			}
		}
	}
	return nil
}

func (wm *MaxWeightMatching) solveMaxWeightMatching() error {

	for t := 0; t < wm.numberOfVertexes; t++ {
		// Each iteration of this loop is a "stage".
		// A stage finds an augmenting path and uses that to improve
		// the matching.

		// Remove labels from top-level blossoms/vertexes.
		// Forget about least-slack edges.
		for i := 0; i < wm.numberOfVertexes*2; i++ {
			wm.label[i] = 0
			wm.bestEdge[i] = -1
			if i >= wm.numberOfVertexes {
				wm.blossomBestEdges[i] = nil
			}
		}

		// Loss of labeling means that we can not be sure that currently
		// allowable edges remain allowable througout this stage.
		for i := 0; i < wm.numberOfEdges; i++ {
			wm.allowEdge[i] = false
		}

		// Make queue empty
		wm.queue = []int{}
		for v := 0; v < wm.numberOfVertexes; v++ {
			if wm.mate[v] == -1 && wm.label[wm.inblossoms[v]] == 0 {
				err := wm.assignLabel(v, 1, -1, 0)
				if err != nil {
					return err
				}
			}
		}
		// Loop until we succeed in augmenting the matching
		augmented := false
		for {
			// Each iteration of this loop is a "substage".
			// A substage tries to find an augmenting path;
			// if found, the path is used to improve the matching and
			// the stage ends. If there is no augmenting path, the
			// primal-dual method is used to pump some slack out of
			// the dual variables.
			// Continue labeling until all vertices which are reachable
			// through an alternating path have got a label.
			for len(wm.queue) > 0 && !augmented {
				// Take an S vertex from the queue.
				v := wm.queue[len(wm.queue)-1]
				wm.queue = wm.queue[:len(wm.queue)-1]
				if !(wm.label[wm.inblossoms[v]] == 1) {
					return errors.New("ERROR 64")
				}

				// Scan it's neighbors
				for _, p := range wm.neighbend[v] {
					k := p / 2
					w := wm.endpoints[p]
					// w is a neighbor to v
					if wm.inblossoms[v] == wm.inblossoms[w] {
						// This edge is internal to a blossom; ignore it
						continue
					}
					var kslack int
					var err error
					if !wm.allowEdge[k] {
						kslack, err = wm.slack(k)
						if err != nil {
							return err
						}
						if kslack <= 0 {
							// edge k has zero slack => it is allowable
							wm.allowEdge[k] = true
						}
					}
					if wm.allowEdge[k] {
						if wm.label[wm.inblossoms[w]] == 0 {
							// (C1) w is a free vertex;
							// label w with T and label its mate with S (R12)
							err = wm.assignLabel(w, 2, p^1, 0)
							if err != nil {
								return err
							}
						} else if wm.label[wm.inblossoms[w]] == 1 {
							// (C2) w is an S-vertex (not in the same blossom);
							// follow back-links to discover either an
							// augmenting path or a bew blossom.
							base, err := wm.scanBlossom(v, w)
							if err != nil {
								return err
							}
							if base >= 0 {
								// Found a new blossom; add it to the blossom
								// bookkeeping and turn it into an S-blossom.
								err = wm.addBlossom(base, k)
								if err != nil {
									return err
								}
							} else {
								// Found an augmenting path; augment the
								// matching and end this stage.
								err = wm.augmentMatching(k)
								if err != nil {
									return err
								}
								augmented = true
								break
							}
						} else if wm.label[w] == 0 {
							// w is inside a T-blossom, but w itself has not
							// yet been reached from outside the blossom;
							// mark it as reached (we need this to relabel
							// during T-blossom expansion).
							if wm.label[wm.inblossoms[w]] != 2 {
								return errors.New("ERROR 64")
							}
							wm.label[w] = 2
							wm.labelEnd[w] = p ^ 1
						}
					} else if wm.label[wm.inblossoms[w]] == 1 {
						// keep track of the least-slack non-allowable edge to
						// a different S-blossom.
						b := wm.inblossoms[v]
						if wm.bestEdge[b] == -1 {
							wm.bestEdge[b] = k
						} else {
							slackb, err := wm.slack(wm.bestEdge[b])
							if err != nil {
								return err
							}
							if kslack < slackb {
								wm.bestEdge[b] = k
							}
						}
					} else if wm.label[w] == 0 {
						// w is a free vertex (or an unreached vertex inside
						// a T-blossom) but we can not reach it yet;
						// keep track of the least-slack edge that reaches w.
						if wm.bestEdge[w] == -1 {
							wm.bestEdge[w] = k
						} else {
							slackw, err := wm.slack(wm.bestEdge[w])
							if err != nil {
								return err
							}
							if kslack < slackw {
								wm.bestEdge[w] = k
							}
						}
					}
				}
			}
			if augmented {
				break
			}

			// There is no augmenting path under these constraints;
			// compute delta and reduce slack in the optimization problem.
			// (Note that our vertex dual variables, edge slacks and deltas
			// are premultipllied by two.)
			deltaType := -1
			delta := -1
			deltaEdge := -1
			deltaBlossom := -1

			// Compute delta1: the minimum value of any vertex dual.
			if !wm.maxCardinality {
				deltaType = 1
				delta = utilities.MinArr(wm.dualVar[:wm.numberOfVertexes])
			}

			// Compute delta2: the minimum slack on any edge between
			// an S-vertex and a free vertex.
			for v := 0; v < wm.numberOfVertexes; v++ {
				if wm.label[wm.inblossoms[v]] == 0 && wm.bestEdge[v] != -1 {
					d, err := wm.slack(wm.bestEdge[v])
					if err != nil {
						return err
					}
					if deltaType == -1 || d < delta {
						delta = d
						deltaType = 2
						deltaEdge = wm.bestEdge[v]
					}
				}

			}

			// Compute delta3: half the minimum slack on any edge between
			// a pair of S-blossoms.
			for b := 0; b < wm.numberOfVertexes*2; b++ {
				if wm.blossomParents[b] == -1 && wm.label[b] == 1 && wm.bestEdge[b] != -1 {
					kslack, err := wm.slack(wm.bestEdge[b])
					if err != nil {
						return err
					}
					d := kslack / 2
					if deltaType == -1 || d < delta {
						delta = d
						deltaType = 3
						deltaEdge = wm.bestEdge[b]
					}
				}
			}

			// Compute delta4: minimum z variable of any T-blossom.
			for b := wm.numberOfVertexes; b < wm.numberOfVertexes*2; b++ {
				if wm.blossomBase[b] >= 0 &&
					wm.blossomParents[b] == -1 &&
					wm.label[b] == 2 &&
					(deltaType == -1 || wm.dualVar[b] < delta) {
					delta = wm.dualVar[b]
					deltaType = 4
					deltaBlossom = b
				}
			}

			if deltaType == -1 {
				// No further improvement possible; max-cardinality optimum
				// reached. Do a final delta update to make the optimum
				// verifyable
				deltaType = 1
				delta = utilities.Max(0, utilities.MinArr(wm.dualVar[:wm.numberOfVertexes]))
			}

			// Update dual variables according to delta.
			for v := 0; v < wm.numberOfVertexes; v++ {
				if wm.label[wm.inblossoms[v]] == 1 {
					// S-vertex: 2 * u = 2*u + 2*deltas
					wm.dualVar[v] -= delta
				} else if wm.label[wm.inblossoms[v]] == 2 {
					// T-vertex: 2*u = 2*u + 2*delta
					wm.dualVar[v] += delta
				}
			}

			for b := wm.numberOfVertexes; b < wm.numberOfVertexes*2; b++ {
				if wm.blossomBase[b] >= 0 && wm.blossomParents[b] == -1 {
					if wm.label[b] == 1 {
						// toplevel S-blososm: z = z + 2*delta
						wm.dualVar[b] += delta
					} else if wm.label[b] == 2 {
						// toplevel T-blossom: z = z - 2*delta
						wm.dualVar[b] -= delta
					}
				}
			}

			// Take action at the point where minimum delta occurred.
			if deltaType == 1 {
				// No further improvement possible; optimum reached.
				break
			} else if deltaType == 2 {
				// Use the least-slack edge to continue the search.
				wm.allowEdge[deltaEdge] = true
				edge := wm.edges[deltaEdge]
				i := edge.i
				j := edge.j
				if wm.label[wm.inblossoms[i]] == 0 {
					i, j = j, i
				}
				if wm.label[wm.inblossoms[i]] != 1 {
					return errors.New("ERROR 65")
				}
				wm.queue = append(wm.queue, i)

			} else if deltaType == 3 {
				// Use the least-slack edge to continue the search
				wm.allowEdge[deltaEdge] = true
				edge := wm.edges[deltaEdge]
				i := edge.i
				if wm.label[wm.inblossoms[i]] != 1 {
					return errors.New("ERROR 66")
				}
				wm.queue = append(wm.queue, i)
			} else if deltaType == 4 {
				// Expand the least-z blossom.
				err := wm.expandBlossom(deltaBlossom, false, 0)
				if err != nil {
					return err
				}
			}
			// End of this substage
		}
		if !augmented {
			break
		}

		// End of a stage; expand all S-blossom which have dualVar = 0
		for b := wm.numberOfVertexes; b < wm.numberOfVertexes*2; b++ {
			if wm.blossomParents[b] == -1 && wm.blossomBase[b] >= 0 && wm.label[b] == 1 && wm.dualVar[b] == 0 {
				err := wm.expandBlossom(b, true, 0)
				if err != nil {
					return err
				}
			}
		}
	}
	// Verify that we reached the optimum solution.
	err := wm.verifyOptimum()
	if err != nil {
		return err
	}
	// Transform mate[] such that mate[v] is the vertex to which v is paired.
	for v := 0; v < wm.numberOfVertexes; v++ {
		if wm.mate[v] >= 0 {
			wm.mate[v] = wm.endpoints[wm.mate[v]]
		}
	}
	for v := 0; v < wm.numberOfVertexes; v++ {
		if !(wm.mate[v] == -1 || wm.mate[wm.mate[v]] == v) {
			return errors.New("ERROR 67")
		}
	}
	return nil
}
