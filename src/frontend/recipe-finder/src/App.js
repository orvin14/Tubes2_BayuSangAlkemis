import React, { useState } from 'react';
import ElementList from './ElementList';
import RecipeTree from './RecipeTree';

function App() {
  const [selectedElement, setSelectedElement] = useState(null);

  return (
    <div style={{ display: 'flex', gap: '2rem' }}>
      <ElementList onSelect={setSelectedElement} />
      <div>
        {selectedElement && (
          <RecipeTree element={selectedElement} />
        )}
      </div>
    </div>
  );
}

export default App;
