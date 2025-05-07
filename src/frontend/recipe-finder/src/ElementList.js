import React, { useEffect, useState } from 'react';

const PAGE_SIZE = 32;

export default function ElementList({ onSelect, search }) {
  const [elements, setElements] = useState([]);
  const [currentPage, setCurrentPage] = useState(1);

  useEffect(() => {
    fetch('/elements.json')
      .then((res) => res.json())
      .then((data) => setElements(data))
      .catch((err) => console.error('Failed to load elements:', err));
  }, []);
  useEffect(() => {
    setCurrentPage(1); 
  }, [search]);

  const filteredElements = elements.filter((el) =>
    el.Name.toLowerCase().includes(search.toLowerCase())
  );

  const totalPages = Math.ceil(filteredElements.length / PAGE_SIZE);
  const indexOfLast = currentPage * PAGE_SIZE;
  const indexOfFirst = indexOfLast - PAGE_SIZE;
  const currentElements = filteredElements.slice(indexOfFirst, indexOfLast);

  return (
    <div className="p-4">
      <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-6 xl:grid-cols-8 gap-4">
        {currentElements.map((el, i) => (
          <div
            key={i}
            className="border p-4 rounded-lg shadow-md text-center hover:scale-105 transition-transform duration-200 cursor-pointer"
            onClick={() => onSelect(el.Name)}
          >
            <img
              src={el.ImageURL}
              alt={el.Name}
              className="mx-auto w-20 h-20 object-contain"
            />
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
        <span className="px-3 py-1">
          {currentPage} / {totalPages}
        </span>
        <button
          onClick={() => setCurrentPage((p) => Math.min(p + 1, totalPages))}
          disabled={currentPage === totalPages}
          className="px-3 py-1 bg-gray-200 rounded disabled:opacity-50"
        >
          Next
        </button>
      </div>
    </div>
  );
}
