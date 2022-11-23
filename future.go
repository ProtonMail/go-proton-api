package proton

type Future[T any] struct {
	resCh chan res[T]
}

type res[T any] struct {
	val T
	err error
}

func NewFuture[T any](fn func() (T, error)) *Future[T] {
	resCh := make(chan res[T])

	go func() {
		val, err := fn()

		resCh <- res[T]{val: val, err: err}
	}()

	return &Future[T]{resCh: resCh}
}

func (job *Future[T]) Then(fn func(T, error)) {
	go func() {
		res := <-job.resCh

		fn(res.val, res.err)
	}()
}

func (job *Future[T]) Get() (T, error) {
	res := <-job.resCh

	return res.val, res.err
}

type Group[T any] struct {
	futures []*Future[T]
}

func NewGroup[T any]() *Group[T] {
	return &Group[T]{}
}

func (group *Group[T]) Add(fn func() (T, error)) {
	group.futures = append(group.futures, NewFuture(fn))
}

func (group *Group[T]) Result() ([]T, error) {
	var out []T

	for _, future := range group.futures {
		res, err := future.Get()
		if err != nil {
			return nil, err
		}

		out = append(out, res)
	}

	return out, nil
}

func (group *Group[T]) ForEach(fn func(T) error) error {
	for _, future := range group.futures {
		res, err := future.Get()
		if err != nil {
			return err
		}

		if err := fn(res); err != nil {
			return err
		}
	}

	return nil
}
