package bfs

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Global mutex for RecipeData
var recipeDataMutex sync.RWMutex
var RecipeData map[string]Recipe

type Recipe struct {
	Tier    int        `json:"tier"`
	Recipes [][]string `json:"recipes"`
}

func readRecipeJson() bool {
	// Use mutex to protect RecipeData during initialization
	recipeDataMutex.Lock()
	defer recipeDataMutex.Unlock()

	// Only read the file if RecipeData is nil
	if RecipeData != nil {
		return true
	}

	// Baca JSON
	file, err := os.Open("./data/recipes_complete.json")
	if err != nil {
		log.Fatal(err)
		return false
	}
	defer file.Close()

	// Create a new map for RecipeData
	tempRecipeData := make(map[string]Recipe)

	// Convert
	if err := json.NewDecoder(file).Decode(&tempRecipeData); err != nil {
		log.Fatal(err)
		log.Println("Error decoding JSON:", err)
		return false
	}

	// Assign the loaded data to RecipeData
	RecipeData = tempRecipeData
	log.Println("Successfully read JSON file")
	return true
}

func TestPrint() {
	// Baca JSON
	file, err := os.Open("./data/recipes_complete.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for element, recipeData := range RecipeData {
		fmt.Printf("Element: %s, Tier: %d, Recipes: %v", element, recipeData.Tier, recipeData.Recipes)
		fmt.Println()
	}
}

func SearchBFS(element string, maxRecipe int) ([]map[string][]string, float64, int) {
	// Cek data with proper synchronization
	recipeDataMutex.Lock()
	if RecipeData == nil {
		recipeDataMutex.Unlock()
		if !readRecipeJson() {
			log.Println("Error reading JSON file")
			return nil, 0, 0
		}
	} else {
		recipeDataMutex.Unlock()
	}

	element = strings.TrimSpace(element)
	nodeCount := 0
	log.Println("Starting BFS for element:", element)
	startTime := time.Now()

	// Use read lock when checking if element exists
	recipeDataMutex.RLock()
	_, ok := RecipeData[element]
	recipeDataMutex.RUnlock()

	if !ok {
		duration := time.Since(startTime)
		log.Println("Element not found in RecipeData")
		log.Printf("BFS took %s", duration)
		return nil, float64(duration), 0
	}

	// When accessing RecipeData, use read locks
	recipeDataMutex.RLock()
	elementTier := RecipeData[element].Tier
	recipeDataMutex.RUnlock()

	var result []map[string][]string
	var resultMutex sync.Mutex

	// Cek apakah merupakan elemen dasar
	if elementTier == 0 {
		duration := time.Since(startTime)
		result = append(result, map[string][]string{element: {}})
		log.Printf("BFS took %s", duration)
		return result, float64(duration), 1
	}

	// thread-safe queue
	recipeQueue := list.New()
	var queueMutex sync.Mutex

	recipeDataMutex.RLock()
	recipes := RecipeData[element].Recipes
	recipeDataMutex.RUnlock()

	nodeCount++
	for _, recipe := range recipes {
		nodeCount += 2

		recipeDataMutex.RLock()
		recipe0Tier := RecipeData[recipe[0]].Tier
		recipe1Tier := RecipeData[recipe[1]].Tier
		elementTier := RecipeData[element].Tier
		recipeDataMutex.RUnlock()

		if recipe0Tier < elementTier && recipe1Tier < elementTier {

			// Variasi resep secara breadth dulu, bukan sampai satu resep jadi
			currentState := make(map[string][]string)
			currentState[element] = recipe

			state := map[string]interface{}{
				"recipeMap": currentState,
				"queue":     list.New(),
			}

			// Menambahkan elemen dari resep ke queue
			recipeDataMutex.RLock()
			if RecipeData[recipe[0]].Tier > 0 {
				state["queue"].(*list.List).PushBack(recipe[0])
			}
			if RecipeData[recipe[1]].Tier > 0 {
				state["queue"].(*list.List).PushBack(recipe[1])
			}
			recipeDataMutex.RUnlock()

			recipeQueue.PushBack(state)
		}
	}

	// Channel untuk menandakan semua goroutine sudah selesai
	done := make(chan bool)

	// Channel untuk receive results
	resultChan := make(chan map[string][]string)

	// Jumlah worker goroutines (number of available CPUs)
	numWorkers := runtime.NumCPU()

	// tunggu semua workers selesai
	var wg sync.WaitGroup

	// Mutex untuk melindungi agar tidak menutup channel done lebih dari sekali
	var doneMutex sync.Mutex
	var isDoneClosed bool = false

	safeCloseDone := func() {
		doneMutex.Lock()
		defer doneMutex.Unlock()

		if !isDoneClosed {
			close(done)
			isDoneClosed = true
		}
	}

	// Goroutine untuk menyimpan result
	go func() {
		for r := range resultChan {
			resultMutex.Lock()
			result = append(result, r)
			if len(result) >= maxRecipe {
				safeCloseDone() // Signal semua workers untuk stop dengan aman
			}
			resultMutex.Unlock()
		}
	}()

	// Launch worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				// Cek apakah sudah ada signal done
				select {
				case <-done:
					return
				default:
					// Lanjut
				}

				queueMutex.Lock()
				if recipeQueue.Len() == 0 {
					queueMutex.Unlock()
					// Cek done signal
					select {
					case <-done:
						return
					default:
						time.Sleep(10 * time.Millisecond) // Small delay to reduce contention (race condition)
						continue
					}
				}

				currentStateElement := recipeQueue.Front()
				recipeQueue.Remove(currentStateElement)
				queueMutex.Unlock()

				currentState := currentStateElement.Value.(map[string]interface{})
				currentRecipeMap := currentState["recipeMap"].(map[string][]string)
				currentQueue := currentState["queue"].(*list.List)

				// Resep sudah jadi
				if currentQueue.Len() == 0 {

					recipeDone := make(map[string][]string)
					for key, value := range currentRecipeMap {
						recipeDone[key] = value
					}

					// Send result melalui channel
					select {
					case <-done:
						return
					case resultChan <- recipeDone:
					}
					continue
				}

				nextElement := currentQueue.Front()
				currentQueue.Remove(nextElement)
				elementToExpand := nextElement.Value.(string)

				// Semua kemungkinan resep untuk elemen ini
				recipeDataMutex.RLock()
				elementRecipes := RecipeData[elementToExpand].Recipes
				recipeDataMutex.RUnlock()

				for _, recipe := range elementRecipes {
					nodeCount += 2
					select {
					case <-done:
						return
					default:
					}

					recipeDataMutex.RLock()
					recipe0Tier := RecipeData[recipe[0]].Tier
					recipe1Tier := RecipeData[recipe[1]].Tier
					elementTier := RecipeData[elementToExpand].Tier
					recipeDataMutex.RUnlock()

					if recipe0Tier < elementTier && recipe1Tier < elementTier {
						// Copy state ini untuk dibikin state baru untuk bfsnya
						newRecipeMap := make(map[string][]string)
						for key, value := range currentRecipeMap {
							newRecipeMap[key] = value
						}

						newRecipeMap[elementToExpand] = recipe

						newQueue := list.New()
						for el := currentQueue.Front(); el != nil; el = el.Next() {
							newQueue.PushBack(el.Value)
						}

						// tambah elemen baru ke queue kalau belum ada di map dan tier > 0
						recipeDataMutex.RLock()
						if _, ok := newRecipeMap[recipe[0]]; !ok && RecipeData[recipe[0]].Tier > 0 {
							newQueue.PushBack(recipe[0])
						}
						if _, ok := newRecipeMap[recipe[1]]; recipe[0] != recipe[1] && !ok && RecipeData[recipe[1]].Tier > 0 {
							newQueue.PushBack(recipe[1])
						}
						recipeDataMutex.RUnlock()

						newState := map[string]interface{}{
							"recipeMap": newRecipeMap,
							"queue":     newQueue,
						}

						queueMutex.Lock()
						recipeQueue.PushBack(newState)
						queueMutex.Unlock()
					}
				}
			}
		}()
	}

	// tunggu semua worker selesai
	wg.Wait()
	safeCloseDone()   // Close done channel safely
	close(resultChan) // Close result channel

	duration := time.Since(startTime)
	log.Printf("BFS took %s", duration)

	return result, float64(duration.Seconds()), nodeCount
}
