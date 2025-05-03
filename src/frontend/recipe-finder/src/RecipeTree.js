import React, { useEffect, useState } from 'react';

function RecipeNode({ node }) {
  return (
    <li>
      <strong>{node.element}</strong>
      {node.children && node.children.length > 0 && (
        <ul>
          {node.children.map((child, index) => (
            <RecipeNode key={index} node={child} />
          ))}
        </ul>
      )}
    </li>
  );
}

export default function RecipeTree({ element }) {
  const [tree, setTree] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/recipe?element=${encodeURIComponent(element)}`)
      .then((res) => res.json())
      .then((data) => {
        setTree(data);
        setLoading(false);
      });
  }, [element]);

  if (loading) return <p>Loading recipe tree for "{element}"...</p>;
  if (!tree) return <p>No recipe found for "{element}"</p>;

  return (
    <div>
      <h2>Recipe for {element}</h2>
      <ul>
        <RecipeNode node={tree} />
      </ul>
    </div>
  );
}
