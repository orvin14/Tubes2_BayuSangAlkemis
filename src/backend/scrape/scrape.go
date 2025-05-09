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