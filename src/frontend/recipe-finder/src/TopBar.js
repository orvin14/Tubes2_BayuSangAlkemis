import React from 'react';

export default function TopBar({
  search,
  onSearchChange,
  showMultiple,
  setShowMultiple,
  maxRecipes,
  setMaxRecipes,
  searchMode,
  setSearchMode,
}) {
  return (
    <div className="bg-[#E9F1FA] p-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <h1 className="text-xl font-bold">Little Alchemy 2</h1>

      <input
        type="text"
        placeholder="Search elements..."
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        className="border rounded px-3 py-1 w-full h-8 md:w-2/3"
      />

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <label className="flex items-center gap-1">
            <input
              type="checkbox"
              checked={showMultiple}
              onChange={() => setShowMultiple((prev) => !prev)}
            />
            Multiple Recipes
          </label>

          {showMultiple && (
            <input
              type="number"
              value={maxRecipes}
              onChange={(e) => setMaxRecipes(Number(e.target.value))}
              className="border rounded px-2 py-1 w-16"
              min={1}
              max={50}
            />
          )}
        </div>

        <div className="flex items-center gap-4">
          <label className="flex items-center gap-1">
            <input
              type="radio"
              name="searchMode"
              value="bfs"
              checked={searchMode === 'bfs'}
              onChange={() => setSearchMode('bfs')}
            />
            BFS
          </label>
          <label className="flex items-center gap-1">
            <input
              type="radio"
              name="searchMode"
              value="dfs"
              checked={searchMode === 'dfs'}
              onChange={() => setSearchMode('dfs')}
            />
            DFS
          </label>
        </div>
      </div>
    </div>
  );
}
