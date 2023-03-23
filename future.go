package proton

import "github.com/ProtonMail/gluon/queue"

type Future[T any] struct {
	resCh        chan res[T]
	panicHandler queue.PanicHandler
}

type res[T any] struct {
	val T
	err error
}

func NewFuture[T any](panicHandler queue.PanicHandler, fn func() (T, error)) *Future[T] {
	resCh := make(chan res[T])
	job := &Future[T]{
		resCh:        resCh,
		panicHandler: panicHandler,
	}

	go func() {
		defer job.handlePanic()

		val, err := fn()

		resCh <- res[T]{val: val, err: err}
	}()

	return job
}

func (job *Future[T]) Then(fn func(T, error)) {
	go func() {
		defer job.handlePanic()

		res := <-job.resCh

		fn(res.val, res.err)
	}()
}

func (job *Future[T]) handlePanic() {
	if job.panicHandler != nil {
		job.panicHandler.HandlePanic()
	}
}

func (job *Future[T]) Get() (T, error) {
	res := <-job.resCh

	return res.val, res.err
}

type Group[T any] struct {
	futures      []*Future[T]
	panicHandler queue.PanicHandler
}

func NewGroup[T any](panicHandler queue.PanicHandler) *Group[T] {
	return &Group[T]{panicHandler: panicHandler}
}

func (group *Group[T]) Add(fn func() (T, error)) {
	group.futures = append(group.futures, NewFuture(group.panicHandler, fn))
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
