package search

import (
	"recipe-finder/bfs"
	"recipe-finder/dfs"
	"log"
)

// DFS
func DFS(element string, maxRecipe int) ([]map[string][]string, float64, int) {
	return dfs.SearchDFS(element, maxRecipe)
}

// BFS
func BFS(element string, maxRecipe int) ([]map[string][]string, float64, int) {
	if maxRecipe > 5 {
		initialBatch := 3
		results, duration, nodes := bfs.SearchBFS(element, initialBatch)

		if len(results) < initialBatch {
			return results, duration, nodes
		}

		log.Printf("Found initial %d recipes, searching for %d more...", len(results), maxRecipe-len(results))
		moreResults, moreDuration, moreNodes := bfs.SearchBFS(element, maxRecipe)

		for _, r := range moreResults {
			if !recipeExists(results, r) {
				results = append(results, r)
				if len(results) >= maxRecipe {
					break
				}
			}
		}

		return results, duration + moreDuration, nodes + moreNodes
	}

	return bfs.SearchBFS(element, maxRecipe)
}
func recipeExists(existing []map[string][]string, candidate map[string][]string) bool {
	for _, r := range existing {
		if isSameRecipe(r, candidate) {
			return true
		}
	}
	return false
}

func isSameRecipe(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for keyA, valA := range a {
		valB, ok := b[keyA]
		if !ok {
			return false
		}
		if len(valA) != len(valB) {
			return false
		}
		for i := range valA {
			if valA[i] != valB[i] {
				return false
			}
		}
	}
	return true
}
