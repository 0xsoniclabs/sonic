package leap

type prefetchIterator[T any] struct {
	buffer  <-chan syncMessage[T]
	seek    chan<- T
	stop    chan<- struct{}
	current *T
}

func Prefetch[T any](iter Iterator[T]) Iterator[T] {
	buffer := make(chan syncMessage[T], 2)
	seek := make(chan T)
	stop := make(chan struct{})
	go func() {
		defer close(buffer)
		defer iter.Release()
		for {
			var next *T
			if iter.Next() {
				cur := iter.Cur()
				next = &cur
			}
			select {
			case buffer <- syncMessage[T]{value: next}:
				// keep going
			case <-stop:
				return
			case target := <-seek:
				buffer <- syncMessage[T]{value: nil, seekDone: true} // signal seek break
				if iter.Seek(target) {
					cur := iter.Cur()
					buffer <- syncMessage[T]{value: &cur}
				}
			}
		}
	}()
	return &prefetchIterator[T]{
		buffer: buffer,
		seek:   seek,
		stop:   stop,
	}
}

var _ Iterator[any] = (*prefetchIterator[any])(nil)

func (i *prefetchIterator[T]) Next() bool {
	msg, ok := <-i.buffer
	if !ok {
		return false
	}
	i.current = msg.value
	return i.current != nil
}

func (i *prefetchIterator[T]) Cur() T {
	return *i.current
}

func (i *prefetchIterator[T]) Seek(value T) bool {
	i.seek <- value
	// drain the buffer up to the seek break
	for msg := range i.buffer {
		if msg.seekDone {
			// seek break
			break
		}
	}
	msg, ok := <-i.buffer
	if !ok {
		return false
	}
	i.current = msg.value
	return i.current != nil
}

func (i *prefetchIterator[T]) Release() {
	close(i.stop)
	for range i.buffer {
		// consume
	}
}

type syncMessage[T any] struct {
	value    *T
	seekDone bool
}
