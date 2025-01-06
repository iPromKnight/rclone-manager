package instance_tracker

import "sync"

type InstanceTracker[T any] struct {
	processMap sync.Map
}

func (t *InstanceTracker[T]) Track(key string, instance *T) {
	t.processMap.Store(key, instance)
}

func (t *InstanceTracker[T]) Untrack(key string) {
	t.processMap.Delete(key)
}

func (t *InstanceTracker[T]) Range(f func(key, value interface{}) bool) {
	t.processMap.Range(f)
}

func (t *InstanceTracker[T]) Get(key string) (*T, bool) {
	val, ok := t.processMap.Load(key)
	if !ok {
		return nil, false
	}
	instance, valid := val.(*T)
	if !valid {
		return nil, false
	}
	return instance, true
}
