package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"time"

	"github.com/achu-1612/inmem"
)

type User struct {
	Name string
	Age  int
}

// func main() {
// 	gob.Register(User{})
// 	c := inmem.New(context.Background(), inmem.Options{
// 		Sync:         true,
// 		SyncInterval: time.Second * 5,
// 		SyncFilePath: "cache.gob",
// 	})

// 	go func() {
// 		for {
// 			<-time.After(time.Second)
// 			// fmt.Println(c.Size(), time.Now())
// 		}
// 	}()

// 	c.Set("key", "value", 0)
// 	c.Set("key1", "value1", 5)
// 	c.Set("key2", "value2", 15)

// 	go func() {
// 		i := 1
// 		for {
// 			i++
// 			<-time.After(time.Second)
// 			c.Set("key"+strconv.Itoa(i), User{Name: "Achu", Age: 25}, 0)
// 		}
// 	}()

// 	<-time.After(time.Second * 100)

// }

func main() {
	gob.Register(User{})
	c, err := inmem.New(context.Background(), inmem.Options{
		Sync:           true,
		SyncInterval:   time.Second * 5,
		SyncFolderPath: "dump",
		Sharding:       true,
		ShardCount:     6,
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println(c.Size())

	// go func() {
	// 	i := 1
	// 	for {
	// 		i++
	// 		<-time.After(time.Millisecond * 10)
	// 		c.Set("key"+strconv.Itoa(i), User{Name: "Achu", Age: 25}, 0)
	// 	}
	// }()

	<-time.After(time.Second * 1000)

}
