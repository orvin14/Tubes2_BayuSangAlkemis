package scrape

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ElementData struct {
	Tier    int        `json:"tier"`
	Recipes [][]string `json:"recipes"`
}

func CompleteScrapeRecipes() map[string]ElementData {
	elements := make(map[string]ElementData)

	// Request HTML page
	res, err := http.Get("https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load HTML doc
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("h3").Each(func(i int, elementTier *goquery.Selection) {
		spanheadline := elementTier.Find("span.mw-headline")
		if spanheadline.Length() > 0 && spanheadline.Text() != "Special element" {
			elementTierText := spanheadline.Text()
			// Extract angka tier
			tierStr := strings.TrimSuffix(strings.TrimPrefix(elementTierText, "Tier "), " elements")
			tier, err := strconv.Atoi(tierStr)
			if err != nil {
				tier = 0 // Parsing gagal -> elemen dasar -> tier 0
			}

			elementTier.NextAllFiltered("table.list-table").First().Each(func(j int, tableSelection *goquery.Selection) {
				tableSelection.Find("tr").Each(func(j int, rowSelection *goquery.Selection) {
					// Skip header rows
					if rowSelection.Find("th").Length() > 0 {
						return
					}

					// Ambil semua kolom dari row
					columns := rowSelection.Find("td")

					// Kolom 1 nama elemen
					// Perlu last karena nama elemen ada di dalam <a> yang terakhir
					elementName := columns.First().Find("a").Last().Text()
					if elementName == "" || elementName == "Time" || elementName == "Ruins" {
						return
					}

					var validRecipes [][]string

					// Kolom 2 resep (<td> kedua)
					columns.Last().Find("li").Each(func(k int, recipeSelection *goquery.Selection) {
						var ingredients []string

						hasTimeOrRuins := false

						recipeSelection.Find("a").Each(func(l int, ingredientSelection *goquery.Selection) {
							ingredientName := ingredientSelection.Text()
							if ingredientName != "" {
								ingredients = append(ingredients, ingredientName)

								if ingredientName == "Time" || ingredientName == "Ruins" {
									hasTimeOrRuins = true
								}
							}
						})

						if len(ingredients) > 0 && !hasTimeOrRuins {
							validRecipes = append(validRecipes, ingredients)
						}
					})

					// Store element with its tier and recipes
					elements[elementName] = ElementData{
						Tier:    tier,
						Recipes: validRecipes,
					}
				})
			})
		}
	})

	return elements
}

func ScrapeToJsonComplete() {
	elements := CompleteScrapeRecipes()
	elements = CleanRecipes(elements)

	jsonData, err := json.MarshalIndent(elements, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}
	if _, err := os.Stat("./data"); os.IsNotExist(err) {
		err = os.Mkdir("./data", 0755)
		if err != nil {
			log.Fatalf("Error creating directory: %v", err)
		}
	}
	err = os.WriteFile("./data/recipes_complete.json", jsonData, 0644)
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	fmt.Println("Successfully exported recipes to data/recipes_complete.json")
}

func CleanRecipes(itemsMap map[string]ElementData) map[string]ElementData {
	// Create a new map for the cleaned data
	cleanedMap := make(map[string]ElementData)

	// First, copy all items to the cleaned map
	for itemName, itemData := range itemsMap {
		cleanedMap[itemName] = itemData
	}

	// Then, clean up each item's recipes
	for itemName, itemData := range cleanedMap {
		// Skip if recipes is nil
		if itemData.Recipes == nil {
			continue
		}

		var validRecipes [][]string

		// Check each recipe
		for _, recipe := range itemData.Recipes {
			valid := true

			// Check if all ingredients in this recipe exist as keys
			for _, ingredient := range recipe {
				if _, exists := itemsMap[ingredient]; !exists {
					valid = false
					fmt.Printf("Removing recipe for %s: ingredient %s doesn't exist\n", itemName, ingredient)
					break
				}
			}

			// If all ingredients are valid, keep this recipe
			if valid {
				validRecipes = append(validRecipes, recipe)
			}
		}

		// Update the recipes for this item
		itemData.Recipes = validRecipes
		cleanedMap[itemName] = itemData
	}
	for itemName := range cleanedMap {
		cleanedMap[itemName] = filterRecipes(cleanedMap, itemName)
	}
	cleanedMap = removeAllInvalidRecipes(cleanedMap)
	return cleanedMap
}
func filterRecipes(itemsMap map[string]ElementData, name string) ElementData {
	element := itemsMap[name]
	var filtered [][]string

	for _, recipe := range element.Recipes {
		valid := true
		for _, ing := range recipe {
			ingData, ok := itemsMap[ing]
			if !ok {
				fmt.Printf("Bahan tidak ditemukan: %s\n", ing)
				valid = false
				break
			}
			if ingData.Tier >= element.Tier {
				fmt.Printf("Bahan %s dengan Tier %d tidak valid untuk %s (Tier %d)\n", ing, ingData.Tier, name, element.Tier)
				valid = false
				break
			}
		}

		if valid {
			filtered = append(filtered, recipe)
		}
	}

	element.Recipes = filtered
	return element
}
func removeAllInvalidRecipes(itemsMap map[string]ElementData) map[string]ElementData {
	// Set untuk menyimpan elemen-elemen yang invalid
	invalidElements := make(map[string]bool)

	// Inisialisasi: elemen yang Recipes-nya nil atau kosong dianggap invalid
	for name, el := range itemsMap {
		if (el.Recipes == nil || len(el.Recipes) == 0) && el.Tier > 0 {
			invalidElements[name] = true
		}
	}

	changed := true
	for changed {
		changed = false

		for name, el := range itemsMap {
			var filtered [][]string
			for _, recipe := range el.Recipes {
				invalid := false
				for _, ing := range recipe {
					if invalidElements[ing] {
						invalid = true
						break
					}
				}
				if !invalid {
					filtered = append(filtered, recipe)
				}
			}

			if len(filtered) != len(el.Recipes) {
				el.Recipes = filtered
				itemsMap[name] = el
				if len(filtered) == 0 && !invalidElements[name] {
					invalidElements[name] = true
					changed = true
				}
			}
		}
	}

	return itemsMap
}
