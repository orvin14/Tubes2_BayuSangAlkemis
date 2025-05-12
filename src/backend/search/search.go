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
	return bfs.SearchBFS(element, maxRecipe)
}
