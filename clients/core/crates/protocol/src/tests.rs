#[cfg(test)]
mod tests {
    use crate::*;

    #[test]
    fn encode_decode_roundtrip() {
        let payload = b"hello world";
        let encoded = encode_message(MsgType::Output, payload);
        assert_eq!(encoded[0], 0x02);
        assert_eq!(&encoded[1..], payload);

        let (msg_type, decoded_payload) = decode_message(&encoded).unwrap();
        assert_eq!(msg_type, MsgType::Output);
        assert_eq!(decoded_payload, payload);
    }

    #[test]
    fn encode_decode_all_msg_types() {
        let types = [
            MsgType::Snapshot,
            MsgType::Output,
            MsgType::Input,
            MsgType::Resize,
            MsgType::Ping,
            MsgType::Pong,
            MsgType::Control,
            MsgType::RunnerDisconnected,
            MsgType::RunnerReconnected,
            MsgType::Resync,
            MsgType::AcpEvent,
            MsgType::AcpCommand,
            MsgType::AcpSnapshot,
        ];
        for msg_type in types {
            let encoded = encode_message(msg_type, b"test");
            let (decoded_type, payload) = decode_message(&encoded).unwrap();
            assert_eq!(decoded_type, msg_type);
            assert_eq!(payload, b"test");
        }
    }

    #[test]
    fn encode_decode_empty_payload() {
        let encoded = encode_message(MsgType::Ping, &[]);
        assert_eq!(encoded.len(), 1);
        assert_eq!(encoded[0], 0x05);

        let (msg_type, payload) = decode_message(&encoded).unwrap();
        assert_eq!(msg_type, MsgType::Ping);
        assert!(payload.is_empty());
    }

    #[test]
    fn decode_empty_message_fails() {
        let result = decode_message(&[]);
        assert!(result.is_err());
    }

    #[test]
    fn decode_unknown_msg_type_fails() {
        let result = decode_message(&[0xFF, 0x01, 0x02]);
        assert!(result.is_err());
    }

    #[test]
    fn resize_encode_80x24() {
        let encoded = encode_resize(80, 24);
        assert_eq!(encoded.len(), 5);
        assert_eq!(encoded[0], MsgType::Resize as u8);
        assert_eq!(encoded[1], 0x00);
        assert_eq!(encoded[2], 0x50);
        assert_eq!(encoded[3], 0x00);
        assert_eq!(encoded[4], 0x18);
    }

    #[test]
    fn resize_decode_80x24() {
        let payload = [0x00, 0x50, 0x00, 0x18];
        let (cols, rows) = decode_resize(&payload).unwrap();
        assert_eq!(cols, 80);
        assert_eq!(rows, 24);
    }

    #[test]
    fn resize_roundtrip() {
        let encoded = encode_resize(200, 50);
        let (_, payload) = decode_message(&encoded).unwrap();
        let (cols, rows) = decode_resize(payload).unwrap();
        assert_eq!(cols, 200);
        assert_eq!(rows, 50);
    }

    #[test]
    fn resize_decode_invalid_length() {
        assert!(decode_resize(&[0x00, 0x50]).is_err());
        assert!(decode_resize(&[0x00, 0x50, 0x00, 0x18, 0xFF]).is_err());
    }

    #[test]
    fn json_encode_decode_roundtrip() {
        use serde::{Deserialize, Serialize};

        #[derive(Debug, Serialize, Deserialize, PartialEq)]
        struct ControlMsg {
            #[serde(rename = "type")]
            msg_type: String,
            rows: u16,
            cols: u16,
        }

        let msg = ControlMsg {
            msg_type: "pod_resized".to_string(),
            rows: 24,
            cols: 80,
        };

        let encoded = encode_json_message(MsgType::Control, &msg).unwrap();
        assert_eq!(encoded[0], MsgType::Control as u8);

        let (decoded_type, payload) = decode_message(&encoded).unwrap();
        assert_eq!(decoded_type, MsgType::Control);

        let decoded: ControlMsg = decode_json_payload(payload).unwrap();
        assert_eq!(decoded, msg);
    }

    #[test]
    fn msg_type_values_match_ts() {
        assert_eq!(MsgType::Snapshot as u8, 0x01);
        assert_eq!(MsgType::Output as u8, 0x02);
        assert_eq!(MsgType::Input as u8, 0x03);
        assert_eq!(MsgType::Resize as u8, 0x04);
        assert_eq!(MsgType::Ping as u8, 0x05);
        assert_eq!(MsgType::Pong as u8, 0x06);
        assert_eq!(MsgType::Control as u8, 0x07);
        assert_eq!(MsgType::RunnerDisconnected as u8, 0x08);
        assert_eq!(MsgType::RunnerReconnected as u8, 0x09);
        assert_eq!(MsgType::Resync as u8, 0x0a);
        assert_eq!(MsgType::AcpEvent as u8, 0x0b);
        assert_eq!(MsgType::AcpCommand as u8, 0x0c);
        assert_eq!(MsgType::AcpSnapshot as u8, 0x0d);
    }

    #[test]
    fn large_payload_over_64kb() {
        let payload = vec![0xAB_u8; 100_000];
        let encoded = encode_message(MsgType::Output, &payload);
        assert_eq!(encoded.len(), 1 + 100_000);

        let (msg_type, decoded) = decode_message(&encoded).unwrap();
        assert_eq!(msg_type, MsgType::Output);
        assert_eq!(decoded.len(), 100_000);
        assert!(decoded.iter().all(|&b| b == 0xAB));
    }

    #[test]
    fn msg_type_from_u8_zero_returns_none() {
        assert!(MsgType::from_u8(0x00).is_none());
    }

    #[test]
    fn msg_type_from_u8_above_max_returns_none() {
        for val in 0x0e..=0xFF {
            assert!(
                MsgType::from_u8(val).is_none(),
                "0x{val:02x} should return None"
            );
        }
    }

    #[test]
    fn msg_type_from_u8_all_valid_values() {
        let expected = [
            (0x01, MsgType::Snapshot),
            (0x02, MsgType::Output),
            (0x03, MsgType::Input),
            (0x04, MsgType::Resize),
            (0x05, MsgType::Ping),
            (0x06, MsgType::Pong),
            (0x07, MsgType::Control),
            (0x08, MsgType::RunnerDisconnected),
            (0x09, MsgType::RunnerReconnected),
            (0x0a, MsgType::Resync),
            (0x0b, MsgType::AcpEvent),
            (0x0c, MsgType::AcpCommand),
            (0x0d, MsgType::AcpSnapshot),
        ];
        for (val, expected_type) in expected {
            assert_eq!(MsgType::from_u8(val), Some(expected_type));
        }
    }

    #[test]
    fn decode_message_unknown_type_error_contains_value() {
        let err = decode_message(&[0xFE]).unwrap_err();
        let msg = err.to_string();
        assert!(msg.contains("0xfe"), "error should contain hex value: {msg}");
    }

    #[test]
    fn decode_message_empty_error_display() {
        let err = decode_message(&[]).unwrap_err();
        assert_eq!(err.to_string(), "empty message");
    }

    #[test]
    fn resize_boundary_zero_values() {
        let encoded = encode_resize(0, 0);
        let (_, payload) = decode_message(&encoded).unwrap();
        let (cols, rows) = decode_resize(payload).unwrap();
        assert_eq!(cols, 0);
        assert_eq!(rows, 0);
    }

    #[test]
    fn resize_boundary_u16_max() {
        let encoded = encode_resize(u16::MAX, u16::MAX);
        let (_, payload) = decode_message(&encoded).unwrap();
        let (cols, rows) = decode_resize(payload).unwrap();
        assert_eq!(cols, u16::MAX);
        assert_eq!(rows, u16::MAX);
    }

    #[test]
    fn resize_big_endian_byte_order() {
        let encoded = encode_resize(0x0102, 0x0304);
        assert_eq!(encoded[1], 0x01);
        assert_eq!(encoded[2], 0x02);
        assert_eq!(encoded[3], 0x03);
        assert_eq!(encoded[4], 0x04);
    }

    #[test]
    fn resize_decode_zero_bytes() {
        let err = decode_resize(&[]).unwrap_err();
        let msg = err.to_string();
        assert!(msg.contains("got 0"), "error should mention 0 bytes: {msg}");
    }

    #[test]
    fn resize_decode_three_bytes() {
        assert!(decode_resize(&[0x00, 0x50, 0x00]).is_err());
    }

    #[test]
    fn json_nested_object_roundtrip() {
        use serde::{Deserialize, Serialize};

        #[derive(Debug, Serialize, Deserialize, PartialEq)]
        struct Inner {
            value: i32,
        }

        #[derive(Debug, Serialize, Deserialize, PartialEq)]
        struct Outer {
            name: String,
            inner: Inner,
            items: Vec<String>,
        }

        let msg = Outer {
            name: "test".into(),
            inner: Inner { value: 42 },
            items: vec!["a".into(), "b".into()],
        };

        let encoded = encode_json_message(MsgType::AcpEvent, &msg).unwrap();
        let (_, payload) = decode_message(&encoded).unwrap();
        let decoded: Outer = decode_json_payload(payload).unwrap();
        assert_eq!(decoded, msg);
    }

    #[test]
    fn json_decode_invalid_payload() {
        let result = decode_json_payload::<serde_json::Value>(&[0xFF, 0xFE]);
        assert!(result.is_err());
    }

    #[test]
    fn json_encode_large_payload() {
        use serde::{Deserialize, Serialize};

        #[derive(Debug, Serialize, Deserialize, PartialEq)]
        struct BigMsg {
            data: String,
        }

        let msg = BigMsg {
            data: "x".repeat(100_000),
        };

        let encoded = encode_json_message(MsgType::Snapshot, &msg).unwrap();
        let (msg_type, payload) = decode_message(&encoded).unwrap();
        assert_eq!(msg_type, MsgType::Snapshot);
        let decoded: BigMsg = decode_json_payload(payload).unwrap();
        assert_eq!(decoded.data.len(), 100_000);
    }

    #[test]
    fn msg_type_debug_and_clone() {
        let t = MsgType::Ping;
        let cloned = t;
        assert_eq!(format!("{:?}", cloned), "Ping");
    }

    #[test]
    fn msg_type_hash() {
        use std::collections::HashSet;
        let mut set = HashSet::new();
        set.insert(MsgType::Ping);
        set.insert(MsgType::Pong);
        set.insert(MsgType::Ping);
        assert_eq!(set.len(), 2);
    }

    #[test]
    fn protocol_error_json_variant() {
        let bad_json = b"not json{{{";
        let err = decode_json_payload::<serde_json::Value>(bad_json).unwrap_err();
        let msg = err.to_string();
        assert!(msg.contains("json error"), "should be json error: {msg}");
    }

    #[test]
    fn single_byte_message_type_only() {
        let encoded = encode_message(MsgType::Pong, &[]);
        assert_eq!(encoded, vec![0x06]);

        let (msg_type, payload) = decode_message(&encoded).unwrap();
        assert_eq!(msg_type, MsgType::Pong);
        assert!(payload.is_empty());
    }

    #[test]
    fn resize_asymmetric_values() {
        let encoded = encode_resize(1, 65535);
        let (_, payload) = decode_message(&encoded).unwrap();
        let (cols, rows) = decode_resize(payload).unwrap();
        assert_eq!(cols, 1);
        assert_eq!(rows, 65535);
    }
}
