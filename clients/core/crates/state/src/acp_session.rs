use std::collections::HashMap;

use crate::acp_types::*;

pub fn now_millis() -> i64 {
    web_time::SystemTime::now()
        .duration_since(web_time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as i64
}

fn seal_last_thinking(thinkings: &mut [AcpThinking]) {
    if let Some(last) = thinkings.last_mut() {
        last.complete = true;
    }
}

fn trim_tool_calls(tool_calls: &mut HashMap<String, AcpToolCall>) {
    if tool_calls.len() <= MAX_TOOL_CALLS {
        return;
    }
    let to_remove = tool_calls.len() - MAX_TOOL_CALLS;
    let mut entries: Vec<_> = tool_calls
        .iter()
        .map(|(k, v)| (k.clone(), v.timestamp, v.status == "running"))
        .collect();
    entries.sort_by(|a, b| a.2.cmp(&b.2).then(a.1.cmp(&b.1)));
    for (key, _, _) in entries.iter().take(to_remove) {
        tool_calls.remove(key);
    }
}

fn cap_vec<T>(v: &mut Vec<T>, max: usize) {
    if v.len() > max {
        v.drain(..v.len() - max);
    }
}

#[derive(Debug, Default)]
pub struct AcpSessionManager {
    sessions: HashMap<String, AcpSession>,
}

impl AcpSessionManager {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn get_or_create_session(&mut self, pod_key: &str) -> &mut AcpSession {
        self.sessions.entry(pod_key.to_string()).or_default()
    }

    pub fn get_session(&self, pod_key: &str) -> Option<&AcpSession> {
        self.sessions.get(pod_key)
    }

    pub fn add_content_chunk(&mut self, pod_key: &str, text: &str, role: &str) {
        let session = self.get_or_create_session(pod_key);
        seal_last_thinking(&mut session.thinkings);
        let is_user = role == "user";

        if is_user {
            if let Some(last) = session.messages.last() {
                if last.role == "user" && last.complete && last.text == text {
                    return;
                }
            }
        }

        let should_append = !is_user
            && session
                .messages
                .last()
                .is_some_and(|m| m.role == role && !m.complete);

        if should_append {
            if let Some(last) = session.messages.last_mut() {
                last.text.push_str(text);
                last.timestamp = now_millis();
            }
        } else {
            session.messages.push(AcpContentChunk {
                text: text.to_string(),
                role: role.to_string(),
                timestamp: now_millis(),
                complete: is_user,
            });
        }
        cap_vec(&mut session.messages, MAX_MESSAGES);
    }

    pub fn mark_last_message_complete(&mut self, pod_key: &str) {
        if let Some(session) = self.sessions.get_mut(pod_key) {
            if let Some(last) = session.messages.last_mut() {
                last.complete = true;
            }
        }
    }

    pub fn update_tool_call(&mut self, pod_key: &str, tool_call: AcpToolCall) {
        let session = self.get_or_create_session(pod_key);
        seal_last_thinking(&mut session.thinkings);
        let ts = session
            .tool_calls
            .get(&tool_call.id)
            .map(|e| e.timestamp)
            .unwrap_or_else(now_millis);
        let mut tc = tool_call;
        tc.timestamp = ts;
        session.tool_calls.insert(tc.id.clone(), tc);
        trim_tool_calls(&mut session.tool_calls);
    }

    pub fn set_tool_call_result(
        &mut self,
        pod_key: &str,
        tool_call_id: &str,
        success: bool,
        result_text: Option<String>,
        error_message: Option<String>,
    ) {
        let session = self.get_or_create_session(pod_key);
        seal_last_thinking(&mut session.thinkings);
        if let Some(tc) = session.tool_calls.get_mut(tool_call_id) {
            tc.success = Some(success);
            tc.result_text = result_text;
            tc.error_message = error_message;
            tc.status = "completed".to_string();
        }
    }

    pub fn update_plan(&mut self, pod_key: &str, steps: Vec<AcpPlanStep>) {
        let session = self.get_or_create_session(pod_key);
        seal_last_thinking(&mut session.thinkings);
        session.plan = steps;
    }

    pub fn add_thinking(&mut self, pod_key: &str, text: &str) {
        let session = self.get_or_create_session(pod_key);
        let should_append = session.thinkings.last().is_some_and(|t| !t.complete);
        if should_append {
            if let Some(last) = session.thinkings.last_mut() {
                last.text.push_str(text);
                last.timestamp = now_millis();
            }
        } else {
            session.thinkings.push(AcpThinking {
                text: text.to_string(),
                timestamp: now_millis(),
                complete: false,
            });
        }
        cap_vec(&mut session.thinkings, MAX_THINKINGS);
    }

    pub fn add_permission_request(&mut self, pod_key: &str, req: AcpPermissionRequest) {
        let session = self.get_or_create_session(pod_key);
        seal_last_thinking(&mut session.thinkings);
        session.pending_permissions.push(req);
    }

    pub fn remove_permission_request(&mut self, pod_key: &str, request_id: &str) {
        if let Some(session) = self.sessions.get_mut(pod_key) {
            session.pending_permissions.retain(|p| p.id != request_id);
        }
    }

    pub fn update_session_state(&mut self, pod_key: &str, state: AcpState) {
        let session = self.get_or_create_session(pod_key);
        seal_last_thinking(&mut session.thinkings);
        if let Some(last) = session.messages.last_mut() {
            last.complete = true;
        }
        session.state = state;
    }

    pub fn add_log(&mut self, pod_key: &str, level: &str, message: &str) {
        let session = self.get_or_create_session(pod_key);
        session.logs.push(AcpLog {
            level: level.to_string(),
            message: message.to_string(),
            timestamp: now_millis(),
        });
        cap_vec(&mut session.logs, MAX_LOGS);
    }

    pub fn update_configuration(&mut self, pod_key: &str, update: AcpConfiguration) {
        let session = self.get_or_create_session(pod_key);
        if !update.permission_mode.is_empty() {
            session.configuration.permission_mode = update.permission_mode;
        }
        if !update.model.is_empty() {
            session.configuration.model = update.model;
        }
    }

    pub fn clear_session(&mut self, pod_key: &str) {
        self.sessions.remove(pod_key);
    }
}
