"use client";

import { memo } from "react";
import { Handle, Position } from "@xyflow/react";
import type { ChannelInfo } from "@/stores/mesh";

interface ChannelNodeProps {
  data: {
    channel: ChannelInfo;
    isSelected?: boolean;
  };
}

function ChannelNode({ data }: ChannelNodeProps) {
  const { channel, isSelected } = data;

  return (
    <div
      className={`px-4 py-3 rounded-lg border-2 bg-card shadow-md min-w-[160px] transition-all ${
        isSelected
          ? "border-blue-500 ring-2 ring-blue-500/20"
          : channel.is_archived
          ? "border-gray-300 bg-gray-50"
          : "border-blue-200 hover:border-blue-400"
      }`}
    >
      {/* Multiple handles for connected pods */}
      <Handle
        type="target"
        position={Position.Top}
        className="w-3 h-3 !bg-blue-500"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        className="w-3 h-3 !bg-blue-500"
      />

      {/* Channel Icon and Name */}
      <div className="flex items-center gap-2 mb-2">
        <span className="text-lg text-blue-500">#</span>
        <span className={`font-medium ${channel.is_archived ? "text-gray-400" : "text-foreground"}`}>
          {channel.name}
        </span>
      </div>

      {/* Description */}
      {channel.description && (
        <p className="text-xs text-muted-foreground mb-2 line-clamp-2">
          {channel.description}
        </p>
      )}

      {/* Stats */}
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <div className="flex items-center gap-1">
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          <span>{channel.pod_keys.length}</span>
        </div>
        <div className="flex items-center gap-1">
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
          </svg>
          <span>{channel.message_count}</span>
        </div>
      </div>

      {/* Archived Badge */}
      {channel.is_archived && (
        <div className="mt-2 text-xs text-gray-400 flex items-center gap-1">
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
          </svg>
          Archived
        </div>
      )}
    </div>
  );
}

export default memo(ChannelNode);
