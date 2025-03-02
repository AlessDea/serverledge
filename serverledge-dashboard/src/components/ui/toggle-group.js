import React from "react";

export function ToggleGroup({ value, onValueChange, children }) {
  return (
    <div className="flex space-x-2">
      {React.Children.map(children, child =>
        React.cloneElement(child, { isSelected: child.props.value === value, onClick: () => onValueChange(child.props.value) })
      )}
    </div>
  );
}

export function ToggleGroupItem({ value, isSelected, onClick, children }) {
  return (
    <button
      className={`px-4 py-2 border rounded-md ${isSelected ? "bg-blue-500 text-white" : "bg-gray-200"}`}
      onClick={onClick}
    >
      {children}
    </button>
  );
}
