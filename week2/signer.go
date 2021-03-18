package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// type job func(in, out chan interface{})

func worker(j job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)
	j(in, out)
	// return out
}

//ExecutePipeline is a func
func ExecutePipeline(flow ...job) {
	var in, out chan interface{}
	wg := &sync.WaitGroup{}

	for _, j := range flow {
		in = out
		out = make(chan interface{}, 100)
		wg.Add(1)
		go worker(j, in, out, wg)
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for data := range in {
		wg.Add(1)
		go func(out chan interface{}, data string) {
			defer wg.Done()

			crc32 := make(chan string)
			md5 := make(chan string)

			go func(out chan string, data string) {
				defer close(out)

				out <- DataSignerCrc32(data)
			}(crc32, data)

			go func(out chan string, data string) {
				defer close(out)

				mu.Lock()
				str := DataSignerMd5(data)
				mu.Unlock()
				out <- DataSignerCrc32(str)

			}(md5, data)

			res := <-crc32 + "~" + <-md5
			out <- res
		}(out, strconv.Itoa(data.(int)))
		// out <- DataSignerCrc32(num) + "~" + DataSignerCrc32(DataSignerMd5(num))
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for data := range in {
		// fmt.Println(data.(string))
		wg.Add(1)
		go func(out chan interface{}, data string) {
			defer wg.Done()

			mu := &sync.Mutex{}
			wgMH := &sync.WaitGroup{}
			res := make([]string, 6)

			for th := 0; th < 6; th++ {
				wgMH.Add(1)

				go func(th int, data string) {
					defer wgMH.Done()

					crc32 := DataSignerCrc32(strconv.Itoa(th) + data)
					mu.Lock()
					res[th] = crc32
					mu.Unlock()
				}(th, data)
			}
			wgMH.Wait()
			out <- strings.Join(res, "")
		}(out, data.(string))
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var results []string
	for val := range in {
		// fmt.Println("collected val is ", val)
		results = append(results, val.(string))
	}
	sort.Strings(results)
	out <- strings.Join(results, "_")
}

func main() {
	// var recieved uint32
	// flow := []job{
	// 	job(func(in, out chan interface{}) {
	// 		fmt.Println("sent first")
	// 		out <- uint32(1)
	// 		fmt.Println("sent second")
	// 		out <- uint32(3)
	// 		fmt.Println("sent third")
	// 		out <- uint32(4)
	// 	}),
	// 	job(func(in, out chan interface{}) {
	// 		for val := range in {
	// 			out <- val.(uint32) * 3
	// 			fmt.Println("got", val)
	// 			time.Sleep(time.Millisecond * 100)
	// 		}
	// 	}),
	// 	job(func(in, out chan interface{}) {
	// 		for val := range in {
	// 			fmt.Println("collected", val)
	// 			atomic.AddUint32(&recieved, val.(uint32))
	// 		}
	// 	}),
	// }

	inputData := []int{0, 1, 1, 2, 3, 5, 8}
	// inputData := []int{0, 1}

	flow := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
	}

	start := time.Now()
	ExecutePipeline(flow...)
	end := time.Since(start)
	expectedTime := 3 * time.Second

	fmt.Println("time", end, expectedTime)
}
