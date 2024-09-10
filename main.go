package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const bufferSize int = 10

type RingIntBuffer struct {
	array []int
	pos   int
	size  int
	m     sync.Mutex
}

// NewRingIntBuffer - создание нового буфера целых чисел
func NewRingIntBuffer(size int) *RingIntBuffer {
	return &RingIntBuffer{make([]int, size), -1, size, sync.Mutex{}}
}

func (r *RingIntBuffer) Push(el int) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.pos == r.size-1 {
		// Сдвигаем все элементы буфера
		// на одну позицию в сторону начала
		for i := 1; i <= r.size-1; i++ {
			r.array[i-1] = r.array[i]
		}
		r.array[r.pos] = el
	} else {
		r.pos++
		r.array[r.pos] = el
	}
}

// Get - получение всех элементов буфера и его последующая очистка
func (r *RingIntBuffer) Get() []int {
	if r.pos < 0 {
		return nil
	}
	r.m.Lock()
	defer r.m.Unlock()
	var output []int = r.array[:r.pos+1]
	return output
}

type StageInt func(<-chan bool, <-chan int) <-chan int

type PipeLineInt struct {
	stages []StageInt
	done   <-chan bool
}

func NewPipelineInt(done <-chan bool, stages ...StageInt) *PipeLineInt {
	return &PipeLineInt{done: done, stages: stages}
}

func (p *PipeLineInt) Run(source <-chan int) <-chan int {
	var c <-chan int = source
	for index := range p.stages {
		c = p.runStageInt(p.stages[index], c)
	}
	return c
}

func (p *PipeLineInt) runStageInt(stage StageInt, sourceChan <-chan int) <-chan int {
	return stage(p.done, sourceChan)
}

func dataSource() (<-chan int, <-chan bool) {
	c := make(chan int)
	done := make(chan bool)
	go func() {
		defer close(done)
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < bufferSize*10; i++ {
			var a int = rand.Intn(1000) - 500
			c <- a
			//fmt.Println(a)
		}
		fmt.Println("End")
	}()
	return c, done
}

func MinesFilter(done <-chan bool, c <-chan int) <-chan int {
	convertedIntChan := make(chan int)
	go func() {
		for {
			select {
			case n := <-c:
				if n > 0 {
					select {
					case convertedIntChan <- n:
					case <-done:
						return
					}
				}
			case <-done:
				return
			}
		}
	}()
	return convertedIntChan
}

func specialFilterStageInt(done <-chan bool, c <-chan int) <-chan int {
	filteredIntChan := make(chan int)
	go func() {
		for {
			select {
			case n := <-c:
				//fmt.Println("Mines ", n)
				if n != 0 && n%3 == 0 {
					//fmt.Println("Mines ", "Y")
					select {
					case filteredIntChan <- n:
					case <-done:
						return
					}
				}
			case <-done:
				return
			}
		}
	}()
	return filteredIntChan
}

func bufferStageInt(done <-chan bool, c <-chan int) <-chan int {
	bufferedIntChan := make(chan int)
	buffer := NewRingIntBuffer(bufferSize)
	go func() {
		for {
			select {
			case data := <-c:
				//fmt.Println("bufferStageInt ", data)
				//fmt.Println("bufferStageInt ", "Y")
				buffer.Push(data)
				fmt.Println("Push ", buffer)
			case <-done:
				fmt.Println("Вывод:", buffer)
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-done:
				return
			}
		}
	}()
	return bufferedIntChan
}

func consumer(done <-chan bool, c <-chan int) {
	for {
		select {
		case data := <-c:
			fmt.Printf("Обработаны данные: %d\n", data)
		case <-done:
			return
		}
	}
}

func main() {
	source, done := dataSource()
	pipeline := NewPipelineInt(done, MinesFilter, specialFilterStageInt, bufferStageInt)
	consumer(done, pipeline.Run(source))
	time.Sleep(1 * time.Second)
}
