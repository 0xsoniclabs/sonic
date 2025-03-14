package scrambler

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// GetExecutionOrder returns the order in which transactions should be processed.
// If processed in this order, the transactions can be processed without violating
// any dependencies. Transactions which cannot be processed due to missing
// dependencies are not included in the result.
//
// Beyond the order defined by dependencies, the order of transactions is
// randomized in a deterministic way. This is to avoid the risk of transaction
// ordering attacks.
func GetExecutionOrder(transactions []transaction, pick ...tieBreaker) []transaction {
	// Step 1: identify partitioning of transactions such that each partition is
	// a set of transactions that have dependencies only within the partition.
	partitions := partition(transactions)

	// Step 2: sort transactions within partitions respecting the dependencies.
	// This step may also remove transactions that can not be processed.
	for i, partition := range partitions {
		partitions[i] = sortPartition(partition, pick...)
	}

	// Step 3: interleave partitions using a random order.
	// --- !! ATTENTION !! ---
	// TODO: to make this function deterministic the partitions need to be
	// sorted before interleaving them. This is not implemented yet.
	// --- !! ATTENTION !! ---
	return interleavePartitions(partitions)
}

type transaction struct {
	main action
	auth []action
}

func (tx transaction) actions() []action {
	return append([]action{tx.main}, tx.auth...)
}

type action struct {
	sender int
	nonce  int
}

// --- Partitioning ---

// partition partitions the transactions into groups of transactions that can be
// sorted independently. Transactions without any inter-dependency in their
// execution order end up in different lists. Transactions with
// inter-dependencies end up in the same sub-list.
func partition(transactions []transaction) [][]transaction {
	return partition2(transactions)
}

func partition1(transactions []transaction) [][]transaction {

	// WARNING: This is a proof-of-concept that is not necessarily efficient.
	// It is a simple implementation that is not optimized to minimize the
	// worst-case runtime complexity. It is intended for prototype purposes
	// only.

	// Initially, every transaction is its own partition.
	partition := make([]int, len(transactions))
	for i := range transactions {
		partition[i] = i
	}

	// We merge partitions until no more merges are possible.
	for {
		merged := false

		for i, txA := range transactions {
			for j, txB := range transactions {
				if i == j {
					continue
				}

				// if the transaction are already in the same partition, we can
				// skip another check.
				if partition[i] == partition[j] {
					continue
				}

				// test whether there is a dependency between the two transactions.
				if !dependsOn(txA, txB) {
					continue
				}

				// merge the partitions of the two transactions.
				new := partition[i]
				old := partition[j]
				if old < new {
					new, old = old, new
				}
				for k := range partition {
					if partition[k] == old {
						partition[k] = new
					}
				}

				merged = true
			}
		}

		if !merged {
			break
		}
	}

	grouped := map[int][]transaction{}
	for i, p := range partition {
		grouped[p] = append(grouped[p], transactions[i])
	}

	res := make([][]transaction, 0, len(grouped))
	for _, group := range grouped {
		res = append(res, group)
	}
	return res
}

func dependsOn(a, b transaction) bool {
	for _, a := range a.actions() {
		for _, b := range b.actions() {
			if a.sender == b.sender {
				return true
			}
		}
	}
	return false
}

type unit struct{}

func partition2(transactions []transaction) [][]transaction {

	// Partitions the given list of transactions into groups of
	// interdependent transactions. Transactions within different groups can be
	// processed in an arbitrary order, while transactions within the same group
	// have interdependencies that need to be sorted.

	type sender = int
	type tx = int
	type component = int

	// Step 1: create a sender -> transaction index mapping.
	// This step is O(|actions|).
	senderToTx := map[sender][]tx{}
	for i, tx := range transactions {
		for _, a := range tx.actions() {
			senderToTx[a.sender] = append(senderToTx[a.sender], i)
		}
	}

	// Step 2: create a graph G = (N, E) where
	// - N is the set of sender addresses
	// - E = { {a, b} | a, b in N, a != b, there is a transaction touching a and b }
	// For this we have
	// - |N| <= |actions|
	// - |E| <= |actions|
	graph := map[sender]map[sender]unit{}
	// This loop is O(|senderToTx|) = O(|actions|).
	for s := range senderToTx {
		graph[s] = map[sender]unit{}
	}
	// This loop is O(|actions|).
	for _, tx := range transactions {
		// By connecting the senders of the actions in the transaction, in a
		// chain-like fashion, we indicate that all of these senders are
		// required to be part of the same connected component. A simple chain
		// is sufficient for computing components, we do not need a fully
		// connected graph.
		actions := tx.actions()
		for i := range len(actions) - 1 {
			a := actions[i]
			b := actions[i+1]
			if a.sender == b.sender {
				continue
			}
			graph[a.sender][b.sender] = unit{}
			graph[b.sender][a.sender] = unit{}
		}
	}

	// Step 3: find connected components in the graph.
	// This step is O(|E|) = O(|actions|).
	components := map[sender]component{}
	numComponents := 0
	for sender := range graph {
		if _, found := components[sender]; found {
			continue
		}
		component := numComponents
		numComponents++
		for element := range getConnectedNodes(graph, sender) {
			components[element] = component
		}
	}

	// Step 4: group transactions by connected components.
	// This step is O(|transactions|).
	res := make([][]transaction, numComponents)
	for _, tx := range transactions {
		component := components[tx.main.sender]
		res[component] = append(res[component], tx)
	}

	return res
}

func getConnectedNodes[N comparable](graph map[N]map[N]unit, seed N) iter.Seq[N] {
	return func(yield func(N) bool) {
		seen := map[N]unit{}
		workList := []N{seed}
		for len(workList) > 0 {
			node := workList[0]
			workList = workList[1:]
			if !yield(node) {
				return
			}
			seen[node] = unit{}
			for neighbor := range graph[node] {
				if _, seen := seen[neighbor]; !seen {
					workList = append(workList, neighbor)
				}
			}
		}
	}
}

// --- Sorting ---

func sortPartition(partition []transaction, tieBreaker ...tieBreaker) []transaction {

	// WARNING: This is a proof-of-concept that is not necessarily efficient.
	// It is a simple implementation that is not optimized to minimize the
	// worst-case runtime complexity. It is intended for prototype purposes
	// only.

	// We start by determining the current nonces of all senders referenced in
	// the partition. Here, we assume that the initial nonce is the smallest
	// nonce that is referenced by any transaction in the partition. This could
	// be improved by fetching the actual nonce from the database.

	nonces := state{} // sender -> nonce
	for _, tx := range partition {
		for _, a := range tx.actions() {
			if nonce, found := nonces[a.sender]; !found || a.nonce < nonce {
				nonces[a.sender] = a.nonce
			}
		}
	}

	// Select the heuristic to be used to break ties when multiple transactions
	// can be processed at the same time.
	pick := pickFirst
	if len(tieBreaker) > 0 {
		pick = tieBreaker[0]
	}

	// Use the selected heuristic to sort the transactions.
	return getTransactionOrder(partition, nonces, pick)
}

// getTransactionOrder returns the order in which transactions should be
// processed. If processed in this order, the transactions can be processed
// without violating any dependencies. Transactions which cannot be processed
// due to missing dependencies are not included in the result.
//
// The provided tie breaker is used to select to pick one out of multiple
// options in case multiple transactions can be processed at the same time.
// This function is deterministic if the tie breaker is deterministic.
func getTransactionOrder(
	transactions []transaction,
	nonces state,
	pick tieBreaker,
) []transaction {
	// From here on, we iteratively search for transactions that can be
	// processed and select among those the one that should be processed next.
	// We continue until no more transactions can be processed.
	var res []transaction
	for {
		// Find all transactions that can be processed.
		ready := []transaction{}
		for _, tx := range transactions {
			if nonces[tx.main.sender] == tx.main.nonce {
				ready = append(ready, tx)
			}
		}
		if len(ready) == 0 {
			return res
		}

		// Select the transaction that should be processed next.
		next := pick(ready, transactions, nonces)
		res = append(res, next)

		// Update the nonces of the senders.
		nonces.apply(next)
	}
}

// tieBreaker is a function that selects one out of multiple transactions that
// can be processed at the same time.
type tieBreaker func(options []transaction, all []transaction, nonces state) transaction

// state is a utility type to track the current nonce of each account while
// sorting transactions.
type state map[int]int // account -> nonce

func (s *state) apply(tx transaction) {
	for _, a := range tx.actions() {
		cur := (*s)[a.sender]
		if a.nonce == cur {
			(*s)[a.sender]++
		}
	}
}

func (s state) copy() state {
	return maps.Clone(s)
}

// pickFirst is a trivial tie breaker that always picks the first transaction.
func pickFirst(
	ready []transaction,
	_ []transaction,
	_ state,
) transaction {
	return ready[0]
}

// pickOptimal is a tie breaker that selects the transaction that allows for the
// most transactions to be processed in total. This implementation is producing
// the optimal result. However, its runtime may be exponential in the size of
// the total set of transactions.
func pickOptimal(
	ready []transaction,
	transactions []transaction,
	nonces state,
) transaction {
	var best transaction
	bestNumberOfTransactions := 0
	var numPassingAuths int
	for _, tx := range ready {
		// Copy the state to avoid modifying the original state.
		noncesCopy := nonces.copy()
		noncesCopy.apply(tx)
		numTransactions := 1 + len(getTransactionOrder(
			transactions, noncesCopy, pickOptimal,
		))

		// Find the number of passing authorizations.
		// Multiple authorizations with the same sender are not taken into account,
		// there is no real world use case for it.
		authPassingCount := 0
		for _, a := range tx.auth {
			if nonces[a.sender] == a.nonce {
				authPassingCount++
			}
		}

		if numTransactions == bestNumberOfTransactions {
			// Pick the one with the most passing authorizations.
			if authPassingCount > numPassingAuths {
				best = tx
				numPassingAuths = authPassingCount
			}
		} else if numTransactions > bestNumberOfTransactions {
			best = tx
			bestNumberOfTransactions = numTransactions
			numPassingAuths = authPassingCount
		}
	}
	return best
}

// --- Interleaving ---

func interleavePartitions[T any](partition [][]T) []T {
	return interleavePartitions3(partition)
}

func interleavePartitions1[T any](partition [][]T) []T {

	// WARNING: This is just a dummy implementation; the real implementation
	// should use a random source to sample from partitions in a random order.

	length := 0
	for _, partition := range partition {
		length += len(partition)
	}
	if length == 0 {
		return nil
	}

	res := make([]T, 0, length)
	for i := range length {
		i := i // TODO: add random source here
		for j := range len(partition) {
			pos := (i + j) % len(partition)
			if len(partition[pos]) > 0 {
				res = append(res, partition[pos][0])
				partition[pos] = partition[pos][1:]
				break
			}
		}
	}
	return res
}

func interleavePartitions2[T any](partition [][]T) []T {

	// Step 1: Create a list of proxies that represent the partitions.
	numTransactions := 0
	// This loop is O(|partition|) = O(|transactions|).
	for _, part := range partition {
		numTransactions += len(part)
	}
	if numTransactions == 0 {
		return nil
	}

	proxies := make([]int, 0, numTransactions)
	// This loop nest is O(|transactions|).
	for i, part := range partition {
		for range len(part) {
			proxies = append(proxies, i)
		}
	}

	// Step2: Shuffle the proxies.
	// This is an implementation of the Fisher-Yates shuffle algorithm.
	// See: https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
	// This loop is O(|transactions|).
	for i := len(proxies) - 1; i > 0; i-- {
		j := 0 // TODO: add random source here, picking a number in [0, i) uniformly
		proxies[i], proxies[j] = proxies[j], proxies[i]
	}

	// Step 3: Create the result by interleaving the parts of the partition
	// according to the shuffled proxies.
	res := make([]T, 0, numTransactions)
	// This loop is O(|transactions|).
	for _, proxy := range proxies {
		res = append(res, partition[proxy][0])
		partition[proxy] = partition[proxy][1:]
	}
	return res
}

func interleavePartitions3[T any](partition [][]T) []T {

	// This implementation determines the random order of interleaving by
	// through the following algorithm:
	//  - given, a random seed R
	//  - we create the a list [0,...,N-1] where N is the number of transactions
	//  - we xor each element of the list with R
	//  - we sort the list
	//  - we use the sorted list to determine the order of transactions
	//
	// See https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle#Sorting
	// for a discussion of this algorithm.

	// Step 1: Create a list of proxies that represent the partitions.
	numTransactions := 0
	// This loop is O(|partition|) = O(|transactions|).
	for _, part := range partition {
		numTransactions += len(part)
	}
	if numTransactions == 0 {
		return nil
	}

	type proxy struct {
		pos  uint
		part int
	}
	proxies := make([]proxy, 0, numTransactions)
	seed := uint(0xaaAAaaAAaa) // TODO: add random seed here
	// This loop nest is O(|transactions|).
	for i, part := range partition {
		for range len(part) {
			pos := uint(len(proxies)) ^ seed
			proxies = append(proxies, proxy{pos, i})
		}
	}

	// Step2: Shuffle the proxies by sorting them according to the seeded position.
	// This step is O(NlogN) where N is |transactions|.
	// If we can find a Radix sort implementation, this could be reduced to O(N).
	slices.SortFunc(proxies, func(a, b proxy) int {
		return cmp.Compare(a.pos, b.pos)
	})

	// Step 3: Create the result by interleaving the parts of the partition
	// according to the shuffled proxies.
	res := make([]T, 0, numTransactions)
	// This loop is O(|transactions|).
	for _, proxy := range proxies {
		res = append(res, partition[proxy.part][0])
		partition[proxy.part] = partition[proxy.part][1:]
	}
	return res
}
