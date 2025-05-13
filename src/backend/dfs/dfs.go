package dfs

import (
	"container/list"
	"encoding/json"
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

// isRecipeComplete checks if a recipe map contains all necessary components
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

// SearchDFS performs a depth-first search to find recipes for the given element
func SearchDFS(element string, maxRecipe int) ([]map[string][]string, float64, int) {
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
	log.Println("Starting DFS for element:", element)
	startTime := time.Now()

	recipeDataMutex.RLock()
	_, ok := RecipeData[element]
	recipeDataMutex.RUnlock()

	if !ok {
		duration := time.Since(startTime)
		log.Println("Element not found in RecipeData")
		log.Printf("DFS took %s", duration)
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
		log.Printf("DFS took %s", duration)
		return result, float64(duration.Seconds()), 1
	}

	// Using a stack instead of a queue for DFS
	recipeStack := list.New()
	var stackMutex sync.Mutex

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

			// Membuat stack untuk elemen-elemen diproses
			stack := list.New()

			recipeDataMutex.RLock()
			// Menambah elemen untuk di proses dengan urutan terbalik sehingga elemen pertama diproses duluan
			if RecipeData[recipe[1]].Tier > 0 {
				stack.PushFront(recipe[1])
			}
			if RecipeData[recipe[0]].Tier > 0 {
				stack.PushFront(recipe[0])
			}
			recipeDataMutex.RUnlock()

			state := map[string]interface{}{
				"recipeMap": currentState,
				"stack":     stack,
			}

			recipeStack.PushFront(state)
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

	seenRecipes := make(map[string]bool)
	var seenMutex sync.Mutex

	go func() {
		for r := range resultChan {
			serialized := serializeRecipe(r)

			seenMutex.Lock()
			if !seenRecipes[serialized] {
				seenRecipes[serialized] = true
				resultMutex.Lock()
				result = append(result, r)
				if len(result) >= maxRecipe {
					safeCloseDone()
				}
				resultMutex.Unlock()
			}
			seenMutex.Unlock()
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

				stackMutex.Lock()
				if recipeStack.Len() == 0 {
					stackMutex.Unlock()

					if getActiveWorkers() == 0 {
						safeCloseDone()
					}

					time.Sleep(10 * time.Millisecond)
					continue
				}

				increaseActive()
				// Mengambil dari depan stack
				currentStateElement := recipeStack.Front()
				recipeStack.Remove(currentStateElement)
				stackMutex.Unlock()

				currentState := currentStateElement.Value.(map[string]interface{})
				currentRecipeMap := currentState["recipeMap"].(map[string][]string)
				currentStack := currentState["stack"].(*list.List)

				// Dianggap resep jika semua elemen bukan dasar masing-masing ditemukan resepnya juga
				if currentStack.Len() == 0 && isRecipeComplete(currentRecipeMap) {
					recipeDone := make(map[string][]string)
					for key, value := range currentRecipeMap {
						recipeDone[key] = value
					}

					select {
					case <-done:
						decreaseActive()
						return
					case resultChan <- recipeDone:
						//log.Printf("[DFS] Found complete recipe with %d elements", len(recipeDone))
					}
					decreaseActive()
					continue
				}

				// Elemen selanjutnya diambil dari depan
				nextElement := currentStack.Front()
				currentStack.Remove(nextElement)
				elementToExpand := nextElement.Value.(string)

				recipeDataMutex.RLock()
				elementRecipes := RecipeData[elementToExpand].Recipes
				recipeDataMutex.RUnlock()

				for _, recipe := range elementRecipes {
					nodeCount += 2
					if nodeCount%progressLogInterval == 0 && time.Since(lastLogTime) > 2*time.Second {
						log.Printf("[DFS] Progress - NodeCount: %d, StackSize: %d, ResultCount: %d",
							nodeCount,
							func() int {
								stackMutex.Lock()
								defer stackMutex.Unlock()
								return recipeStack.Len()
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

						// Membuat stack baru dengan elemen yang tersisa
						newStack := list.New()
						for el := currentStack.Front(); el != nil; el = el.Next() {
							newStack.PushBack(el.Value)
						}

						recipeDataMutex.RLock()
						// Melakukan proses untuk elemen terdalam lebih dulu, hasil ditambahkan ke depan

						if _, ok := newRecipeMap[recipe[1]]; recipe[0] != recipe[1] && !ok && RecipeData[recipe[1]].Tier > 0 {
							newStack.PushFront(recipe[1])
						}
						if _, ok := newRecipeMap[recipe[0]]; !ok && RecipeData[recipe[0]].Tier > 0 {
							newStack.PushFront(recipe[0])
						}
						recipeDataMutex.RUnlock()

						// Jika kedua komponen merupakan elemen dasar, dan tidak ada elemen lain di stack
						// maka merupakan peta resep lengkap
						recipeDataMutex.RLock()
						isRecipe0Base := RecipeData[recipe[0]].Tier == 0
						isRecipe1Base := RecipeData[recipe[1]].Tier == 0
						recipeDataMutex.RUnlock()

						if isRecipe0Base && isRecipe1Base && newStack.Len() == 0 && isRecipeComplete(newRecipeMap) {
							select {
							case <-done:
								return
							case resultChan <- newRecipeMap:
								// Resep lengkap dengan kedua komponen adalah elemen dasar
							}
						}

						newState := map[string]interface{}{
							"recipeMap": newRecipeMap,
							"stack":     newStack,
						}

						stackMutex.Lock()
						recipeStack.PushFront(newState) // Push to front for DFS
						stackMutex.Unlock()
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
	log.Printf("DFS took %s", duration)

	return result, float64(duration.Seconds()), nodeCount
}

// Mengubah peta resep ke bentuk string
func serializeRecipe(r map[string][]string) string {
	var sb strings.Builder
	keys := make([]string, 0, len(r))
	for k := range r {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := r[k]
		sort.Strings(v)
		sb.WriteString(k + ":" + strings.Join(v, ",") + ";")
	}
	return sb.String()
}
