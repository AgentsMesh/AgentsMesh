-- Phase 1 schema upgrade: Block.Items goes from [][]InlineElement to [][]Block.
-- Each existing list-item (a flat array of inline elements) is wrapped into a
-- single paragraph block so the new readers see [{type:"paragraph",
-- elements:[...]}] in place of the old [...]. Walks both channel_messages and
-- channel_message_edits because both hold MessageContent JSONB.

CREATE OR REPLACE FUNCTION pg_temp.am_upgrade_block_items(node JSONB)
RETURNS JSONB AS $$
DECLARE
    blocks JSONB;
    new_blocks JSONB := '[]'::JSONB;
    block JSONB;
    new_block JSONB;
    items JSONB;
    new_items JSONB;
    item JSONB;
    children JSONB;
BEGIN
    IF node IS NULL OR jsonb_typeof(node) <> 'object' THEN
        RETURN node;
    END IF;
    blocks := node -> 'blocks';
    IF blocks IS NULL OR jsonb_typeof(blocks) <> 'array' THEN
        RETURN node;
    END IF;
    FOR block IN SELECT * FROM jsonb_array_elements(blocks) LOOP
        new_block := block;
        items := block -> 'items';
        IF items IS NOT NULL AND jsonb_typeof(items) = 'array' THEN
            new_items := '[]'::JSONB;
            FOR item IN SELECT * FROM jsonb_array_elements(items) LOOP
                IF jsonb_typeof(item) = 'array'
                   AND jsonb_array_length(item) > 0
                   AND (item -> 0 ->> 'type') NOT IN ('paragraph','heading','code_block','quote','list')
                THEN
                    new_items := new_items || jsonb_build_array(
                        jsonb_build_array(jsonb_build_object(
                            'type', 'paragraph',
                            'elements', item
                        ))
                    );
                ELSIF jsonb_typeof(item) = 'array' THEN
                    DECLARE
                        wrapped JSONB := '[]'::JSONB;
                        sub JSONB;
                    BEGIN
                        FOR sub IN SELECT * FROM jsonb_array_elements(item) LOOP
                            wrapped := wrapped || jsonb_build_array(
                                pg_temp.am_upgrade_block_items(jsonb_build_object('blocks', jsonb_build_array(sub))) -> 'blocks' -> 0
                            );
                        END LOOP;
                        new_items := new_items || jsonb_build_array(wrapped);
                    END;
                ELSE
                    new_items := new_items || jsonb_build_array(item);
                END IF;
            END LOOP;
            new_block := jsonb_set(new_block, '{items}', new_items);
        END IF;
        children := block -> 'children';
        IF children IS NOT NULL AND jsonb_typeof(children) = 'array' THEN
            new_block := jsonb_set(
                new_block,
                '{children}',
                pg_temp.am_upgrade_block_items(jsonb_build_object('blocks', children)) -> 'blocks'
            );
        END IF;
        new_blocks := new_blocks || jsonb_build_array(new_block);
    END LOOP;
    RETURN jsonb_set(node, '{blocks}', new_blocks);
END;
$$ LANGUAGE plpgsql;

UPDATE channel_messages
SET content = pg_temp.am_upgrade_block_items(content)
WHERE content IS NOT NULL
  AND content @? '$.**.items';

UPDATE channel_message_edits
SET previous_content = pg_temp.am_upgrade_block_items(previous_content)
WHERE previous_content IS NOT NULL
  AND previous_content @? '$.**.items';
