package bfs

import (
	"container/list"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"runtime"


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

// Modified isRecipeComplete without logging but preserving functionality
func isRecipeComplete(recipeMap map[string][]string) bool {
    recipeDataMutex.RLock()
    defer recipeDataMutex.RUnlock()

    // Check each element in the recipe map
    for element, components := range recipeMap {
        // Skip checking base elements
        if elementData, exists := RecipeData[element]; exists && elementData.Tier == 0 {
            continue
        }

        // If this element has no components, it's incomplete
        if len(components) == 0 {
            return false
        }

        // Check if all components exist and are properly expanded
        for _, component := range components {
            // Check if component exists in RecipeData
            componentData, exists := RecipeData[component]
            if !exists {
                return false
            }

            // If component is not a base element (tier > 0), it must be in the recipe map
            if componentData.Tier > 0 {
                if _, hasRecipe := recipeMap[component]; !hasRecipe {
                    return false
                }
            }
        }
    }

    return true
}

// SearchBFS performs a breadth-first search to find recipes for the given element
func SearchBFS(element string, maxRecipe int) ([]map[string][]string, float64, int) {
	progressLogInterval := 500
	lastLogTime := time.Now()

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

	done := make(chan bool)
	resultChan := make(chan map[string][]string)

	numWorkers := runtime.NumCPU()
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

	go func() {
		for r := range resultChan {
			resultMutex.Lock()
			result = append(result, r)
			if len(result) >= maxRecipe {
				safeCloseDone()
			}
			resultMutex.Unlock()
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
				elementToExpand := nextElement.Value.(string)

				recipeDataMutex.RLock()
				elementRecipes := RecipeData[elementToExpand].Recipes
				recipeDataMutex.RUnlock()

				for _, recipe := range elementRecipes {
					nodeCount += 2
					if nodeCount%progressLogInterval == 0 && time.Since(lastLogTime) > 2*time.Second {
						log.Printf("[BFS] Progress - NodeCount: %d, QueueSize: %d, ResultCount: %d",
							nodeCount,
							func() int {
								queueMutex.Lock()
								defer queueMutex.Unlock()
								return recipeQueue.Len()
							}(),
							func() int {
								resultMutex.Lock()
								defer resultMutex.Unlock()
								return len(result)
							}(),
						)
						lastLogTime = time.Now()
					}

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

						newQueue := list.New()
						for el := currentQueue.Front(); el != nil; el = el.Next() {
							newQueue.PushBack(el.Value)
						}

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
				decreaseActive()
			}
		}()
	}

	wg.Wait()
	safeCloseDone()
	close(resultChan)

	duration := time.Since(startTime)
	log.Printf("BFS took %s", duration)

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

// DebugRecipe takes a recipe map and prints it in a readable format
// Kept for API compatibility but without internal logging
func DebugRecipe(recipeMap map[string][]string) {
    // First, find the root element (usually the target element)
    rootElements := findRootElements(recipeMap)

    for _, root := range rootElements {
        printRecipeTree(recipeMap, root, 1)
    }
}

// findRootElements attempts to identify root elements in the recipe
func findRootElements(recipeMap map[string][]string) []string {
    // Create a set of all elements that appear as components
    componentsSet := make(map[string]bool)
    for _, components := range recipeMap {
        for _, component := range components {
            componentsSet[component] = true
        }
    }

    // Find elements that are not components of other elements
    var roots []string
    for element := range recipeMap {
        if !componentsSet[element] {
            roots = append(roots, element)
        }
    }

    // If no clear roots, just return all elements
    if len(roots) == 0 {
        for element := range recipeMap {
            roots = append(roots, element)
        }
    }

    return roots
}

// printRecipeTree recursively prints the recipe tree - keep functionality but remove logs
func printRecipeTree(recipeMap map[string][]string, element string, depth int) {
    components := recipeMap[element]

    if len(components) == 0 {
        return
    }

    for _, component := range components {
        // Recursively print components that have recipes
        if subComponents, hasRecipe := recipeMap[component]; hasRecipe && len(subComponents) > 0 {
            printRecipeTree(recipeMap, component, depth+1)
        }
    }
}

// CountUniqueRecipes eliminates duplicates and returns actual unique recipes
func CountUniqueRecipes(recipes []map[string][]string) int {
    // Create a set of recipe fingerprints
    uniqueFingerprints := make(map[string]bool)

    for _, recipe := range recipes {
        fingerprint := createRecipeFingerprint(recipe)
        uniqueFingerprints[fingerprint] = true
    }

    return len(uniqueFingerprints)
}

// Creates a canonical string representation of a recipe for deduplication
func createRecipeFingerprint(recipe map[string][]string) string {
    // Sort elements first to ensure consistent ordering
    elements := make([]string, 0, len(recipe))
    for element := range recipe {
        elements = append(elements, element)
    }

    // Simple bubble sort
    for i := 0; i < len(elements); i++ {
        for j := i + 1; j < len(elements); j++ {
            if elements[i] > elements[j] {
                elements[i], elements[j] = elements[j], elements[i]
            }
        }
    }

    // Build fingerprint
    var fingerprint strings.Builder
    for _, element := range elements {
        fingerprint.WriteString(element)
        fingerprint.WriteString(":[")

        components := recipe[element]
        // Also sort components for consistency
        for i := 0; i < len(components); i++ {
            for j := i + 1; j < len(components); j++ {
                if components[i] > components[j] {
                    components[i], components[j] = components[j], components[i]
                }
            }
        }

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

// FindMissingRecipe checks if specific recipes are present in the results
func FindMissingRecipe(recipes []map[string][]string, targetElement string, knownRecipes [][]string) [][]string {
    // Convert recipes to a set of fingerprints
    recipeFingerprints := make(map[string]bool)

    for _, recipe := range recipes {
        if components, exists := recipe[targetElement]; exists {
            // Sort components for consistent comparison
            sortedComponents := make([]string, len(components))
            copy(sortedComponents, components)
            for i := 0; i < len(sortedComponents); i++ {
                for j := i + 1; j < len(sortedComponents); j++ {
                    if sortedComponents[i] > sortedComponents[j] {
                        sortedComponents[i], sortedComponents[j] = sortedComponents[j], sortedComponents[i]
                    }
                }
            }

            // Create fingerprint for this recipe's components
            fingerprint := strings.Join(sortedComponents, ",")
            recipeFingerprints[fingerprint] = true
        }
    }

    // Check which known recipes are missing
    var missingRecipes [][]string

    for _, knownRecipe := range knownRecipes {
        // Sort for consistent comparison
        sortedRecipe := make([]string, len(knownRecipe))
        copy(sortedRecipe, knownRecipe)
        for i := 0; i < len(sortedRecipe); i++ {
            for j := i + 1; j < len(sortedRecipe); j++ {
                if sortedRecipe[i] > sortedRecipe[j] {
                    sortedRecipe[i], sortedRecipe[j] = sortedRecipe[j], sortedRecipe[i]
                }
            }
        }

        fingerprint := strings.Join(sortedRecipe, ",")
        if !recipeFingerprints[fingerprint] {
            missingRecipes = append(missingRecipes, knownRecipe)
        }
    }

    return missingRecipes
}