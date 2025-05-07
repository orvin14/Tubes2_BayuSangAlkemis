package search
import (
	"time"
)

// Dummy data untuk testing
// TODO : Set Recipes to the real data
var DummyRecipe1 = map[string][][]string{
	"Brick":    {{"Mud", "Fire"}},
	"Mud":      {{"Water", "Earth"}},
}
var DummyRecipe2 = map[string][][]string{
	"Brick": {{"Clay", "Fire"}},
	"Clay":  {{"Air", "Earth"}},
}
// DFS 
func DFS(element string, maxRecipe int) ([]map[string][][]string, float64, int) {
	var results []map[string][][]string
	start := time.Now()
	visitedNode := 0

	// Contoh 2 resep hasil BFS
	recipe1 := map[string][][]string{
		"Brick": {{"Mud", "Fire"}},
		"Mud":   {{"Water", "Earth"}},
	}
	recipe2 := map[string][][]string{
		"Brick": {{"Clay", "Fire"}},
		"Clay":  {{"Air", "Earth"}},
	}

	results = append(results, recipe1, recipe2)
	duration := time.Since(start).Seconds()
	visitedNode += 3 // Simulasi jumlah node yang dikunjungi
	duration += 1.5
	return results, duration, visitedNode
}

// BFS
func BFS(element string, maxRecipe int) ([]map[string][][]string, float64, int) {
	var results []map[string][][]string
	var visitedNode int
	var duration float64
	start := time.Now()
	visitedNode = 0

	// Contoh 2 resep hasil BFS
	recipe1 := map[string][][]string{
		"Brick": {{"Mud", "Fire"}},
		"Mud":   {{"Water", "Earth"}},
	}
	recipe2 := map[string][][]string{
		"Brick": {{"Clay", "Fire"}},
		"Clay":  {{"Air", "Earth"}},
	}
	if element == "Brick" {
	results = append(results, recipe1, recipe2)
	duration = time.Since(start).Seconds()
	visitedNode += 2 // Simulasi jumlah node yang dikunjungi
	duration += 0.5
	}
	return results, duration, visitedNode
}
