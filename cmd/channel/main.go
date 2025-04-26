package main

import (
	"fmt"
	"github.com/google/uuid"
	"math/rand/v2"
	"sync"
	"time"
)

func generateUUID(count int, channel chan string) {
	defer close(channel)
	for i := 0; i < count; i++ {
		channel <- uuid.New().String()
	}
}

func generateUUIDV2(count int, channel chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < count; i++ {
		channel <- uuid.New().String()
	}
}

func executeV1(totalUUIDs int) {
	startV1 := time.Now()

	fmt.Println("Executando V1")

	channel := make(chan string, totalUUIDs)
	go generateUUID(totalUUIDs, channel)

	for v := range channel {
		fmt.Println(v)
	}

	elapsedV1 := time.Since(startV1)
	fmt.Printf("Tempo de execução da V1: %s\n", elapsedV1)
}

func executeV2(totalUUIDs, numGoroutines int) {
	startV2 := time.Now()

	fmt.Println("Executando V2")

	channelV2 := make(chan string, totalUUIDs)
	var wg sync.WaitGroup

	uuidsPerGoroutine := totalUUIDs / numGoroutines
	remainingUUIDs := totalUUIDs % numGoroutines

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go generateUUIDV2(uuidsPerGoroutine, channelV2, &wg)
	}
	if remainingUUIDs > 0 {
		wg.Add(1)
		go generateUUIDV2(remainingUUIDs, channelV2, &wg)
	}

	go func() {
		wg.Wait()
		close(channelV2)
	}()

	for v := range channelV2 {
		fmt.Println(v)
	}

	elapsedV2 := time.Since(startV2)
	fmt.Printf("Tempo de execução da V2: %s\n", elapsedV2)
}

func enrichData() (bool, string, int) {
	stringEnrich := rand.IntN(951) + 50 // Gera um número entre 50 e 1000
	intEnrich := rand.IntN(951) + 50    // Gera um número entre 50 e 1000
	boolEnrich := rand.IntN(951) + 50   // Gera um número entre 50 e 1000

	start := time.Now()
	fmt.Println("Executando Enrich")
	var wg sync.WaitGroup
	wg.Add(3)

	boolChan := make(chan bool, 1)
	stringChan := make(chan string, 1)
	intChan := make(chan int, 1)

	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * time.Duration(boolEnrich))
		boolChan <- rand.IntN(2) == 1
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * time.Duration(stringEnrich))
		stringChan <- uuid.New().String()
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * time.Duration(intEnrich))
		intChan <- rand.IntN(1000) + 1
	}()

	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("Tempo de execução do Enrich: %s\n", elapsed)

	numbers := []int{stringEnrich, intEnrich, boolEnrich}
	m := numbers[0]
	for _, num := range numbers {
		if num > m {
			m = num
		}
	}
	fmt.Printf("Max Time de execução do Enrich: %d\n", m)
	return <-boolChan, <-stringChan, <-intChan
}

func main() {
	totalUUIDs := 100
	numGoroutines := 10

	executeV1(totalUUIDs)
	executeV2(totalUUIDs, numGoroutines)

	boolean, text, number := enrichData()
	fmt.Printf("bool: %t, text: %s, int: %d\n", boolean, text, number)
}
