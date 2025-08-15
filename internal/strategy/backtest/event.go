package backtest

import (
	"container/heap"
	"time"
)

// EventType represents the type of event
type EventType int

const (
	EventTypeMarketData EventType = iota
	EventTypeOrder
	EventTypeTrade
	EventTypeFunding
)

// Event represents a backtest event
type Event struct {
	Type     EventType
	Time     time.Time
	Priority int
	Data     interface{}
	index    int // heap interface requirement
}

// EventManager manages the event queue
type EventManager struct {
	queue eventQueue
}

// NewEventManager creates a new event manager
func NewEventManager() *EventManager {
	return &EventManager{
		queue: make(eventQueue, 0),
	}
}

// AddEvent adds an event to the queue
func (m *EventManager) AddEvent(event *Event) {
	heap.Push(&m.queue, event)
}

// ProcessEvents processes all events up to the specified time
func (m *EventManager) ProcessEvents(currentTime time.Time) []*Event {
	var processed []*Event

	for m.queue.Len() > 0 {
		event := heap.Pop(&m.queue).(*Event)
		if event.Time.After(currentTime) {
			// Put the event back and break
			heap.Push(&m.queue, event)
			break
		}
		processed = append(processed, event)
	}

	return processed
}

// eventQueue implements heap.Interface
type eventQueue []*Event

func (q eventQueue) Len() int { return len(q) }

func (q eventQueue) Less(i, j int) bool {
	// 首先按时间排序
	if !q[i].Time.Equal(q[j].Time) {
		return q[i].Time.Before(q[j].Time)
	}
	// 时间相同时按优先级排序
	return q[i].Priority > q[j].Priority
}

func (q eventQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *eventQueue) Push(x interface{}) {
	n := len(*q)
	event := x.(*Event)
	event.index = n
	*q = append(*q, event)
}

func (q *eventQueue) Pop() interface{} {
	old := *q
	n := len(old)
	event := old[n-1]
	old[n-1] = nil   // avoid memory leak
	event.index = -1 // for safety
	*q = old[0 : n-1]
	return event
}

// Update modifies the priority and value of an event in the queue
func (q *eventQueue) Update(event *Event, time time.Time, priority int) {
	event.Time = time
	event.Priority = priority
	heap.Fix(q, event.index)
}
