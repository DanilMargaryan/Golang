package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func SingleHash(in, out chan interface{}) {
	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	for val := range in {
		wg.Add(1)
		go func(val interface{}) {
			defer wg.Done()

			var x1, x2 string
			wgIn := sync.WaitGroup{}

			if val, ok := (val).(int); ok {
				wgIn.Add(2)
				str := strconv.Itoa(val)

				go func() {
					defer wgIn.Done()
					x1 = DataSignerCrc32(str)
				}()
				go func() {
					defer wgIn.Done()

					mutex.Lock()
					md5res := DataSignerMd5(str)
					mutex.Unlock()

					x2 = DataSignerCrc32(md5res)
				}()
			}

			wgIn.Wait()

			fmt.Println(x1 + "~" + x2)
			out <- x1 + "~" + x2
			fmt.Println("out: ", x1+"~"+x2)
		}(val)
	}
	wg.Wait()
	fmt.Println("finish SingleHash")
}

func MultiHash(in, out chan interface{}) {
	wgOut := sync.WaitGroup{}
	const n = 6
	for inResult := range in {
		wgOut.Add(1)
		go func(inResult interface{}) {
			defer wgOut.Done()
			wgIn := sync.WaitGroup{}
			if val, ok := inResult.(string); ok {

				var result [n]string
				for th := 0; th < n; th++ {
					wgIn.Add(1)
					go func(th int) {
						defer wgIn.Done()
						result[th] += DataSignerCrc32(strconv.Itoa(th) + val)
					}(th)
				}
				wgIn.Wait()
				out <- strings.Join(result[:], "")
				fmt.Println("->MultiHash: ", result)
			}
		}(inResult)
	}
	wgOut.Wait()
}

func CombineResults(in, out chan interface{}) {
	var results []string
	for inResult := range in {
		if val, ok := inResult.(string); ok {
			results = append(results, val)
		}
	}
	sort.Strings(results)
	out <- strings.Join(results, "_")
}

func ExecutePipeline(jobs ...job) {
	var in = make(chan interface{})
	var out chan interface{}

	wg := sync.WaitGroup{}

	for _, jobFunc := range jobs {
		out = make(chan interface{})
		wg.Add(1)

		go func(in chan interface{}, out chan interface{}, fn job) {
			defer wg.Done()
			defer close(out)
			fn(in, out)
		}(in, out, jobFunc)

		in = out
	}

	wg.Wait()
}
