// import React, { useEffect, useState } from 'react';
// import Tree from 'react-d3-tree';

// const dummyRecipes = {
//   Brick: [
//     ['Clay', 'Stone'],
//     ['Mud', 'Fire'],
//     ['Dust', 'Air']
//   ],
//   Clay: [['Mud', 'Sand'], ['Water', 'Earth']],
//   Mud: [['Water', 'Earth']],
//   Stone: [['Lava', 'Air'], ['Earth', 'Pressure']],
//   Sand: [['Air', 'Stone']],
//   Lava: [['Earth', 'Fire']],
//   Pressure: [['Air', 'Air']],
//   Dust: [['Stone', 'Air']]
// };

// // Fungsi menghasilkan semua jalur pembuatan suatu elemen
// function generateAllRecipes(element) {
//   const recipes = dummyRecipes[element];
//   if (!recipes || recipes.length === 0) return [[{ name: element }]];

//   const allPaths = [];

//   for (const recipe of recipes) {
//     const subPaths = recipe.map(child => generateAllRecipes(child));
//     const combinations = cartesianProduct(subPaths);

//     for (const combo of combinations) {
//       allPaths.push([
//         {
//           name: element,
//           children: combo.map(path => path[0])
//         }
//       ]);
//     }
//   }

//   return allPaths;
// }

// // Kombinasi kartesian
// function cartesianProduct(arrays) {
//   return arrays.reduce((acc, curr) => {
//     const res = [];
//     for (const a of acc) {
//       for (const b of curr) {
//         res.push([...a, b]);
//       }
//     }
//     return res;
//   }, [[]]);
// }

// // Membangun satu pohon besar dari semua path
// function mergePathsIntoSingleTree(paths) {
//   return {
//     name: paths[0][0].name,
//     children: paths.map(path => path[0]) // Setiap path dimasukkan sebagai anak 
//   };
// }

// export default function RecipeTree({ element, maxRecipes = 5 }) {
//   const [treeData, setTreeData] = useState(null);

//   useEffect(() => {
//     const allPaths = generateAllRecipes(element).slice(0, maxRecipes);
//     const mergedTree = mergePathsIntoSingleTree(allPaths);
//     setTreeData([mergedTree]);
//   }, [element, maxRecipes]);

//   if (!treeData) return <p className="text-center mt-4">No recipe found for "{element}"</p>;

//   return (
//     <div className="w-full h-[800px] border border-gray-300 rounded p-4 bg-white shadow">
//       <Tree
//         data={treeData}
//         zoom={0.5}
//         orientation="vertical"
//         collapsible={false}
//         separation={{ siblings: 1.5, nonSiblings: 2 }}
//         translate={{ x: 580, y: 50 }}
//       />
//     </div>
//   );
// }
import React, { useEffect, useState } from 'react';
import Tree from 'react-d3-tree';

// Generate all combinations of making the element (recursive tree per recipe)
function generateAllRecipes(element, recipeMap) {
  const recipes = recipeMap[element];
  if (!recipes || recipes.length === 0) return [[{ name: element }]];

  const allPaths = [];

  for (const recipe of recipes) {
    const subPaths = recipe.map(child => generateAllRecipes(child, recipeMap));
    const combinations = cartesianProduct(subPaths);

    for (const combo of combinations) {
      allPaths.push([
        {
          name: element,
          children: combo.map(path => path[0])
        }
      ]);
    }
  }

  return allPaths;
}

// Utility: Cartesian product
function cartesianProduct(arrays) {
  return arrays.reduce((acc, curr) => {
    const res = [];
    for (const a of acc) {
      for (const b of curr) {
        res.push([...a, b]);
      }
    }
    return res;
  }, [[]]);
}

export default function RecipeTree({ element, maxRecipes = 5,mode="bfs" }) {
  const [trees, setTrees] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!element) return;

    setLoading(true);
    setError("");

    // Meminta data dari backend
    fetch('http://localhost:8080/api/recipe', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              element,
              algorithm: mode,
              maxRecipe: maxRecipes,
        }),
     
    })
               .then(res => {
            if (!res.ok) throw new Error("Failed to fetch recipe");
            return res.json();
          })
          .then(data => {
            // Mengambil data resep dan membangunnya menjadi pohon
            const allPaths = generateAllRecipes(element, data).slice(0, maxRecipes);
            const trees = allPaths.map((path, i) => ({
              id: i,
              data: [path[0]]
            }));
            setTrees(trees);
          })
          .catch(err => setError(err.message))
          .finally(() => setLoading(false));
        
  }, [element, maxRecipes, mode]);

  if (loading) return <p className="text-center mt-4">Loading recipes...</p>;
  if (error) return <p className="text-center text-red-500">{error}</p>;
  if (trees.length === 0) return <p className="text-center mt-4">No recipe found for "{element}"</p>;

  return (
    <div className="flex flex-wrap gap-4 p-4">
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



