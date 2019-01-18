package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPluginList(t *testing.T) {
	assert.NotNil(t, NewPluginList())
}

// TestPut 测试 Put 成功
func TestPut(t *testing.T){
	l := NewPluginList()
	mailBox := newMailBox()
	assert.True(t, l.Put(0, mailBox))
}

// TestGet 测试 Get 返回正确结果
func TestGet(t *testing.T){
	l := NewPluginList()
	m := newMailBox()
	l.Put(0, m)
	id := (*mailBox) (nil)
	p, ok := l.Get(id)
	assert.True(t, ok)
	_, ok = p.(*mailBox)
	assert.True(t, ok)
}

// TestLen 测试 Len 返回正确结果
func TestLen(t *testing.T){
	l := NewPluginList()
	m := newMailBox()
	l.Put(0, m)
	assert.Equal(t, 1, l.Len())
}

// TestSort 测试优先级低的排在前面
func TestSort(t *testing.T){
	l := NewPluginList()
	p1 := newMailBox()
	p2 := &MockPlugin{}
	l.Put(2, p1)
	l.Put(1, p2)
	l.SortByPriority()
	sorted := make([]PluginInterface, 0)
	l.Each(func(v PluginInterface){
		sorted = append(sorted, v)
	})
	assert.NotEmpty(t, sorted)
	_, ok := (sorted[0]).(*MockPlugin)
	assert.True(t, ok)
}