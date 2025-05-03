import React, { useEffect, useState } from "react";

const PAGE_SIZE = 24;

const ElementList = () => {
  const [elements, setElements] = useState([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedRecipe, setSelectedRecipe] = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetch("/elements.json")
      .then((res) => res.json())
      .then((data) => setElements(data))
      .catch((err) => console.error("Failed to load elements:", err));
  }, []);

  const fetchRecipe = async (elementName) => {
    setLoading(true);
    setSelectedRecipe(null);
  
    try {
      // Dummy data
      const dummyData = {
        name: elementName,
        ingredients: [
          {
            name: "Fire",
            ingredients: [
              { name: "Air", ingredients: [] },
              { name: "Energy", ingredients: [] }
            ]
          },
          {
            name: "Water",
            ingredients: [
              { name: "Earth", ingredients: [] },
              { name: "Rain", ingredients: [] }
            ]
          }
        ]
      };
  
      // Simulasi delay
      await new Promise((res) => setTimeout(res, 500));
  
      setSelectedRecipe({ element: elementName, recipe: dummyData });
    } catch (err) {
      console.error("Failed to load dummy recipe:", err);
      setSelectedRecipe({ element: elementName, recipe: null });
    } finally {
      setLoading(false);
    }
  };

  const renderRecipeTree = (node) => {
    if (!node) return null;
    return (
      <li>
        <span className="font-semibold">{node.name}</span>
        {node.ingredients && node.ingredients.length > 0 && (
          <ul className="ml-6 list-disc">
            {node.ingredients.map((child, idx) => (
              <React.Fragment key={idx}>{renderRecipeTree(child)}</React.Fragment>
            ))}
          </ul>
        )}
      </li>
    );
  };

  const indexOfLast = currentPage * PAGE_SIZE;
  const indexOfFirst = indexOfLast - PAGE_SIZE;
  const currentElements = elements.slice(indexOfFirst, indexOfLast);
  const totalPages = Math.ceil(elements.length / PAGE_SIZE);

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4">Little Alchemy 2 Elements</h1>

      <div className="grid grid-cols-8 gap-4">
        {currentElements.map((el, i) => (
          <div
            key={i}
            className="border p-4 rounded-lg shadow-md text-center hover:scale-105 transition-transform duration-200 cursor-pointer"
            onClick={() => fetchRecipe(el.Name)}
          >
            <img src={el.ImageURL} alt={el.Name} className="mx-auto w-60 h-30" />
            <p className="mt-2 font-semibold">{el.Name}</p>
          </div>
        ))}
      </div>

      <div className="mt-4 flex justify-center space-x-2">
        <button
          onClick={() => setCurrentPage((p) => Math.max(p - 1, 1))}
          disabled={currentPage === 1}
          className="px-3 py-1 bg-gray-200 rounded disabled:opacity-50"
        >
          Prev
        </button>
        <span className="px-3 py-1">{currentPage} / {totalPages}</span>
        <button
          onClick={() => setCurrentPage((p) => Math.min(p + 1, totalPages))}
          disabled={currentPage === totalPages}
          className="px-3 py-1 bg-gray-200 rounded disabled:opacity-50"
        >
          Next
        </button>
      </div>

      {loading && <p className="mt-4 text-blue-600">Loading recipe...</p>}

      {selectedRecipe && (
        <div className="mt-6 p-4 border rounded bg-gray-50 shadow">
          <h2 className="text-xl font-bold mb-2">
            Recipe Tree for: {selectedRecipe.element}
          </h2>
          {selectedRecipe.recipe ? (
            <ul className="ml-4">{renderRecipeTree(selectedRecipe.recipe)}</ul>
          ) : (
            <p className="text-red-600">Recipe not found or failed to load.</p>
          )}
        </div>
      )}
    </div>
  );
};

export default ElementList;
