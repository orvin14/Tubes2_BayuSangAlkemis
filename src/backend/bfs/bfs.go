package bfs

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
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
		return false
	}

	// Assign the loaded data to RecipeData
	RecipeData = tempRecipeData
	return true
}

// SearchBFS performs a breadth-first search to find recipes for the given element
func SearchBFS(element string, maxRecipe int) ([]map[string][]string, float64, int) {
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

	recipeDataMutex.RLock()
	_, ok := RecipeData[element]
	recipeDataMutex.RUnlock()

	if !ok {
		duration := time.Since(startTime)
		log.Println("Element not found in RecipeData")
		log.Printf("BFS took %s", duration)
		return nil, float64(duration.Seconds()), 0
	}

	recipeDataMutex.RLock()
	elementTier := RecipeData[element].Tier
	recipeDataMutex.RUnlock()

	var result []map[string][]string
	var resultMutex sync.Mutex

	if elementTier == 0 || len(RecipeData[element].Recipes) == 0 {
		duration := time.Since(startTime)
		result = append(result, map[string][]string{element: {}})
		log.Printf("BFS took %s", duration)
		return result, float64(duration.Seconds()), 1
	}

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
			currentState := make(map[string][]string)
			currentState[element] = recipe

			state := map[string]interface{}{
				"recipeMap": currentState,
				"queue":     list.New(),
			}

			fmt.Printf("Adding initial state: %s", element)
			fmt.Printf("Recipe: %s", recipe[0])
			fmt.Printf("+ %s", recipe[1])
			fmt.Println()

			recipeDataMutex.RLock()
			if RecipeData[recipe[0]].Tier > 0 && recipe0Tier < elementTier {
				state["queue"].(*list.List).PushBack(recipe[0])
				fmt.Printf("Adding to queue: %s\n", recipe[0])
			}
			if RecipeData[recipe[1]].Tier > 0 && recipe[0] != recipe[1] && recipe1Tier < elementTier {
				state["queue"].(*list.List).PushBack(recipe[1])
				fmt.Printf("Adding to queue: %s\n", recipe[1])
			}
			recipeDataMutex.RUnlock()

			recipeQueue.PushBack(state)
		}
	}

	done := make(chan bool)
	resultChan := make(chan map[string][]string)

	numWorkers := max(runtime.NumCPU()/2, 1)
	var wg sync.WaitGroup

	var doneMutex sync.Mutex
	isDoneClosed := false

	safeCloseDone := func() {
		doneMutex.Lock()
		defer doneMutex.Unlock()
		if !isDoneClosed {
			close(done)
			isDoneClosed = true
		}
	}

	var activeWorkers int
	var activeWorkersMutex sync.Mutex

	increaseActive := func() {
		activeWorkersMutex.Lock()
		activeWorkers++
		activeWorkersMutex.Unlock()
	}

	decreaseActive := func() {
		activeWorkersMutex.Lock()
		activeWorkers--
		activeWorkersMutex.Unlock()
	}

	getActiveWorkers := func() int {
		activeWorkersMutex.Lock()
		defer activeWorkersMutex.Unlock()
		return activeWorkers
	}
	
	// Channel to coordinate queue management
	queueFullSignal := make(chan bool, 1)
	
	seenRecipes := make(map[string]bool)
	var seenMutex sync.Mutex
	
	go func() {
		for r := range resultChan {
			serialized := createRecipeFingerprint(r)

			seenMutex.Lock()
			if !seenRecipes[serialized] {
				seenRecipes[serialized] = true
				resultMutex.Lock()
				result = append(result, r)
				fmt.Printf("Found new recipe: %s\n", serialized)
				fmt.Println()
				if len(result) >= maxRecipe {
					safeCloseDone()
				}
				resultMutex.Unlock()
			}
			seenMutex.Unlock()
		}
	}()

	// Queue monitoring goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				queueMutex.Lock()
				queueSize := recipeQueue.Len()
				queueMutex.Unlock()
				
				// If queue was full but now has some space
				if queueSize < 90000 {
					select {
					case queueFullSignal <- false:
						// Signal sent that queue has space
					default:
						// Channel already has signal or no one is waiting
					}
				}
				
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
				}

				queueMutex.Lock()
				if recipeQueue.Len() == 0 {
					queueMutex.Unlock()

					if getActiveWorkers() == 0 {
						safeCloseDone()
					}

					time.Sleep(10 * time.Millisecond)
					continue
				}

				increaseActive()
				currentStateElement := recipeQueue.Front()
				recipeQueue.Remove(currentStateElement)
				queueMutex.Unlock()

				currentState := currentStateElement.Value.(map[string]interface{})
				currentRecipeMap := currentState["recipeMap"].(map[string][]string)
				currentQueue := currentState["queue"].(*list.List)

				if currentQueue.Len() == 0 {
					recipeDone := make(map[string][]string)
					for key, value := range currentRecipeMap {
						recipeDone[key] = value
					}

					select {
					case <-done:
						decreaseActive()
						return
					case resultChan <- recipeDone:
					}
					decreaseActive()
					continue
				}

				nextElement := currentQueue.Front()
				currentQueue.Remove(nextElement)
				for _, ok := currentRecipeMap[nextElement.Value.(string)]; ok; {
					if currentQueue.Len() == 0 {
						break
					}
					nextElement = currentQueue.Front()
					currentQueue.Remove(currentQueue.Front())
					if nextElement == nil {
						break
					}
					_, ok = currentRecipeMap[nextElement.Value.(string)]
				}
				elementToExpand := nextElement.Value.(string)
				fmt.Printf("Expanding: %s\n", elementToExpand)

				recipeDataMutex.RLock()
				elementRecipes := RecipeData[elementToExpand].Recipes
				recipeDataMutex.RUnlock()

				for _, recipe := range elementRecipes {
					nodeCount += 2

					select {
					case <-done:
						decreaseActive()
						return
					default:
					}

					recipeDataMutex.RLock()
					recipe0Tier := RecipeData[recipe[0]].Tier
					recipe1Tier := RecipeData[recipe[1]].Tier
					elementTier := RecipeData[elementToExpand].Tier
					recipeDataMutex.RUnlock()

					if recipe0Tier < elementTier && recipe1Tier < elementTier {
						newRecipeMap := make(map[string][]string)
						for key, value := range currentRecipeMap {
							newRecipeMap[key] = value
						}
						newRecipeMap[elementToExpand] = recipe

						fmt.Printf("Adding new recipe: %s\n", elementToExpand)
						fmt.Printf("Recipe: %s", recipe[0])
						fmt.Printf("+ %s", recipe[1])
						fmt.Println()

						recipe0Found := false
						recipe1Found := false
						newQueue := list.New()
						for el := currentQueue.Front(); el != nil; el = el.Next() {
							newQueue.PushBack(el.Value)
							if el.Value.(string) == recipe[0] {
								recipe0Found = true
							}
							if el.Value.(string) == recipe[1] {
								recipe1Found = true
							}
						}

						recipeDataMutex.RLock()
						if _, ok := newRecipeMap[recipe[0]]; !recipe0Found && !ok && recipe0Tier > 0 {
							newQueue.PushBack(recipe[0])
							fmt.Printf("Adding to queue: %s\n", recipe[0])
						}
						if _, ok := newRecipeMap[recipe[1]]; !recipe1Found && recipe[0] != recipe[1] && !ok && recipe1Tier > 0 {
							newQueue.PushBack(recipe[1])
							fmt.Printf("Adding to queue: %s\n", recipe[1])
						}
						recipeDataMutex.RUnlock()

						newState := map[string]interface{}{
							"recipeMap": newRecipeMap,
							"queue":     newQueue,
						}
						for item := newQueue.Front(); item != nil; item = item.Next() {
							fmt.Printf("Queue item: %s\n", item.Value)
						}

						fmt.Printf("Adding new recipe: %s\n", elementToExpand)
						
						// Improved queue management with backoff strategy
						var addedToQueue bool
						backoffTime := 1 * time.Millisecond
						maxBackoff := 500 * time.Millisecond
						attempts := 0
						maxAttempts := 10 // Set a reasonable limit for attempts
						
						for attempts < maxAttempts {
							select {
							case <-done:
								decreaseActive()
								return
							default:
							}
							
							queueMutex.Lock()
							queueSize := recipeQueue.Len()
							queueFull := queueSize >= 100000
							queueMutex.Unlock()
							
							resultMutex.Lock()
							allRecipes := len(result) >= maxRecipe
							resultMutex.Unlock()
							
							if allRecipes {
								// We've reached our goal, no need to add more
								break
							}
							
							if !queueFull {
								queueMutex.Lock()
								recipeQueue.PushBack(newState)
								queueMutex.Unlock()
								
								fmt.Printf("Added new state: %s\n", elementToExpand)
								addedToQueue = true
								break
							}
							
							// If the queue is full, wait for the queue to have space
							attempts++
							
							if attempts >= 3 {
								// After a few attempts, check if we should yield to let other workers process the queue
								select {
								case <-queueFullSignal:
									// Queue has space now
									continue
								case <-time.After(backoffTime):
									// Apply exponential backoff with jitter
									if backoffTime*2 > maxBackoff {
										backoffTime = maxBackoff
									} else {
										backoffTime = backoffTime * 2
									}
								case <-done:
									decreaseActive()
									return
								}
							} else {
								time.Sleep(backoffTime)
							}
						}
						
						// If we couldn't add to queue after max attempts, handle the situation
						if !addedToQueue && attempts >= maxAttempts {
							fmt.Printf("Failed to add state for %s after %d attempts, queue may be deadlocked\n", 
								elementToExpand, maxAttempts)
							
							// Handle the situation - either discard this state or
							// If there are workers waiting but no progress is being made,
							// we might want to force some cleanup or restart
							
							// Check if the search is still making progress
							select {
							case <-done:
								decreaseActive()
								return
							default:
								// Check if we should terminate due to lack of progress
								resultMutex.Lock()
								resultCount := len(result)
								resultMutex.Unlock()
								
								if resultCount > 0 {
									// If we have some results already, it's okay to discard this state
									fmt.Printf("Discarding this state for %s after max attempts, we have %d results\n", 
										elementToExpand, resultCount)
								}
							}
						}
					}
				}
				decreaseActive()
			}
		}()
	}

	wg.Wait()
	safeCloseDone()
	close(resultChan)

	duration := time.Since(startTime)
	log.Printf("BFS took %s", duration)

	// fmt.Println(result)
	return result, float64(duration.Seconds()), nodeCount
}

// Helper functions for limiting workers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Creates a canonical string representation of a recipe for deduplication
func createRecipeFingerprint(recipe map[string][]string) string {
	// Sort elements first to ensure consistent ordering
	elements := make([]string, 0, len(recipe))
	for element := range recipe {
		elements = append(elements, element)
	}

	sort.Strings(elements)

	// Build fingerprint
	var fingerprint strings.Builder
	for _, element := range elements {
		fingerprint.WriteString(element)
		fingerprint.WriteString(":[")

		components := recipe[element]
		// Also sort components for consistency
		sort.Strings(components)

		for i, component := range components {
			if i > 0 {
				fingerprint.WriteString(",")
			}
			fingerprint.WriteString(component)
		}
		fingerprint.WriteString("]")
	}

	return fingerprint.String()
}