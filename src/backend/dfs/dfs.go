package dfs

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type Recipe struct {
	Tier    int        `json:"tier"`
	Recipes [][]string `json:"recipes"`
}

var (
	recipeDataMutex sync.RWMutex
	RecipeData      map[string]Recipe
)

// Membaca file JSON ke dalam RecipeData (thread-safe)
func readRecipeJson() bool {
	recipeDataMutex.Lock()
	defer recipeDataMutex.Unlock()

	if RecipeData != nil {
		return true
	}

	file, err := os.Open("./data/recipes_complete.json")
	if err != nil {
		log.Fatal(err)
		return false
	}
	defer file.Close()

	tempData := make(map[string]Recipe)
	if err := json.NewDecoder(file).Decode(&tempData); err != nil {
		log.Fatal(err)
		log.Println("Error decoding JSON:", err)
		return false
	}

	RecipeData = tempData
	log.Println("[DFS] Successfully read JSON file")
	return true
}

// DFS multithreaded
func SearchDFS(element string, maxRecipe int) ([]map[string][]string, float64, int) {
	element = strings.TrimSpace(element)
	nodeChan := make(chan int, 1000)
	resultChan := make(chan map[string][]string, maxRecipe)
	doneChan := make(chan struct{})
	var wg sync.WaitGroup

	startTime := time.Now()

	recipeDataMutex.RLock()
	if RecipeData == nil {
		recipeDataMutex.RUnlock()
		if !readRecipeJson() {
			log.Println("[DFS] Failed to read recipe data")
			return nil, 0, 0
		}
	} else {
		recipeDataMutex.RUnlock()
	}

	recipeDataMutex.RLock()
	recipe, exists := RecipeData[element]
	recipeDataMutex.RUnlock()

	if !exists {
		return nil, float64(time.Since(startTime).Seconds()), 0
	}

	if recipe.Tier == 0 {
		return []map[string][]string{{element: {}}}, float64(time.Since(startTime).Seconds()), 1
	}

	visited := sync.Map{}

	var dfs func(current string, recipeMap map[string][]string)
	dfs = func(current string, recipeMap map[string][]string) {
		select {
		case <-doneChan:
			return
		default:
		}

		recipeDataMutex.RLock()
		currentRecipe := RecipeData[current]
		recipeDataMutex.RUnlock()

		for _, rec := range currentRecipe.Recipes {
			recipeDataMutex.RLock()
			if RecipeData[rec[0]].Tier >= currentRecipe.Tier || RecipeData[rec[1]].Tier >= currentRecipe.Tier {
				recipeDataMutex.RUnlock()
				continue
			}
			recipeDataMutex.RUnlock()

			nodeChan <- 2

			newMap := make(map[string][]string)
			for k, v := range recipeMap {
				newMap[k] = v
			}
			newMap[current] = rec

			allBase := true
			for _, comp := range rec {
				recipeDataMutex.RLock()
				if RecipeData[comp].Tier > 0 {
					allBase = false
				}
				recipeDataMutex.RUnlock()
			}

			if allBase {
				select {
				case resultChan <- newMap:
					if len(resultChan) >= maxRecipe {
						close(doneChan)
					}
				default:
				}
			} else {
				for _, comp := range rec {
					recipeDataMutex.RLock()
					if RecipeData[comp].Tier > 0 {
						recipeDataMutex.RUnlock()
						compKey := current + ":" + comp + strings.Join(rec, ",")
						if _, ok := visited.LoadOrStore(compKey, true); !ok {
							wg.Add(1)
							go func(c string, m map[string][]string) {
								defer wg.Done()
								dfs(c, m)
							}(comp, newMap)
						}
					} else {
						recipeDataMutex.RUnlock()
					}
				}
			}
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		dfs(element, map[string][]string{})
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []map[string][]string
	nodeCount := 0
collectLoop:
	for {
		select {
		case r, ok := <-resultChan:
			if !ok {
				break collectLoop
			}
			results = append(results, r)
		case n := <-nodeChan:
			nodeCount += n
		}
	}

	duration := time.Since(startTime).Seconds()
	return results, duration, nodeCount
}
