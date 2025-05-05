import React, { useState } from 'react';
import ElementList from './ElementList';
import RecipeTree from './RecipeTree';
import TopBar from './TopBar';
// TO RUN THIS APP: USE COMMAND NPM START
function App() {
  const [selectedElement, setSelectedElement] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [search, setSearch] = useState('');
  const [showMultiple, setShowMultiple] = useState(false);
  const [maxRecipes, setMaxRecipes] = useState(5);
  const [searchMode, setSearchMode] = useState('bfs'); // bfs or dfs

  const handleSelect = (element) => {
    setSelectedElement(element);
    setShowModal(true);
  };

  const closeModal = () => {
    setShowModal(false);
    setSelectedElement(null);
  };

  return (
    <div>
      <TopBar
        search={search}
        onSearchChange={(value) => setSearch(value)}
        showMultiple={showMultiple}
        setShowMultiple={setShowMultiple}
        maxRecipes={maxRecipes}
        setMaxRecipes={setMaxRecipes}
        searchMode={searchMode}
        setSearchMode={setSearchMode}
      />

      <ElementList onSelect={handleSelect} search={search} />

      {showModal && selectedElement && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center z-50">
          <div className="bg-white p-6 rounded-lg max-h-[90vh] overflow-auto relative w-[80vw]">
            <h2 className="mb-4 text-xl font-semibold text-center">
              Recipe for {selectedElement}
            </h2>
            <button
              onClick={closeModal}
              className="absolute top-2 right-2 bg-red-500 text-white rounded-full w-8 h-8 flex items-center justify-center hover:bg-red-600 transition"
            >
              ✕
            </button>
            <RecipeTree
              element={selectedElement}
              maxRecipes={showMultiple ? maxRecipes : 1}
              searchMode={searchMode}
            />
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
