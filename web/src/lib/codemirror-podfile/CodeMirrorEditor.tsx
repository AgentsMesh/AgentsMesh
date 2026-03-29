"use client";

/**
 * Generic CodeMirror 6 React wrapper.
 * Manages EditorView lifecycle and syncs value/extensions with React state.
 * Uses Compartment for dynamic extension reconfiguration.
 */
import React, { useRef, useEffect } from "react";
import { EditorView, type ViewUpdate } from "@codemirror/view";
import { EditorState, Compartment, type Extension } from "@codemirror/state";

interface CodeMirrorEditorProps {
  value: string;
  onChange: (value: string) => void;
  extensions?: Extension[];
  className?: string;
}

export function CodeMirrorEditor({
  value,
  onChange,
  extensions = [],
  className,
}: CodeMirrorEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const compartmentRef = useRef<Compartment>(new Compartment());
  const onChangeRef = useRef(onChange);
  // Track whether the update came from the editor itself
  const isLocalUpdate = useRef(false);

  // Keep onChange ref up to date without re-creating editor
  onChangeRef.current = onChange;

  // Create editor on mount
  useEffect(() => {
    if (!containerRef.current) return;

    const compartment = compartmentRef.current;

    const updateListener = EditorView.updateListener.of((update: ViewUpdate) => {
      if (update.docChanged) {
        isLocalUpdate.current = true;
        onChangeRef.current(update.state.doc.toString());
      }
    });

    const state = EditorState.create({
      doc: value,
      extensions: [updateListener, compartment.of(extensions)],
    });

    const view = new EditorView({ state, parent: containerRef.current });
    viewRef.current = view;

    return () => {
      view.destroy();
      viewRef.current = null;
    };
    // Only run on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync external value changes (skip editor-originated changes)
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;

    if (isLocalUpdate.current) {
      isLocalUpdate.current = false;
      return;
    }

    const currentDoc = view.state.doc.toString();
    if (currentDoc !== value) {
      view.dispatch({
        changes: { from: 0, to: currentDoc.length, insert: value },
      });
    }
  }, [value]);

  // Reconfigure extensions via Compartment when they change
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    view.dispatch({
      effects: compartmentRef.current.reconfigure(extensions),
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [extensions]);

  return <div ref={containerRef} className={className} />;
}
