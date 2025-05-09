import React, { useEffect, useState } from 'react';
import Tree from 'react-d3-tree';

// Fungsi untuk menghasilkan pohon resep
function generateRecipeTree(element, recipeMap) {
  if (!recipeMap || !recipeMap[element]) return { name: element };

  const recipes = recipeMap[element];
  if (!recipes || recipes.length === 0) return { name: element };
  return {
    name: element,
    children: recipes.map(child => generateRecipeTree(child, recipeMap)),
  };
}

export default function RecipeTree({ element, maxRecipes, searchMode }) {
  const [trees, setTrees] = useState([]);  
  const [loading, setLoading] = useState(true);  
  const [error, setError] = useState("");  
  const [executionTime, setExecutionTime] = useState(0);  
  const [visitedNodes, setVisitedNodes] = useState(0); 

  useEffect(() => {
    if (!element) return;

    setLoading(true);
    setError("");

    fetch('http://localhost:8080/api/recipe', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        element,
        algorithm: searchMode,
        maxRecipe: maxRecipes,
      }),
    })
      .then(res => {
        if (!res.ok) throw new Error("Failed to fetch recipe");
        return res.json();
      })
      .then(data => {
        if (!data.results || !Array.isArray(data.results)) throw new Error("Invalid data format");
        if (data.results.length === 0) throw new Error("No recipes found");

        const recipeMaps = data.results.slice(0, maxRecipes);
        const recipeTrees = recipeMaps.map((recipeMap, i) => ({
          id: i,
          data: generateRecipeTree(element, recipeMap),
        }));

        setTrees(recipeTrees);
        setExecutionTime(data.duration || 0);
        setVisitedNodes(data.visitedNode || 0);
      })
      .catch(err => setError(err.message))
      .finally(() => setLoading(false));
  }, [element, maxRecipes, searchMode]);

  if (loading) return <p className="text-center mt-4">Loading recipes...</p>;
  if (error) return <p className="text-center text-red-500">{error}</p>;
  if (trees.length === 0) return <p className="text-center mt-4">No recipe found for "{element}"</p>;

  return (
    <div className="flex flex-wrap gap-4 p-4">
      <div className="w-full text-center mb-4">
        <p>Execution Time: {executionTime.toFixed(2)} seconds</p>
        <p>Total Nodes Visited: {visitedNodes}</p>
      </div>
      {trees.map((tree, index) => (
        <div key={index} className="w-full h-[800px] border border-gray-300 rounded bg-white shadow p-2">
          <h2 className="text-center font-semibold mb-2">Recipe {index + 1}</h2>
          <Tree
            data={tree.data}
            zoom={0.5}
            orientation="vertical"
            collapsible={false}
            separation={{ siblings: 1.5, nonSiblings: 2 }}
            translate={{ x: 550, y: 20 }}
          />
        </div>
      ))}
    </div>
  );
}

