package main
// TO RUN THIS FILE, USE THE COMMAND: go run main.go
import (
	"encoding/json"
	"log"
	"net/http"
	"recipe-finder/search"
)

type RecipeRequest struct {
	Element   string `json:"element"`
	Algorithm string `json:"algorithm"` // "bfs" or "dfs"
	MaxRecipe int    `json:"maxRecipe"`
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
func exploreRecipes(element string, algorithm string, maxRecipe int, currentDepth int) map[string][][]string {
	if currentDepth >= maxRecipe {
		return nil
	}
	var result map[string][][]string
	switch algorithm {
	case "bfs":
		result = search.BFS(element, maxRecipe)
	case "dfs":
		result = search.DFS(element, maxRecipe)
	default:
		return nil
	}

	return result
}

func handleRecipe(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result := exploreRecipes(req.Element, req.Algorithm, req.MaxRecipe, 0)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func main() {
	http.HandleFunc("/api/recipe", handleRecipe)

	log.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
