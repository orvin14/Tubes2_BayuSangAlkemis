package search

import (
	"recipe-finder/bfs"
	"recipe-finder/dfs"
)

// DFS
func DFS(element string, maxRecipe int) ([]map[string][]string, float64, int) {
	return dfs.SearchDFS(element, maxRecipe)
}

// BFS
func BFS(element string, maxRecipe int) ([]map[string][]string, float64, int) {
	// Process search in batches if searching for many recipes
	if maxRecipe > 5 {
		// First try with a smaller batch to get quick results
		initialBatch := 3
		results, duration, nodes := bfs.SearchBFS(element, initialBatch)

		// If we found enough recipes or there aren't more to find, return these
		if len(results) < initialBatch {
			return results, duration, nodes
		}

		// If we need more, search for the rest
		log.Printf("Found initial %d recipes, searching for %d more...", len(results), maxRecipe-len(results))
		moreResults, moreDuration, moreNodes := bfs.SearchBFS(element, maxRecipe-len(results))

		// Combine results
		results = append(results, moreResults...)
		return results, duration + moreDuration, nodes + moreNodes
	}

	// For small recipe counts, just search directly
	return bfs.SearchBFS(element, maxRecipe)
}
func loadRecipeData() map[string]struct {
	Tier    int        `json:"tier"`
	Recipes [][]string `json:"recipes"`
} {
	file, err := os.ReadFile("data/recipes_complete.json")
	if err != nil {
		log.Fatalf("Failed to load recipe file: %v", err)
	}
	var data map[string]struct {
		Tier    int        `json:"tier"`
		Recipes [][]string `json:"recipes"`
	}
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Fatalf("Invalid JSON format: %v", err)
	}
	return data
}
func containsAll(arr []string, elem string) bool {
	return arr[0] == elem || arr[1] == elem
}

func copyMap(original map[string][][]string) map[string][][]string {
	newMap := make(map[string][][]string)
	for k, v := range original {
		newSlice := make([][]string, len(v))
		for i, arr := range v {
			newSlice[i] = append([]string{}, arr...)
		}
		newMap[k] = newSlice
	}
	return newMap
}

func mergePaths(a, b map[string][][]string) map[string][][]string {
	result := copyMap(a)
	for k, v := range b {
		result[k] = append(result[k], v...)
	}
	return result
}

func serializePath(path map[string][][]string) string {
	var sb strings.Builder
	keys := make([]string, 0, len(path))
	for k := range path {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sb.WriteString(k + ":")
		for _, inputs := range path[k] {
			sortedInputs := append([]string{}, inputs...)
			sort.Strings(sortedInputs)
			sb.WriteString(strings.Join(sortedInputs, "+") + ",")
		}
		sb.WriteString(";")
	}
	return sb.String()
}

func allLeavesAreBaseElements(path map[string][][]string, base map[string]bool) bool {
	for _, recipes := range path {
		for _, pair := range recipes {
			for _, el := range pair {
				if !base[el] && path[el] == nil {
					return false
				}
			}
		}
	}
	return true
}

// seseorang benerin ini
// 11:15 10 Mei 2025: Gamau, lagi CTF
func BidirectionalSearch(target string, maxRecipe int) ([]map[string][][]string, float64, int) {
	start := time.Now()
	var results []map[string][][]string
	var forwardVisitedCount, backwardVisitedCount, visitedNode int
	baseElements := map[string]bool{"Fire": true, "Earth": true, "Air": true, "Water": true}

	type Node struct {
		Element string
		Path    map[string][][]string
	}

	recipes := loadRecipeData()
	if baseElements[target] {
		return results, time.Since(start).Seconds(), 0
	}

	// Multiple paths per element
	visitedF := map[string][]map[string][][]string{}
	visitedB := map[string][]map[string][][]string{}

	forwardQueue := []Node{}
	for be := range baseElements {
		forwardQueue = append(forwardQueue, Node{be, map[string][][]string{}})
		visitedF[be] = []map[string][][]string{{}}
	}

	backwardQueue := []Node{{target, map[string][][]string{}}}
	visitedB[target] = []map[string][][]string{{}}

	seen := map[string]bool{}
	done := false

	for len(forwardQueue) > 0 && len(backwardQueue) > 0 && !done {
		nextForward := []Node{}
		for _, node := range forwardQueue {
			for product, data := range recipes {
				for _, inputs := range data.Recipes {
					if containsAll(inputs, node.Element) {
						other := inputs[0]
						if other == node.Element {
							other = inputs[1]
						}
						for range visitedF[other] {
							newPath := copyMap(node.Path)
							newPath[product] = append(newPath[product], inputs)
							visitedF[product] = append(visitedF[product], newPath)
							nextForward = append(nextForward, Node{product, newPath})

							if matches, ok := visitedB[product]; ok {
								for _, bPath := range matches {
									combined := mergePaths(newPath, bPath)
									if allLeavesAreBaseElements(combined, baseElements) {
										key := serializePath(combined)
										if !seen[key] {
											results = append(results, combined)
											seen[key] = true
											if len(results) == maxRecipe {
												done = true
												break
											}
										}
									}
								}
							}
							forwardVisitedCount++
						}
					}
					if done {
						break
					}
				}
				if done {
					break
				}
			}
			if done {
				break
			}
		}
		forwardQueue = nextForward
		if done {
			break
		}

		nextBackward := []Node{}
		for _, node := range backwardQueue {
			data, ok := recipes[node.Element]
			if !ok {
				continue
			}
			for _, inputs := range data.Recipes {
				for _, input := range inputs {
					newPath := copyMap(node.Path)
					newPath[node.Element] = append(newPath[node.Element], inputs)
					visitedB[input] = append(visitedB[input], newPath)
					nextBackward = append(nextBackward, Node{input, newPath})

					if matches, ok := visitedF[input]; ok {
						for _, fPath := range matches {
							combined := mergePaths(fPath, newPath)
							if allLeavesAreBaseElements(combined, baseElements) {
								key := serializePath(combined)
								if !seen[key] {
									results = append(results, combined)
									seen[key] = true
									if len(results) == maxRecipe {
										done = true
										break
									}
								}
							}
						}
					}
					backwardVisitedCount++
					if done {
						break
					}
				}
				if done {
					break
				}
			}
			if done {
				break
			}
		}
		backwardQueue = nextBackward
	}

	duration := time.Since(start).Seconds()
	visitedNode = forwardVisitedCount + backwardVisitedCount
	return results, duration, visitedNode
}
