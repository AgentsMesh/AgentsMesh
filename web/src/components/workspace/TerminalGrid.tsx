"use client";

import React, { useMemo } from "react";
import { Group, Panel, Separator } from "react-resizable-panels";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspace";
import { TerminalPane } from "./TerminalPane";
import { Terminal as TerminalIcon, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";

interface TerminalGridProps {
  onPopout?: (paneId: string) => void;
  onAddNew?: () => void;
  className?: string;
}

/**
 * 空槽位占位组件
 */
function EmptyPaneSlot({ onAddNew }: { onAddNew?: () => void }) {
  return (
    <div className="flex items-center justify-center h-full bg-terminal-bg-secondary rounded-lg border border-dashed border-terminal-border">
      {onAddNew && (
        <Button
          variant="ghost"
          className="text-terminal-text-muted hover:text-terminal-text hover:bg-terminal-bg-active"
          onClick={onAddNew}
        >
          <Plus className="w-5 h-5 mr-2" />
          Add Terminal
        </Button>
      )}
    </div>
  );
}

/**
 * VS Code 风格的拖拽条 - 默认隐藏，hover 时高亮
 */
function ResizeHandle({ direction }: { direction: "horizontal" | "vertical" }) {
  const isHorizontal = direction === "horizontal";

  return (
    <Separator
      className={cn(
        "group relative flex items-center justify-center bg-transparent transition-colors",
        isHorizontal
          ? "w-1 cursor-col-resize hover:bg-primary"
          : "h-1 cursor-row-resize hover:bg-primary"
      )}
    >
      {/* 扩大点击区域 */}
      <div
        className={cn(
          "absolute z-10",
          isHorizontal ? "w-3 h-full -left-1" : "h-3 w-full -top-1"
        )}
      />
    </Separator>
  );
}

export function TerminalGrid({ onPopout, onAddNew, className }: TerminalGridProps) {
  const panes = useWorkspaceStore((s) => s.panes);
  const activePane = useWorkspaceStore((s) => s.activePane);
  const gridLayout = useWorkspaceStore((s) => s.gridLayout);
  const removePane = useWorkspaceStore((s) => s.removePane);

  // 根据布局计算可见的 panes
  const visiblePanes = useMemo(() => {
    const maxVisible = gridLayout.rows * gridLayout.cols;

    // 确保 activePane 在可见范围内
    if (activePane) {
      const activeIndex = panes.findIndex((p) => p.id === activePane);
      if (activeIndex !== -1) {
        const startIndex = Math.max(0, activeIndex - Math.floor(maxVisible / 2));
        return panes.slice(startIndex, startIndex + maxVisible);
      }
    }

    return panes.slice(0, maxVisible);
  }, [panes, activePane, gridLayout]);

  // 渲染单个终端面板或空槽位
  const renderPane = (index: number) => {
    const pane = visiblePanes[index];
    if (!pane) {
      return <EmptyPaneSlot onAddNew={onAddNew} />;
    }
    return (
      <TerminalPane
        key={pane.id}
        paneId={pane.id}
        podKey={pane.podKey}
        isActive={pane.id === activePane}
        onClose={() => removePane(pane.id)}
        onPopout={onPopout ? () => onPopout(pane.id) : undefined}
        showHeader={true}
      />
    );
  };

  // 空状态
  if (panes.length === 0) {
    return (
      <div className={cn("flex-1 flex items-center justify-center bg-terminal-bg", className)}>
        <div className="text-center">
          <TerminalIcon className="w-16 h-16 mx-auto mb-4 text-terminal-border" />
          <h3 className="text-lg font-medium text-terminal-text mb-2">No terminals open</h3>
          <p className="text-sm text-terminal-text-muted mb-4">
            Open a pod to start a terminal session
          </p>
          {onAddNew && (
            <Button
              onClick={onAddNew}
              className="bg-primary hover:bg-primary/90"
            >
              <Plus className="w-4 h-4 mr-2" />
              Open Terminal
            </Button>
          )}
        </div>
      </div>
    );
  }

  // 1x1 布局 - 单窗口
  if (gridLayout.type === "1x1") {
    return (
      <div className={cn("flex-1 p-1 bg-terminal-bg", className)}>
        {renderPane(0)}
      </div>
    );
  }

  // 1x2 布局 - 两列
  // key 确保布局切换时重置面板状态（符合用户选择的"切换布局时重置为均分"）
  if (gridLayout.type === "1x2") {
    return (
      <Group
        key="layout-1x2"
        orientation="horizontal"
        className={cn("flex-1 p-1 bg-terminal-bg", className)}
      >
        <Panel defaultSize={50} minSize={20}>
          {renderPane(0)}
        </Panel>
        <ResizeHandle direction="horizontal" />
        <Panel defaultSize={50} minSize={20}>
          {renderPane(1)}
        </Panel>
      </Group>
    );
  }

  // 2x1 布局 - 两行
  if (gridLayout.type === "2x1") {
    return (
      <Group
        key="layout-2x1"
        orientation="vertical"
        className={cn("flex-1 p-1 bg-terminal-bg", className)}
      >
        <Panel defaultSize={50} minSize={20}>
          {renderPane(0)}
        </Panel>
        <ResizeHandle direction="vertical" />
        <Panel defaultSize={50} minSize={20}>
          {renderPane(1)}
        </Panel>
      </Group>
    );
  }

  // 2x2 布局 - 四宫格
  if (gridLayout.type === "2x2") {
    return (
      <Group
        key="layout-2x2"
        orientation="vertical"
        className={cn("flex-1 p-1 bg-terminal-bg", className)}
      >
        <Panel defaultSize={50} minSize={20}>
          <Group orientation="horizontal" className="h-full">
            <Panel defaultSize={50} minSize={20}>
              {renderPane(0)}
            </Panel>
            <ResizeHandle direction="horizontal" />
            <Panel defaultSize={50} minSize={20}>
              {renderPane(1)}
            </Panel>
          </Group>
        </Panel>
        <ResizeHandle direction="vertical" />
        <Panel defaultSize={50} minSize={20}>
          <Group orientation="horizontal" className="h-full">
            <Panel defaultSize={50} minSize={20}>
              {renderPane(2)}
            </Panel>
            <ResizeHandle direction="horizontal" />
            <Panel defaultSize={50} minSize={20}>
              {renderPane(3)}
            </Panel>
          </Group>
        </Panel>
      </Group>
    );
  }

  // Fallback: 使用 1x1 布局
  return (
    <div className={cn("flex-1 p-1 bg-terminal-bg", className)}>
      {renderPane(0)}
    </div>
  );
}

export default TerminalGrid;
