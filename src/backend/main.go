package main
// TO RUN THIS FILE, USE THE COMMAND: go run main.go
import (
	"encoding/json"
	"log"
	"net/http"
	"recipe-finder/search"
	"recipe-finder/scrape"
)

type RecipeRequest struct {
	Element   string `json:"element"`
	Algorithm string `json:"algorithm"` // "bfs" or "dfs"
	MaxRecipe int    `json:"maxRecipe"`
}
type RecipeResponse struct {
	Results     []map[string][]string `json:"results"`
	Duration    float64                 `json:"duration"`
	VisitedNode int                     `json:"visitedNode"`
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
func exploreRecipes(element string, algorithm string, maxRecipe int) ([]map[string][]string, float64, int) {
	var result []map[string][]string
	var duration float64
	var visitedNode int
	switch algorithm {
	case "bfs":
		result, duration, visitedNode = search.BFS(element, maxRecipe)
	case "dfs":
		result, duration, visitedNode = search.DFS(element, maxRecipe)
	default:
		return nil,0,0
	}
	return result, duration, visitedNode
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

	result, duration, visitedNode := exploreRecipes(req.Element, req.Algorithm, req.MaxRecipe)
	response := RecipeResponse{
		Results:     result,
		Duration:    duration,
		VisitedNode: visitedNode,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/api/recipe", handleRecipe)
	scrape.ScrapeToJsonComplete()

	log.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
