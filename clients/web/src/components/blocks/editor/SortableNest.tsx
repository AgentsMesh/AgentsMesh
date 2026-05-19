"use client";

import React, { createContext, useContext } from "react";
import { DndContext, PointerSensor, useSensor, useSensors } from "@dnd-kit/core";
import type { DragEndEvent } from "@dnd-kit/core";
import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

import { readNestChildren, readRefs } from "@/stores/blockstore";
import { keyBetween } from "@/lib/blockstore/fractionalIndex";
import { updateRefOp } from "@/lib/blockstore/opBuilder";
import { dispatchOps } from "@/stores/blockstoreDispatch";

type DragListeners = Record<string, (e: unknown) => void> | undefined;
const DragHandleContext = createContext<DragListeners>(undefined);
export function useDragHandle(): DragListeners {
  return useContext(DragHandleContext);
}

export interface SortableNestProps {
  workspaceID: string;
  parentID: string;
  childRefIDs: number[];
  renderItem: (refID: number) => React.ReactNode;
}

export function SortableNest({ workspaceID, parentID, childRefIDs, renderItem }: SortableNestProps) {
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 4 } }));
  const itemIDs = childRefIDs.map(String);

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    await reorderNest(workspaceID, parentID, Number(active.id), Number(over.id));
  };

  return (
    <DndContext sensors={sensors} onDragEnd={handleDragEnd}>
      <SortableContext items={itemIDs} strategy={verticalListSortingStrategy}>
        {childRefIDs.map((id) => (
          <SortableItem key={id} id={id}>
            {renderItem(id)}
          </SortableItem>
        ))}
      </SortableContext>
    </DndContext>
  );
}

function SortableItem({ id, children }: { id: number; children: React.ReactNode }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: String(id) });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.35 : 1,
  };
  return (
    <div ref={setNodeRef} style={style} {...attributes}>
      <DragHandleContext.Provider value={listeners as DragListeners}>
        {children}
      </DragHandleContext.Provider>
    </div>
  );
}

async function reorderNest(
  workspaceID: string,
  parentID: string,
  movedRefID: number,
  targetRefID: number,
) {
  const nestChildren = readNestChildren();
  const refs = readRefs();
  const refIDs: number[] = nestChildren[parentID] ?? [];
  const fromIdx = refIDs.indexOf(movedRefID);
  const toIdx = refIDs.indexOf(targetRefID);
  if (fromIdx < 0 || toIdx < 0) return;

  const without = refIDs.filter((id: number) => id !== movedRefID);
  const insertAt = without.indexOf(targetRefID) + (fromIdx < toIdx ? 1 : 0);
  const beforeKey = insertAt > 0 ? refs[without[insertAt - 1]]?.order_key ?? null : null;
  const afterKey = insertAt < without.length ? refs[without[insertAt]]?.order_key ?? null : null;
  const newKey = keyBetween(beforeKey, afterKey);

  await dispatchOps(workspaceID, [
    updateRefOp({ ref_id: movedRefID, order_key: newKey }),
  ]);
}
