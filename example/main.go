package main

import (
	"fmt"
	"time"

	"github.com/achu-1612/inmem"
)

func main() {
	c := inmem.New(inmem.Options{})

	go func() {
		for {
			<-time.After(time.Second)
			fmt.Println(c.Size(), time.Now())
		}
	}()

	c.Set("key", "value", 0)
	c.Set("key1", "value1", 5)
	c.Set("key2", "value2", 15)

	go func() {
		<-time.After(time.Second * 6)
		c.Set("key3", "value3", 2)
	}()

	<-time.After(time.Second * 20)

	// fmt.Println(c.Get("key"))
	// fmt.Println(c.Get("key1"))
	// fmt.Println(c.Get("key2"))

}
