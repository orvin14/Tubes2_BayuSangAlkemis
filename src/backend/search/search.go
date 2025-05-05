package search

// Dummy data untuk testing
// TODO : Set Recipes to the real data
var DummyRecipes = map[string][][]string{
	"Brick":    {{"Mud", "Fire"}, {"Clay", "Dust"}, {"Stone", "Sand"}, {"Lava", "Pressure"}},
	"Clay":     {{"Mud", "Sand"}, {"Water", "Earth"}},
	"Mud":      {{"Water", "Earth"}},
	"Stone":    {{"Lava", "Air"}, {"Earth", "Pressure"}},
	"Sand":     {{"Air", "Stone"}},
	"Lava":     {{"Earth", "Fire"}},
	"Pressure": {{"Air", "Air"}},
	"Dust":     {{"Stone", "Air"}},
}

// DFS 
func DFS(element string, maxRecipe int) map[string][][]string {
	// TODO : Implement DFS algorithm
	return DummyRecipes // Dummy data for testing
}

// BFS
func BFS(element string, maxRecipe int) map[string][][]string {
	// TODO : Implement BFS algorithm
	return DummyRecipes // Dummy data for testing
}
