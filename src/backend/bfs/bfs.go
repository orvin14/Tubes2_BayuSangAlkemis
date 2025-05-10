package bfs

import (
    "container/list"
    "encoding/json"
    "fmt"
    "log"
    "os"
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
func SearchBFS(element string, maxRecipe int, workerLimit ...int) ([]map[string][]string, float64, int) {
    // Declare helpers to avoid compile errors, even if we don't log their output
    _ = DebugRecipe
    _ = CountUniqueRecipes
    _ = FindMissingRecipe

    // Ensure RecipeData is loaded
    if RecipeData == nil {
        if !readRecipeJson() {
            return nil, 0, 0
        }
    }

    // Set worker limit based on optional parameter or default to 6
    numWorkers := 6
    if len(workerLimit) > 0 && workerLimit[0] > 0 {
        numWorkers = workerLimit[0]
    }

    element = strings.TrimSpace(element)
    nodeCount := 0
    startTime := time.Now()

    // Check if element exists
    recipeDataMutex.RLock()
    targetRecipe, ok := RecipeData[element]
    recipeDataMutex.RUnlock()

    if !ok {
        duration := time.Since(startTime)
        return nil, float64(duration.Seconds()), 0
    }

    var result []map[string][]string

    // Handle base elements (tier 0)
    if targetRecipe.Tier == 0 {
        duration := time.Since(startTime)
        result = append(result, map[string][]string{element: {}})
        return result, float64(duration.Seconds()), 1
    }

    // Main synchronization objects
    var resultMutex sync.Mutex
    var wg sync.WaitGroup
    var doneMutex sync.Mutex
    var isDone bool = false
    done := make(chan bool)
    resultChan := make(chan map[string][]string, 100) // Buffered channel

    // Thread-safe queue for BFS
    queue := list.New()
    var queueMutex sync.Mutex

    // We'll use a lightweight visited tracking just to avoid obvious cycles
    // But we don't want to be too aggressive as it might prevent valid paths
    visitedStates := make(map[string]bool)
    var visitedMutex sync.Mutex

    // Safe close function to avoid closing channels multiple times
    safeClose := func() {
        doneMutex.Lock()
        defer doneMutex.Unlock()
        if !isDone {
            close(done)
            isDone = true
        }
    }

    // Initialize BFS with all recipes for the target element
    recipeDataMutex.RLock()
    initialRecipes := RecipeData[element].Recipes
    recipeDataMutex.RUnlock()

    nodeCount++

    targetTier := targetRecipe.Tier

    // Create initial states for each recipe of the target element
    for _, recipe := range initialRecipes {
        if len(recipe) < 1 {
            // Skip empty recipes
            continue
        }

        // Modified tier check: allow same tier components but prevent cycles
        recipeDataMutex.RLock()
        valid := true
        for _, component := range recipe {
            // Skip self-references completely (direct cycles)
            if component == element {
                valid = false
                break
            }

            // Instead of strict tier check, just ensure we don't have higher tier components
            if compData, exists := RecipeData[component]; exists {
                if compData.Tier >= targetTier {
                    valid = false
                    break
                }
            }
        }
        recipeDataMutex.RUnlock()

        if !valid {
            continue // Skip recipes with invalid tier progression
        }

        // Create initial state with this recipe
        recipeMap := map[string][]string{
            element: recipe,
        }

        // Create list of non-base elements to expand
        toExpand := list.New()
        recipeDataMutex.RLock()
        for _, component := range recipe {
            if compData, exists := RecipeData[component]; exists && compData.Tier > 0 {
                toExpand.PushBack(component)
            }
        }
        recipeDataMutex.RUnlock()

        // Add to BFS queue
        state := map[string]interface{}{
            "recipeMap": recipeMap,
            "toExpand":  toExpand,
        }

        queueMutex.Lock()
        queue.PushBack(state)
        queueMutex.Unlock()
    }

    // Collector goroutine to gather results
    go func() {
        seenRecipeFingerprints := make(map[string]bool)

        for recipeMap := range resultChan {
            // Double-check that the recipe is truly complete
            if isRecipeComplete(recipeMap) {
                // Use the full recipe fingerprint instead of just the top-level components
                fingerprint := createRecipeFingerprint(recipeMap)

                // If we haven't seen this exact recipe before
                resultMutex.Lock()
                if !seenRecipeFingerprints[fingerprint] {
                    seenRecipeFingerprints[fingerprint] = true
                    result = append(result, recipeMap)

                    if maxRecipe > 0 && len(result) >= maxRecipe {
                        safeClose() // Signal workers to stop when we have enough recipes
                    }
                }
                resultMutex.Unlock()
            }
        }
    }()

    // Worker goroutines for BFS processing
    var activeWorkers int32
    var activeMutex sync.Mutex

    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func(workerId int) {
            defer wg.Done()

            // Mark worker as active
            activeMutex.Lock()
            activeWorkers++
            activeMutex.Unlock()

            for {
                // Check if we should terminate
                select {
                case <-done:
                    return
                default:
                    // Continue processing
                }

                // Get next state from queue
                queueMutex.Lock()
                if queue.Len() == 0 {
                    // No more work in queue
                    queueMutex.Unlock()

                    // Update active worker count
                    activeMutex.Lock()
                    activeWorkers--
                    remaining := activeWorkers
                    activeMutex.Unlock()

                    if remaining == 0 {
                        // All workers are idle and queue is empty; we're done
                        safeClose()
                        return
                    }

                    // Wait a bit before checking again or terminating
                    select {
                    case <-done:
                        return
                    case <-time.After(10 * time.Millisecond):
                        // Check if there's new work or if we should terminate
                        activeMutex.Lock()
                        queueMutex.Lock()
                        queueLen := queue.Len()
                        queueMutex.Unlock()

                        if activeWorkers == 0 && queueLen == 0 {
                            // Still no active workers and queue is empty, safe to terminate
                            activeMutex.Unlock()
                            return
                        }

                        // Either new work appeared or some workers are active
                        if queueLen > 0 {
                            // New work appeared, become active again
                            activeWorkers++
                            activeMutex.Unlock()
                            continue
                        }

                        activeMutex.Unlock()
                        time.Sleep(10 * time.Millisecond) // Wait a bit more
                    }
                    continue
                }

                // Get work from queue
                stateElement := queue.Front()
                queue.Remove(stateElement)
                // queueLen := queue.Len()
                queueMutex.Unlock()

                // Make sure we're marked as active
                activeMutex.Lock()
                activeWorkers++
                activeMutex.Unlock()

                // Process current state
                state := stateElement.Value.(map[string]interface{})
                recipeMap := state["recipeMap"].(map[string][]string)
                toExpand := state["toExpand"].(*list.List)

                // Add a brief sleep to reduce CPU pressure
                time.Sleep(1 * time.Millisecond)

                // Recipe is complete if there's nothing left to expand
                if toExpand.Len() == 0 {
                    // Create a copy of the recipe map and send it as a result
                    result := make(map[string][]string)
                    for k, v := range recipeMap {
                        result[k] = v
                    }

                    select {
                    case <-done:
                        return
                    case resultChan <- result:
                        // Result sent successfully
                    }
                    continue
                }

                // Get next element to expand
                elementElement := toExpand.Front()
                toExpand.Remove(elementElement)
                elementToExpand := elementElement.Value.(string)

                // Skip if already expanded
                if _, alreadyExpanded := recipeMap[elementToExpand]; alreadyExpanded {
                    // Put state back in queue with remaining elements
                    newToExpand := list.New()
                    for el := toExpand.Front(); el != nil; el = el.Next() {
                        newToExpand.PushBack(el.Value)
                    }

                    if newToExpand.Len() > 0 {
                        newState := map[string]interface{}{
                            "recipeMap": recipeMap,
                            "toExpand":  newToExpand,
                        }

                        queueMutex.Lock()
                        queue.PushBack(newState)
                        queueMutex.Unlock()
                    } else {
                        // Recipe is complete, send result
                        result := make(map[string][]string)
                        for k, v := range recipeMap {
                            result[k] = v
                        }

                        select {
                        case <-done:
                            return
                        case resultChan <- result:
                            // Result sent successfully
                        }
                    }
                    continue
                }

                // Get recipes for this element
                recipeDataMutex.RLock()
                elementData, exists := RecipeData[elementToExpand]
                recipeDataMutex.RUnlock()

                if !exists {
                    continue
                }

                recipeDataMutex.RLock()
                elementRecipes := elementData.Recipes
                recipeDataMutex.RUnlock()

                // Try each recipe for this element
                recipeAdded := false
                for _, recipe := range elementRecipes {
                    if len(recipe) < 1 {
                        continue // Skip empty recipes
                    }

                    // Check if recipe is valid - modified tier check
                    recipeDataMutex.RLock()
                    valid := true
                    // Create a set of elements in the current path to detect cycles
                    currentPath := make(map[string]bool)
                    for k := range recipeMap {
                        currentPath[k] = true
                    }
                    currentPath[elementToExpand] = true

                    for _, component := range recipe {
                        // Avoid cycles in recipe graph
                        if currentPath[component] {
                            valid = false
                            break
                        }

                        // Check if component exists
                        if _, exists := RecipeData[component]; !exists {
                            valid = false
                            break
                        }

                        // Relaxed tier check: allow same tier but prevent higher tier components
                        if compData, exists := RecipeData[component]; exists {
                            if compData.Tier >= targetTier {
                                valid = false
                                break
                            }
                        }
                    }
                    recipeDataMutex.RUnlock()

                    if valid {
                        // This recipe is valid - create a new state
                        newRecipeMap := make(map[string][]string)
                        for k, v := range recipeMap {
                            newRecipeMap[k] = v
                        }
                        newRecipeMap[elementToExpand] = recipe

                        // Create a new expansion queue with remaining elements
                        newToExpand := list.New()
                        for el := toExpand.Front(); el != nil; el = el.Next() {
                            newToExpand.PushBack(el.Value)
                        }

                        // Add recipe components to expansion queue if needed
                        recipeDataMutex.RLock()
                        for _, component := range recipe {
                            if _, exists := newRecipeMap[component]; !exists {
                                if compData, exists := RecipeData[component]; exists && compData.Tier > 0 {
                                    newToExpand.PushBack(component)
                                }
                            }
                        }
                        recipeDataMutex.RUnlock()

                        // We need a better state key that distinguishes between different paths
                        // Use a combination of the current element and its recipe, plus a hash of the entire recipe map
                        recipeMapKey := ""
                        elementKeys := make([]string, 0, len(newRecipeMap))
                        for k := range newRecipeMap {
                            elementKeys = append(elementKeys, k)
                        }
                        // Sort elementKeys for consistent fingerprinting
                        for i := 0; i < len(elementKeys); i++ {
                            for j := i + 1; j < len(elementKeys); j++ {
                                if elementKeys[i] > elementKeys[j] {
                                    elementKeys[i], elementKeys[j] = elementKeys[j], elementKeys[i]
                                }
                            }
                        }
                        // Create a more unique fingerprint that includes component info
                        for _, k := range elementKeys {
                            components := newRecipeMap[k]
                            // Sort components for consistency
                            sortedComps := make([]string, len(components))
                            copy(sortedComps, components)
                            for i := 0; i < len(sortedComps); i++ {
                                for j := i + 1; j < len(sortedComps); j++ {
                                    if sortedComps[i] > sortedComps[j] {
                                        sortedComps[i], sortedComps[j] = sortedComps[j], sortedComps[i]
                                    }
                                }
                            }
                            recipeMapKey += fmt.Sprintf("%s:[%s]", k, strings.Join(sortedComps, ","))
                        }

                        stateKey := fmt.Sprintf("%s:%v:%s", elementToExpand, recipe, recipeMapKey)

                        // Don't use visited state tracking - we want all possible recipes
                        // This might generate duplicates but we need full coverage
                        visitedMutex.Lock()
                        if !visitedStates[stateKey] {
                            visitedStates[stateKey] = true
                            visitedMutex.Unlock()

                            // Create new state
                            newState := map[string]interface{}{
                                "recipeMap": newRecipeMap,
                                "toExpand":  newToExpand,
                            }

                            queueMutex.Lock()
                            // Improved queue management to prevent memory issues
                            for queue.Len() >= 400 {
                                // If queue is too large, remove the oldest item
                                lastElement := queue.Back()
                                if lastElement != nil {
                                    queue.Remove(lastElement)
                                }
                            }
                            queue.PushBack(newState)
                            recipeAdded = true
                            nodeCount++
                            queueMutex.Unlock()
                        } else {
                            visitedMutex.Unlock()
                        }
                    }
                }

                // If no valid recipes were found for this element, this path is a dead end
                if !recipeAdded {
                    continue
                }
            }
        }(i)
    }

    // Wait for all workers to finish
    wg.Wait()
    safeClose()
    close(resultChan)

    duration := time.Since(startTime)
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