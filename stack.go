package main

type item struct {
	value interface{}
	next  *item
}

type Stack struct {
	top  *item
	size int
}

func (s *Stack) Length() int {
	return s.size
}

func (s *Stack) Push(value interface{}) {
	s.top = &item{
		value: value,
		next:  s.top,
	}
	s.size++
}

func (s *Stack) Pop() (value interface{}, exists bool) {
	exists = false
	if s.size > 0 {
		value, s.top = s.top.value, s.top.next
		s.size--
		exists = true
	}
	return
}

func (s *Stack) Peek() (value interface{}, exists bool) {
	exists = false
	if s.size > 0 {
		value = s.top.value
		exists = true
	}
	return
}
