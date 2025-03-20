package schedule

type heap[K comparable, V any] struct {
	data  []entry[K, V]
	order func(a, b V) int
	index map[K]int
}

type entry[K any, V any] struct {
	key   K
	value V
}

func newHeap[K comparable, V any](order func(a, b V) int) *heap[K, V] {
	return &heap[K, V]{
		order: order,
		index: make(map[K]int),
	}
}

func (h *heap[K, V]) Add(key K, value V) {
	// Add the key to the index.
	if h.index == nil {
		h.index = make(map[K]int)
	}
	h.index[key] = len(h.data)
	// Add the entry to the data.
	h.data = append(h.data, entry[K, V]{key, value})
	// Move the entry up to where it belongs.
	h.moveUp(len(h.data) - 1)
}

func (h *heap[K, V]) Peek() (V, bool) {
	if len(h.data) == 0 {
		var zero V
		return zero, false
	}
	return h.data[0].value, true
}

func (h *heap[K, V]) Pop() (V, bool) {
	if len(h.data) == 0 {
		var zero V
		return zero, false
	}
	key := h.data[0].key
	value := h.data[0].value
	h.swap(0, len(h.data)-1)
	h.data = h.data[:len(h.data)-1]
	delete(h.index, key)

	// Move the entry down to where it belongs.
	h.moveDown(0)

	return value, true
}

func (h *heap[K, V]) Update(key K, value V) bool {
	pos, found := h.index[key]
	if !found {
		return false
	}

	// Update the value of the entry.
	h.data[pos].value = value

	// Move the entry to where it belongs.
	h.moveUp(pos)
	h.moveDown(pos)
	return true
}

func (h *heap[K, V]) swap(i, j int) {
	h.data[i], h.data[j] = h.data[j], h.data[i]
	h.index[h.data[i].key] = i
	h.index[h.data[j].key] = j
}

func (h *heap[K, V]) compare(a, b V) int {
	if h.order == nil {
		return 0
	}
	return h.order(a, b)
}

func (h *heap[K, V]) moveUp(start int) {
	for i := start; i > 0; {
		j := (i - 1) / 2
		if h.compare(h.data[j].value, h.data[i].value) >= 0 {
			break
		}
		h.swap(i, j)
		i = j
	}
}

func (h *heap[K, V]) moveDown(start int) {
	for i := start; ; {
		j := i*2 + 1
		if j >= len(h.data) {
			break
		}
		if j+1 < len(h.data) && h.compare(h.data[j+1].value, h.data[j].value) > 0 {
			j++
		}
		if h.compare(h.data[i].value, h.data[j].value) >= 0 {
			break
		}
		h.swap(i, j)
		i = j
	}
}
