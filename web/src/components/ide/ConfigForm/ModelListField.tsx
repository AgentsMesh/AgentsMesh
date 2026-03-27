"use client";

import React, { memo, useState } from "react";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVertical, X, Plus } from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * Sortable item for model list
 */
function SortableModelItem({
  model,
  onRemove,
}: {
  model: string;
  onRemove: () => void;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: model });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        "flex items-center gap-2 px-3 py-2 bg-background border border-border rounded-md",
        isDragging && "opacity-50 z-50 shadow-md"
      )}
    >
      <button
        type="button"
        {...attributes}
        {...listeners}
        className="cursor-grab active:cursor-grabbing text-muted-foreground hover:text-foreground"
        aria-label="Drag to reorder"
      >
        <GripVertical className="h-4 w-4" />
      </button>
      <span className="flex-1 text-sm truncate">{model}</span>
      <button
        type="button"
        onClick={onRemove}
        className="text-muted-foreground hover:text-destructive"
        aria-label={`Remove ${model}`}
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}

/**
 * Model list field renderer (drag-and-drop list of models)
 */
export const ModelListField = memo(function ModelListField({
  fieldKey,
  label,
  description,
  value,
  onChange,
}: {
  fieldKey: string;
  label: string;
  description: string;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const [newModel, setNewModel] = useState("");

  const models = Array.isArray(value) ? value.filter((m): m is string => typeof m === "string") : [];

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = models.indexOf(String(active.id));
    const newIndex = models.indexOf(String(over.id));
    if (oldIndex === -1 || newIndex === -1) return;

    const newModels = [...models];
    newModels.splice(oldIndex, 1);
    newModels.splice(newIndex, 0, models[oldIndex]);
    onChange(newModels);
  };

  const handleAddModel = () => {
    const trimmed = newModel.trim();
    if (!trimmed || models.includes(trimmed)) {
      setNewModel("");
      return;
    }
    onChange([...models, trimmed]);
    setNewModel("");
  };

  const handleRemoveModel = (modelToRemove: string) => {
    onChange(models.filter((m) => m !== modelToRemove));
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleAddModel();
    }
  };

  return (
    <div>
      <label htmlFor={fieldKey} className="block text-sm font-medium mb-1">
        {label}
      </label>

      <div className="flex gap-2 mb-2">
        <input
          type="text"
          id={fieldKey}
          value={newModel}
          onChange={(e) => setNewModel(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="provider/model (e.g., anthropic/claude-sonnet-4)"
          className="flex-1 px-3 py-2 text-sm border border-border rounded-md bg-background"
        />
        <button
          type="button"
          onClick={handleAddModel}
          disabled={!newModel.trim() || models.includes(newModel.trim())}
          className="px-3 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Plus className="h-4 w-4" />
        </button>
      </div>

      {models.length > 0 && (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext items={models} strategy={verticalListSortingStrategy}>
            <div className="space-y-2">
              {models.map((model) => (
                <SortableModelItem
                  key={model}
                  model={model}
                  onRemove={() => handleRemoveModel(model)}
                />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      )}

      {description && (
        <p id={`${fieldKey}-desc`} className="text-xs text-muted-foreground mt-2">
          {description}
        </p>
      )}
    </div>
  );
});
