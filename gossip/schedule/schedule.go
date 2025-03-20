package schedule

import (
	"cmp"
	"iter"
)

func getExecutionOrder(
	transactions []transaction,
	provider NonceProvider,
	seed uint64,
) []transaction {

	// Step 0: Get the current state of all nonces representing the initial
	// state to be assumed by the scheduling algorithm.
	state := getInitialState(transactions, provider)

	// Step 1: perform clean steps removing duplicates and other
	// redundant information from the transactions. It also fetches for
	transactions = clean(transactions, state)

	// Step 2: identify partitioning of transactions such that each partition is
	// a set of transactions that have dependencies only within the partition.
	partitions := partition(transactions)

	// Step 3: sort transactions within partitions respecting the dependencies.
	// This step may also remove transactions that can not be processed.
	for i, partition := range partitions {
		partitions[i] = sort(partition, state)
	}

	// Step 4: interleave partitions using a deterministic random order.
	return interleave(partitions, seed)
}

// --- Interfaces ---

type NonceProvider interface {
	GetNonce(sender sender) nonce
}

// --- State Initialization ---

func getInitialState(transactions []transaction, provider NonceProvider) state {
	state := state{}
	for _, tx := range transactions {
		for _, a := range tx.actions() {
			if _, found := state[a.sender]; !found {
				state[a.sender] = provider.GetNonce(a.sender)
			}
		}
	}
	return state
}

// --- Preprocessing ---

func clean(transactions []transaction, state state) []transaction {
	transactions = removeDuplicates(transactions)
	transactions = removeUnsatisfiableAuthorizations(transactions)
	transactions = removeTriviallyUnreachableActions(transactions, state)
	return transactions
}

func removeDuplicates(transactions []transaction) []transaction {

	// WARNING: this function is removing duplicates and replacements based on
	// the main action only. It does not consider transaction hashes or gas
	// prices as a real-world implementation would have to do.

	seen := map[action]unit{}
	res := make([]transaction, 0, len(transactions))
	for _, tx := range transactions {
		if _, found := seen[tx.main]; found {
			continue
		}
		seen[tx.main] = unit{}
		res = append(res, tx)
	}

	return res
}

// removeUnsatisfiableAuthorizations removes authorizations that can never be
// successful. Examples are authorizations that have the same sender and nonce
// as the transaction itself or are duplicates of other authorizations in the
// same transaction.
//
// This function is O(|actions|).
func removeUnsatisfiableAuthorizations(transactions []transaction) []transaction {
	for i := range transactions {
		seen := map[action]unit{}
		tx := transactions[i]
		res := transaction{
			main: tx.main,
		}
		for _, auth := range tx.auth {
			// authorizations to the transactions sender account must have
			// a higher nonce than the transaction itself.
			if auth.sender == tx.main.sender && auth.nonce <= tx.main.nonce {
				continue
			}
			// duplicates can be ignored for the scheduling
			if _, found := seen[auth]; found {
				continue
			}
			seen[auth] = unit{}
			res.auth = append(res.auth, auth)
		}
		transactions[i] = res
	}
	return transactions
}

// removeTriviallyUnreachableActions removes actions that can never be reached
// due to the current state of the nonces and the actions present in the
// transactions.
func removeTriviallyUnreachableActions(
	transactions []transaction,
	state state,
) []transaction {

	// collect all actions in an index
	actions := map[action]unit{}
	for _, tx := range transactions {
		actions[tx.main] = unit{}
		for _, auth := range tx.auth {
			actions[auth] = unit{}
		}
	}

	// compute all reachable actions
	reachable := map[action]unit{}
	for sender, nonce := range state {
		reachable[action{sender, nonce}] = unit{}
		for cur := nonce + 1; ; cur++ {
			if _, found := actions[action{sender, cur}]; !found {
				break
			}
			reachable[action{sender, cur}] = unit{}
		}
	}

	// filter out unreachable transactions and actions
	res := make([]transaction, 0, len(transactions))
	for _, tx := range transactions {
		if _, found := reachable[tx.main]; !found {
			continue
		}
		auth := make([]action, 0, len(tx.auth))
		for _, a := range tx.auth {
			if _, found := reachable[a]; !found {
				continue
			}
			auth = append(auth, a)
		}
		res = append(res, transaction{tx.main, auth})
	}
	return res
}

// --- Partitioning ---

// partition partitions the given list of transactions into groups of
// interdependent transactions. Transactions within different groups can be
// processed in an arbitrary order, while transactions within the same group
// have interdependencies that need to be sorted.
func partition(transactions []transaction) [][]transaction {
	type component = int

	// Step 1: create a graph G = (N, E) where
	// - N is the set of sender addresses
	// - E = { {a, b} | a, b in N, a != b, a and b are senders of consecutive actions in a transaction }
	// For this we have
	// - |N| <= |actions|
	// - |E| <= |actions|
	graph := map[sender]map[sender]unit{}
	for _, tx := range transactions {
		// Each sender of a transaction is a node in the graph.
		if _, found := graph[tx.main.sender]; !found {
			graph[tx.main.sender] = map[sender]unit{}
		}
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
			if _, found := graph[a.sender]; !found {
				graph[a.sender] = map[sender]unit{}
			}
			if _, found := graph[b.sender]; !found {
				graph[b.sender] = map[sender]unit{}
			}
			graph[a.sender][b.sender] = unit{}
			graph[b.sender][a.sender] = unit{}
		}
	}

	// Step 2: find connected components in the graph.
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

	// Step 3: group transactions by connected components.
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

// --- Sorting of Partitions ---

func sort(transactions []transaction, state state) []transaction {

	// Create a set of transactions still to be scheduled, identified by their
	// unique main action.
	pending := map[action]unit{}
	for _, tx := range transactions { // O(|transactions|) <= O(|actions|)
		pending[tx.main] = unit{}
	}

	// Initialize the queue of transactions sorted by their evaluation.
	queue := newHeap[action](func(a, b valuedTransaction) int {
		return b.value.compare(a.value) // minimum value has highest priority
	})
	for _, tx := range transactions { // O(|actions|)
		queue.Add(tx.main, valuedTransaction{
			tx:    tx,
			value: evaluate(tx, state, pending),
		})
	}

	// Create a map of dependencies linking actions to transactions they may
	// be affecting. Transactions are identified by their position in the input
	// slice.
	dependencies := map[action][]int{}
	for i, tx := range transactions { // O(|actions|)
		dependencies[tx.main.predecessor()] = append(dependencies[tx.main.predecessor()], i)
		dependencies[tx.main] = append(dependencies[tx.main], i)
		dependencies[tx.main.successor()] = append(dependencies[tx.main.successor()], i)
		for _, a := range tx.auth {
			dependencies[a.predecessor()] = append(dependencies[a.predecessor()], i)
			dependencies[a] = append(dependencies[a], i)
			dependencies[a.successor()] = append(dependencies[a.successor()], i)
		}
	}

	// create the execution order step by step
	res := []transaction{}
	// This loop has a complexity of
	//   O(max(|actions| * log(|transactions|, |transactions| * log(|transactions|)))
	// which is O(|actions| * log(|actions|)).
	for { // up to |transactions| iterations

		// Pick the top candidate and check if it can be processed.
		next, found := queue.Pop() // O(log(|transactions|))
		if !found || !next.value.runnable {
			break
		}

		// Schedule the selected transaction and apply effects.
		res = append(res, next.tx)
		delete(pending, next.tx.main)

		// Update the evaluation of all candidates affected by the scheduled
		// transaction.
		// The following loops have a combined complexity of
		// O(max(|actions|*log(|transactions|), |actions|*O(|tx.auth|)))
		// over all iterations of the outer loop. This is because each
		// dependency is processed at most once and there are at most
		// 3*|actions| dependencies.
		state[next.tx.main.sender]++
		for _, pos := range dependencies[next.tx.main] {
			tx := transactions[pos]
			queue.Update(tx.main, valuedTransaction{ // O(log(|transactions|))
				tx:    tx,
				value: evaluate(tx, state, pending), // O(|tx.auth|)
			})
		}
		delete(dependencies, next.tx.main)
		for _, a := range next.tx.auth {
			if state[a.sender] == a.nonce {
				state[a.sender]++
				for _, pos := range dependencies[a] {
					tx := transactions[pos]
					queue.Update(tx.main, valuedTransaction{ // O(log(|transactions|))
						tx:    tx,
						value: evaluate(tx, state, pending), // O(|tx.auth|)
					})
				}
				delete(dependencies, a)
			}
		}
	}

	return res
}

// value defines the evaluation result used for defining the scheduling priority
// of transactions.
type value struct {
	runnable                                          bool
	numAuthorizationsCollidingWithPendingTransactions int
	numBlockedAuthorizations                          int
	numAuthorizations                                 int
}

func (v value) compare(o value) int {
	if v.runnable != o.runnable {
		if v.runnable {
			return -1
		} else {
			return 1
		}
	}
	if res := cmp.Compare(
		v.numAuthorizationsCollidingWithPendingTransactions,
		o.numAuthorizationsCollidingWithPendingTransactions,
	); res != 0 {
		return res
	}
	if res := cmp.Compare(
		v.numBlockedAuthorizations,
		o.numBlockedAuthorizations,
	); res != 0 {
		return res
	}
	return -cmp.Compare(v.numAuthorizations, o.numAuthorizations)
}

type valuedTransaction struct {
	tx    transaction
	value value
}

// evaluate computes the value of a transaction based on the current state of
// nonces and the set of pending transactions.
//
// This function is O(|tx.auth|).
func evaluate(
	tx transaction,
	nonces state,
	pending map[action]unit,
) value {

	value := value{
		runnable:          nonces[tx.main.sender] == tx.main.nonce,
		numAuthorizations: len(tx.auth),
	}

	for _, a := range tx.auth {
		nonce := nonces[a.sender]
		if nonce > a.nonce { // this authorization is obsolete
			continue
		}
		if _, found := pending[a]; found {
			if a.nonce == nonces[a.sender] {
				value.numAuthorizationsCollidingWithPendingTransactions++
			}
		}
		if a.nonce > nonce {
			value.numBlockedAuthorizations++
		}
	}

	return value
}

// --- Interleaving ---

func interleave[T any](partition [][]T, seed uint64) []T {

	// Step 1: Create a list of proxies that represent the partitions.
	numElements := 0
	// This loop is O(|partition|) = O(|transactions|).
	for _, part := range partition {
		numElements += len(part)
	}
	if numElements == 0 {
		return nil
	}

	proxies := make([]int, 0, numElements)
	// This loop nest is O(numElements).
	for i, part := range partition {
		for range len(part) {
			proxies = append(proxies, i)
		}
	}

	rand := randomGenerator{}
	rand.seed(seed)

	// Step2: Shuffle the proxies.
	// This is an implementation of the Fisher-Yates shuffle algorithm.
	// See: https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
	// This loop is O(numElements).
	for i := len(proxies) - 1; i > 0; i-- {
		j := rand.randN(uint64(i + 1))
		proxies[i], proxies[j] = proxies[j], proxies[i]
	}

	// Step 3: Create the result by interleaving the parts of the partition
	// according to the shuffled proxies.
	res := make([]T, 0, numElements)
	// This loop is O(numElements).
	for _, proxy := range proxies {
		res = append(res, partition[proxy][0])
		partition[proxy] = partition[proxy][1:]
	}
	return res
}

type unit struct{}
