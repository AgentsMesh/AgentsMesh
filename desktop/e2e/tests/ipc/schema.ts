// AUTO-GENERATED — regenerate: pnpm --filter desktop e2e:gen
export interface IpcMethodSchema {
  name: string;
  group: string;
  params: Array<{ name: string; type: string }>;
  returnType: string;
}

export const ipcSchema: IpcMethodSchema[] = [
  {
    "name": "agent_list_agents",
    "group": "agent",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "agent_get_config_schema",
    "group": "agent",
    "params": [
      {
        "name": "agent_slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "agent_list_user_configs",
    "group": "agent",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "agent_get_user_config",
    "group": "agent",
    "params": [
      {
        "name": "agent_slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "agent_set_user_config",
    "group": "agent",
    "params": [
      {
        "name": "agent_slug",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "agent_delete_user_config",
    "group": "agent",
    "params": [
      {
        "name": "agent_slug",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "agent_get_agentpod_settings",
    "group": "agent",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "agent_update_agentpod_settings",
    "group": "agent",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "agent_list_providers",
    "group": "agent",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "agent_create_provider",
    "group": "agent",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "agent_update_provider",
    "group": "agent",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "agent_delete_provider",
    "group": "agent",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "agent_set_default_provider",
    "group": "agent",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "apikey_list",
    "group": "apikey",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "apikey_get",
    "group": "apikey",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "apikey_create",
    "group": "apikey",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "apikey_update",
    "group": "apikey",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "apikey_delete",
    "group": "apikey",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "apikey_revoke",
    "group": "apikey",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "auth_api_register",
    "group": "auth_api",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "auth_api_verify_email",
    "group": "auth_api",
    "params": [
      {
        "name": "token",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "auth_api_resend_verification",
    "group": "auth_api",
    "params": [
      {
        "name": "email",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "auth_api_forgot_password",
    "group": "auth_api",
    "params": [
      {
        "name": "email",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "auth_api_reset_password",
    "group": "auth_api",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "autopilot_controllers_json",
    "group": "autopilot",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "autopilot_current_controller_json",
    "group": "autopilot",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "autopilot_get_controller_by_pod_key_json",
    "group": "autopilot",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "autopilot_get_iterations_json",
    "group": "autopilot",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "autopilot_get_thinking_json",
    "group": "autopilot",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "autopilot_get_thinking_history_json",
    "group": "autopilot",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "autopilot_set_controllers",
    "group": "autopilot",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_set_current_controller",
    "group": "autopilot",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_add_controller",
    "group": "autopilot",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_update_controller",
    "group": "autopilot",
    "params": [
      {
        "name": "key",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_remove_controller",
    "group": "autopilot",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_set_iterations",
    "group": "autopilot",
    "params": [
      {
        "name": "key",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_add_iteration",
    "group": "autopilot",
    "params": [
      {
        "name": "key",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_update_thinking",
    "group": "autopilot",
    "params": [
      {
        "name": "key",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_fetch_controllers",
    "group": "autopilot_api",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "autopilot_fetch_controller",
    "group": "autopilot_api",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "autopilot_create_controller",
    "group": "autopilot_api",
    "params": [
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "autopilot_pause_controller",
    "group": "autopilot_api",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_resume_controller",
    "group": "autopilot_api",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_stop_controller",
    "group": "autopilot_api",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_approve_controller",
    "group": "autopilot_api",
    "params": [
      {
        "name": "key",
        "type": "String"
      },
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_takeover_controller",
    "group": "autopilot_api",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_handback_controller",
    "group": "autopilot_api",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "autopilot_fetch_iterations",
    "group": "autopilot_api",
    "params": [
      {
        "name": "key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_get_overview",
    "group": "billing",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "billing_get_subscription",
    "group": "billing",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "billing_create_subscription",
    "group": "billing",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_cancel_subscription",
    "group": "billing",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "billing_update_subscription",
    "group": "billing",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_list_plans",
    "group": "billing",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "billing_get_usage",
    "group": "billing",
    "params": [
      {
        "name": "usage_type",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_check_quota",
    "group": "billing",
    "params": [
      {
        "name": "resource",
        "type": "String"
      },
      {
        "name": "amount",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_create_checkout",
    "group": "billing",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_get_checkout_status",
    "group": "billing",
    "params": [
      {
        "name": "order_no",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_request_cancel",
    "group": "billing",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_reactivate",
    "group": "billing",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "billing_upgrade",
    "group": "billing",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_change_cycle",
    "group": "billing",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_update_auto_renew",
    "group": "billing",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_get_seat_usage",
    "group": "billing",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "billing_purchase_seats",
    "group": "billing",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_list_invoices",
    "group": "billing",
    "params": [
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_get_customer_portal",
    "group": "billing",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "billing_get_deployment_info",
    "group": "billing",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "billing_get_public_pricing",
    "group": "billing",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "billing_get_public_deployment_info",
    "group": "billing",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "binding_request_binding",
    "group": "binding",
    "params": [
      {
        "name": "json",
        "type": "String"
      },
      {
        "name": "pod_key",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "binding_accept_binding",
    "group": "binding",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "binding_reject_binding",
    "group": "binding",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "binding_request_scopes",
    "group": "binding",
    "params": [
      {
        "name": "binding_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "binding_approve_scopes",
    "group": "binding",
    "params": [
      {
        "name": "binding_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "binding_unbind",
    "group": "binding",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "binding_list_bindings",
    "group": "binding",
    "params": [
      {
        "name": "status",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "binding_get_pending_bindings",
    "group": "binding",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "binding_get_bound_pods",
    "group": "binding",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "binding_check_binding",
    "group": "binding",
    "params": [
      {
        "name": "target_pod",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_channels_json",
    "group": "channel",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "channel_current_channel_json",
    "group": "channel",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "channel_get_channel_json",
    "group": "channel",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_filter_channels_json",
    "group": "channel",
    "params": [
      {
        "name": "query",
        "type": "String"
      },
      {
        "name": "include_archived",
        "type": "bool"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_get_messages_json",
    "group": "channel",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_get_unread_count",
    "group": "channel",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      }
    ],
    "returnType": "u32"
  },
  {
    "name": "channel_total_unread_count",
    "group": "channel",
    "params": [],
    "returnType": "u32"
  },
  {
    "name": "channel_unread_counts_json",
    "group": "channel",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "channel_get_mention_count",
    "group": "channel",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      }
    ],
    "returnType": "u32"
  },
  {
    "name": "channel_total_mention_count",
    "group": "channel",
    "params": [],
    "returnType": "u32"
  },
  {
    "name": "channel_mention_counts_json",
    "group": "channel",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "channel_sorted_channel_ids_json",
    "group": "channel",
    "params": [
      {
        "name": "mode",
        "type": "String"
      },
      {
        "name": "include_archived",
        "type": "bool"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_get_last_message_json",
    "group": "channel",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_set_channels",
    "group": "channel",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_set_current_channel",
    "group": "channel",
    "params": [
      {
        "name": "id",
        "type": "Option<i64>"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_select_channel",
    "group": "channel",
    "params": [
      {
        "name": "id",
        "type": "Option<i64>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_add_channel_local",
    "group": "channel",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_update_channel_local",
    "group": "channel",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_remove_channel_local",
    "group": "channel",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_fetch_channels",
    "group": "channel_api",
    "params": [
      {
        "name": "include_archived",
        "type": "Option<bool>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_fetch_channel",
    "group": "channel_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_create_channel",
    "group": "channel_api",
    "params": [
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_update_channel",
    "group": "channel_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_archive_channel",
    "group": "channel_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_unarchive_channel",
    "group": "channel_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_join_channel",
    "group": "channel_api",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_leave_channel",
    "group": "channel_api",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_fetch_messages",
    "group": "channel_api",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "before_id",
        "type": "Option<i64>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_send_message",
    "group": "channel_api",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_edit_message",
    "group": "channel_api",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "message_id",
        "type": "i64"
      },
      {
        "name": "content",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_delete_message",
    "group": "channel_api",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "message_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_fetch_unread_counts",
    "group": "channel_api",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "channel_mark_read",
    "group": "channel_api",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "message_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_mute_channel",
    "group": "channel_api",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "muted",
        "type": "bool"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_get_channel_pods",
    "group": "channel_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "channel_set_current_user",
    "group": "channel_state",
    "params": [
      {
        "name": "user_json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_set_current_user_id",
    "group": "channel_state",
    "params": [
      {
        "name": "user_id",
        "type": "Option<i64>"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_set_messages",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      },
      {
        "name": "has_more",
        "type": "bool"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_prepend_messages",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      },
      {
        "name": "has_more",
        "type": "bool"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_add_message",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_on_new_message",
    "group": "channel_state",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "bool"
  },
  {
    "name": "channel_update_message_local",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_remove_message_local",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "message_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_set_unread_counts",
    "group": "channel_state",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_increment_unread",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_clear_channel_unread",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_set_mention_counts",
    "group": "channel_state",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_increment_mention",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_clear_channel_mentions",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "channel_set_last_message",
    "group": "channel_state",
    "params": [
      {
        "name": "channel_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "extension_list_skill_registries",
    "group": "extension",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "extension_create_skill_registry",
    "group": "extension",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_sync_skill_registry",
    "group": "extension",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "extension_toggle_skill_registry",
    "group": "extension",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_delete_skill_registry",
    "group": "extension",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "extension_list_skill_registry_overrides",
    "group": "extension",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "extension_list_market_skills",
    "group": "extension",
    "params": [
      {
        "name": "query",
        "type": "Option<String>"
      },
      {
        "name": "category",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_list_market_mcp_servers",
    "group": "extension",
    "params": [
      {
        "name": "query",
        "type": "Option<String>"
      },
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_list_repo_skills",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "scope",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_install_skill_from_market",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_install_skill_from_github",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_update_skill",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "install_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_uninstall_skill",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "install_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "extension_list_repo_mcp_servers",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "scope",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_install_mcp_from_market",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_install_custom_mcp_server",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_update_mcp_server",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "install_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "extension_uninstall_mcp_server",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "install_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "extension_install_skill_from_upload",
    "group": "extension",
    "params": [
      {
        "name": "repo_id",
        "type": "i64"
      },
      {
        "name": "file_data",
        "type": "Vec<u8>"
      },
      {
        "name": "file_name",
        "type": "String"
      },
      {
        "name": "scope",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "file_presign_upload",
    "group": "file",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "file_upload_file",
    "group": "file",
    "params": [
      {
        "name": "file_data",
        "type": "Vec<u8>"
      },
      {
        "name": "filename",
        "type": "String"
      },
      {
        "name": "content_type",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "invitation_list",
    "group": "invitation",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "invitation_create",
    "group": "invitation",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "invitation_revoke",
    "group": "invitation",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "invitation_resend",
    "group": "invitation",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "invitation_get_by_token",
    "group": "invitation",
    "params": [
      {
        "name": "token",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "invitation_accept",
    "group": "invitation",
    "params": [
      {
        "name": "token",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "invitation_list_pending",
    "group": "invitation",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "loop_svc_loops_json",
    "group": "loop_service",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "loop_svc_current_loop_json",
    "group": "loop_service",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "loop_svc_runs_json",
    "group": "loop_service",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "loop_svc_get_loop_by_slug_json",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "loop_svc_set_loops",
    "group": "loop_service",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "loop_svc_set_current_loop",
    "group": "loop_service",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "loop_svc_update_loop_local",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "loop_svc_add_run",
    "group": "loop_service",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "loop_svc_set_runs",
    "group": "loop_service",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "loop_svc_append_runs",
    "group": "loop_service",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "loop_svc_update_run_status",
    "group": "loop_service",
    "params": [
      {
        "name": "run_id",
        "type": "i64"
      },
      {
        "name": "status",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "loop_svc_clear_runs",
    "group": "loop_service",
    "params": [],
    "returnType": "()"
  },
  {
    "name": "loop_svc_fetch_loops",
    "group": "loop_service",
    "params": [
      {
        "name": "status",
        "type": "Option<String>"
      },
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "loop_svc_fetch_loop",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "loop_svc_create_loop",
    "group": "loop_service",
    "params": [
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "loop_svc_update_loop",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "loop_svc_delete_loop",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "loop_svc_enable_loop",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "loop_svc_disable_loop",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "loop_svc_trigger_loop",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "loop_svc_fetch_runs",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "status",
        "type": "Option<String>"
      },
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "loop_svc_cancel_run",
    "group": "loop_service",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "run_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "mesh_topology_json",
    "group": "mesh",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "mesh_selected_node",
    "group": "mesh",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "mesh_get_node_json",
    "group": "mesh",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "mesh_get_edges_for_node_json",
    "group": "mesh",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "mesh_get_channels_for_node_json",
    "group": "mesh",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "mesh_get_active_nodes_json",
    "group": "mesh",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "mesh_get_nodes_by_runner_json",
    "group": "mesh",
    "params": [
      {
        "name": "runner_id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "mesh_get_runner_info_json",
    "group": "mesh",
    "params": [
      {
        "name": "runner_id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "mesh_set_topology",
    "group": "mesh",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "mesh_clear_topology",
    "group": "mesh",
    "params": [],
    "returnType": "()"
  },
  {
    "name": "mesh_select_node",
    "group": "mesh",
    "params": [
      {
        "name": "pod_key",
        "type": "Option<String>"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "mesh_fetch_topology",
    "group": "mesh",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "message_send_message",
    "group": "message",
    "params": [
      {
        "name": "json",
        "type": "String"
      },
      {
        "name": "pod_key",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "message_get_messages",
    "group": "message",
    "params": [
      {
        "name": "unread_only",
        "type": "Option<bool>"
      },
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "message_get_unread_count",
    "group": "message",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "message_get_message",
    "group": "message",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "message_mark_read",
    "group": "message",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "message_mark_all_read",
    "group": "message",
    "params": [],
    "returnType": "()"
  },
  {
    "name": "message_get_conversation",
    "group": "message",
    "params": [
      {
        "name": "correlation_id",
        "type": "String"
      },
      {
        "name": "limit",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "message_get_sent_messages",
    "group": "message",
    "params": [
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "message_get_dead_letters",
    "group": "message",
    "params": [
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "message_replay_dead_letter",
    "group": "message",
    "params": [
      {
        "name": "entry_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "notification_get_preferences",
    "group": "notification",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "notification_set_preference",
    "group": "notification",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "org_list",
    "group": "org",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "org_get",
    "group": "org",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "org_create",
    "group": "org",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "org_update",
    "group": "org",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "org_delete",
    "group": "org",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "org_list_members",
    "group": "org",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "org_invite_member",
    "group": "org",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "org_remove_member",
    "group": "org",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "user_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "org_update_member_role",
    "group": "org",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "user_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "pod_pods_json",
    "group": "pod",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "pod_current_pod_json",
    "group": "pod",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "pod_get_pod_json",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "pod_upsert_pod",
    "group": "pod",
    "params": [
      {
        "name": "pod_json",
        "type": "String"
      },
      {
        "name": "timestamp",
        "type": "Option<i64>"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_set_pods",
    "group": "pod",
    "params": [
      {
        "name": "pods_json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_set_current_pod",
    "group": "pod",
    "params": [
      {
        "name": "pod_json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_update_pod_status",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      },
      {
        "name": "status",
        "type": "String"
      },
      {
        "name": "agent_status",
        "type": "Option<String>"
      },
      {
        "name": "error_code",
        "type": "Option<String>"
      },
      {
        "name": "error_message",
        "type": "Option<String>"
      },
      {
        "name": "timestamp",
        "type": "Option<i64>"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_update_pod_title",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      },
      {
        "name": "title",
        "type": "String"
      },
      {
        "name": "timestamp",
        "type": "Option<i64>"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_update_pod_alias",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      },
      {
        "name": "alias",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_update_agent_status",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      },
      {
        "name": "agent_status",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_remove_pod",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_fetch_pods",
    "group": "pod",
    "params": [
      {
        "name": "status",
        "type": "Option<String>"
      },
      {
        "name": "runner_id",
        "type": "Option<i64>"
      },
      {
        "name": "created_by_id",
        "type": "Option<i64>"
      },
      {
        "name": "limit",
        "type": "Option<i64>"
      },
      {
        "name": "offset",
        "type": "Option<i64>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "pod_fetch_sidebar_pods",
    "group": "pod",
    "params": [
      {
        "name": "filter",
        "type": "String"
      },
      {
        "name": "user_id",
        "type": "Option<i64>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "pod_load_more_pods",
    "group": "pod",
    "params": [
      {
        "name": "filter",
        "type": "String"
      },
      {
        "name": "user_id",
        "type": "Option<i64>"
      },
      {
        "name": "offset",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "pod_fetch_pod",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "pod_create_pod",
    "group": "pod",
    "params": [
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "pod_terminate_pod",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_update_pod_alias_api",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      },
      {
        "name": "alias",
        "type": "Option<String>"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "pod_get_pod_connection",
    "group": "pod",
    "params": [
      {
        "name": "pod_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "repository_list",
    "group": "repository",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "repository_get",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "repository_create",
    "group": "repository",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "repository_update",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "repository_delete",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "repository_list_branches",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "repository_sync_branches",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "repository_register_webhook",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "repository_delete_webhook",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "repository_get_webhook_status",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "repository_get_webhook_secret",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "repository_list_merge_requests",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "branch",
        "type": "Option<String>"
      },
      {
        "name": "mr_state",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "repository_mark_webhook_configured",
    "group": "repository",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_runners_json",
    "group": "runner",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "runner_available_runners_json",
    "group": "runner",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "runner_current_runner_json",
    "group": "runner",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "runner_get_runner_json",
    "group": "runner",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_set_runners",
    "group": "runner",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_set_available_runners",
    "group": "runner",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_set_current_runner",
    "group": "runner",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_update_runner_local",
    "group": "runner",
    "params": [
      {
        "name": "id",
        "type": "f64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_update_runner_status",
    "group": "runner",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "status",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_remove_runner_local",
    "group": "runner",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_fetch_runners",
    "group": "runner_api",
    "params": [
      {
        "name": "status",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_fetch_available_runners",
    "group": "runner_api",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "runner_fetch_runner",
    "group": "runner_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_update_runner",
    "group": "runner_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_delete_runner",
    "group": "runner_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_create_token",
    "group": "runner_api",
    "params": [
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_fetch_tokens",
    "group": "runner_api",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "runner_delete_token",
    "group": "runner_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_list_runner_logs",
    "group": "runner_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_request_log_upload",
    "group": "runner_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "runner_upgrade_runner",
    "group": "runner_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_list_runner_pods",
    "group": "runner_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "status",
        "type": "Option<String>"
      },
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_query_runner_sandboxes",
    "group": "runner_api",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_get_auth_status",
    "group": "runner_api",
    "params": [
      {
        "name": "auth_key",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "runner_authorize_runner",
    "group": "runner_api",
    "params": [
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "support_ticket_list",
    "group": "support_ticket",
    "params": [
      {
        "name": "status",
        "type": "Option<String>"
      },
      {
        "name": "page",
        "type": "Option<u32>"
      },
      {
        "name": "page_size",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "support_ticket_get_detail",
    "group": "support_ticket",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "support_ticket_get_attachment_url",
    "group": "support_ticket",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "support_ticket_create_ticket",
    "group": "support_ticket",
    "params": [
      {
        "name": "title",
        "type": "String"
      },
      {
        "name": "category",
        "type": "String"
      },
      {
        "name": "content",
        "type": "String"
      },
      {
        "name": "priority",
        "type": "Option<String>"
      },
      {
        "name": "file_data",
        "type": "Vec<Vec<u8>>"
      },
      {
        "name": "file_names",
        "type": "Vec<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "support_ticket_add_message",
    "group": "support_ticket",
    "params": [
      {
        "name": "ticket_id",
        "type": "i64"
      },
      {
        "name": "content",
        "type": "String"
      },
      {
        "name": "file_data",
        "type": "Vec<Vec<u8>>"
      },
      {
        "name": "file_names",
        "type": "Vec<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_tickets_json",
    "group": "ticket",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "ticket_get_ticket_by_slug_json",
    "group": "ticket",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_current_ticket_json",
    "group": "ticket",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "ticket_board_columns_json",
    "group": "ticket",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "ticket_labels_json",
    "group": "ticket",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "ticket_filter_tickets_json",
    "group": "ticket",
    "params": [
      {
        "name": "search",
        "type": "String"
      },
      {
        "name": "statuses_json",
        "type": "String"
      },
      {
        "name": "priorities_json",
        "type": "String"
      },
      {
        "name": "repository_ids_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_set_tickets",
    "group": "ticket",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_add_ticket",
    "group": "ticket",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_update_ticket_local",
    "group": "ticket",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_update_ticket_status_local",
    "group": "ticket",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "status",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_remove_ticket",
    "group": "ticket",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_set_current_ticket",
    "group": "ticket",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_set_board_columns",
    "group": "ticket",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_append_column_tickets",
    "group": "ticket",
    "params": [
      {
        "name": "status",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_set_labels",
    "group": "ticket",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_add_label",
    "group": "ticket",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_remove_label",
    "group": "ticket",
    "params": [
      {
        "name": "id",
        "type": "f64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_fetch_tickets",
    "group": "ticket_api",
    "params": [
      {
        "name": "status",
        "type": "Option<String>"
      },
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_fetch_board",
    "group": "ticket_api",
    "params": [
      {
        "name": "repository_id",
        "type": "Option<i64>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_load_more_column",
    "group": "ticket_api",
    "params": [
      {
        "name": "status",
        "type": "String"
      },
      {
        "name": "offset",
        "type": "u32"
      },
      {
        "name": "limit",
        "type": "u32"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_fetch_ticket",
    "group": "ticket_api",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_create_ticket",
    "group": "ticket_api",
    "params": [
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_update_ticket",
    "group": "ticket_api",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "request_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_delete_ticket",
    "group": "ticket_api",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_update_ticket_status",
    "group": "ticket_api",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "status",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_fetch_labels",
    "group": "ticket_api",
    "params": [
      {
        "name": "repository_id",
        "type": "Option<i64>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_create_label",
    "group": "ticket_api",
    "params": [
      {
        "name": "name",
        "type": "String"
      },
      {
        "name": "color",
        "type": "String"
      },
      {
        "name": "repository_id",
        "type": "Option<i64>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_delete_label",
    "group": "ticket_api",
    "params": [
      {
        "name": "id",
        "type": "f64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_get_ticket_pods",
    "group": "ticket_api",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "active_only",
        "type": "Option<bool>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_get_sub_tickets",
    "group": "ticket_api",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_relations_list_relations",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_relations_create_relation",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_relations_delete_relation",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "relation_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_relations_list_commits",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_relations_link_commit",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_relations_unlink_commit",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "commit_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "ticket_relations_list_merge_requests",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_relations_list_comments",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "limit",
        "type": "Option<u32>"
      },
      {
        "name": "offset",
        "type": "Option<u32>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_relations_create_comment",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_relations_update_comment",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "comment_id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "ticket_relations_delete_comment",
    "group": "ticket_relations",
    "params": [
      {
        "name": "slug",
        "type": "String"
      },
      {
        "name": "comment_id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "user_get_me",
    "group": "user",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "user_get_organizations",
    "group": "user",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "user_credential_list_git_credentials",
    "group": "user_credential",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "user_credential_create_git_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_get_git_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_update_git_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_delete_git_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "user_credential_get_default_git_credential",
    "group": "user_credential",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "user_credential_set_default_git_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "user_credential_clear_default_git_credential",
    "group": "user_credential",
    "params": [],
    "returnType": "()"
  },
  {
    "name": "user_credential_list_agent_credentials",
    "group": "user_credential",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "user_credential_list_agent_credentials_for_agent",
    "group": "user_credential",
    "params": [
      {
        "name": "agent_slug",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_create_agent_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "agent_slug",
        "type": "String"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_get_agent_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_update_agent_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_delete_agent_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "user_credential_set_default_agent_credential",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "user_credential_list_repo_providers",
    "group": "user_credential",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "user_credential_create_repo_provider",
    "group": "user_credential",
    "params": [
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_get_repo_provider",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_update_repo_provider",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "user_credential_delete_repo_provider",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "user_credential_set_default_repo_provider",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "user_credential_test_repo_provider",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "user_credential_list_provider_repositories",
    "group": "user_credential",
    "params": [
      {
        "name": "id",
        "type": "i64"
      },
      {
        "name": "page",
        "type": "Option<u32>"
      },
      {
        "name": "per_page",
        "type": "Option<u32>"
      },
      {
        "name": "search",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "token_usage_get_dashboard",
    "group": "user_credential",
    "params": [
      {
        "name": "start_time",
        "type": "Option<String>"
      },
      {
        "name": "end_time",
        "type": "Option<String>"
      },
      {
        "name": "agent_slug",
        "type": "Option<String>"
      },
      {
        "name": "user_id",
        "type": "Option<i64>"
      },
      {
        "name": "model",
        "type": "Option<String>"
      },
      {
        "name": "granularity",
        "type": "Option<String>"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "blockstore_apply_ops",
    "group": "blockstore",
    "params": [
      {
        "name": "req_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "blockstore_list_workspaces",
    "group": "blockstore",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "blockstore_ensure_default_workspace",
    "group": "blockstore",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "blockstore_load_subtree",
    "group": "blockstore",
    "params": [
      {
        "name": "workspace_id",
        "type": "String"
      },
      {
        "name": "root_id",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "blockstore_load_type_defs",
    "group": "blockstore",
    "params": [
      {
        "name": "workspace_id",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "blockstore_catchup",
    "group": "blockstore",
    "params": [
      {
        "name": "workspace_id",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "blockstore_semantic_search",
    "group": "blockstore",
    "params": [
      {
        "name": "workspace_id",
        "type": "String"
      },
      {
        "name": "req_json",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "blockstore_apply_remote_op",
    "group": "blockstore",
    "params": [
      {
        "name": "op_json",
        "type": "String"
      }
    ],
    "returnType": "()"
  },
  {
    "name": "blockstore_workspaces_json",
    "group": "blockstore",
    "params": [],
    "returnType": "String"
  },
  {
    "name": "blockstore_get_block_json",
    "group": "blockstore",
    "params": [
      {
        "name": "id",
        "type": "String"
      }
    ],
    "returnType": "Option<String"
  },
  {
    "name": "blockstore_list_children_json",
    "group": "blockstore",
    "params": [
      {
        "name": "parent_id",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "blockstore_list_backlinks_json",
    "group": "blockstore",
    "params": [
      {
        "name": "target_id",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "blockstore_type_defs_json",
    "group": "blockstore",
    "params": [
      {
        "name": "workspace_id",
        "type": "String"
      }
    ],
    "returnType": "String"
  },
  {
    "name": "blockstore_last_op_id",
    "group": "blockstore",
    "params": [
      {
        "name": "workspace_id",
        "type": "String"
      }
    ],
    "returnType": "i64"
  },
  {
    "name": "blockstore_set_last_op_id",
    "group": "blockstore",
    "params": [
      {
        "name": "workspace_id",
        "type": "String"
      },
      {
        "name": "id",
        "type": "i64"
      }
    ],
    "returnType": "()"
  }
];
