package opcua

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func ResolveStructureDefinition(
	ctx context.Context,
	c *Client,
	encodingID *ua.NodeID,
) (*ua.StructureDefinition, error) {
	encNode := c.Node(encodingID)

	refs, err := encNode.References(
		ctx,
		id.HasEncoding,
		ua.BrowseDirectionInverse,
		ua.NodeClassDataType,
		false,
	)
	if err != nil {
		return nil, err
	}

	if len(refs) == 0 {
		return nil, fmt.Errorf("datatype not found for encoding %s", encodingID)
	}

	dtID := refs[0].NodeID.NodeID
	dtNode := c.Node(dtID)

	v, err := dtNode.Attribute(ctx, ua.AttributeIDDataTypeDefinition)
	if err != nil {
		return nil, err
	}
	if v == nil || v.Value() == nil {
		return nil, fmt.Errorf("datatype definition attribute is nil")
	}

	eo, ok := v.Value().(*ua.ExtensionObject)
	if !ok {
		return nil, fmt.Errorf("unexpected datatype definition type %T", v.Value())
	}

	def, ok := eo.Value.(*ua.StructureDefinition)
	if !ok {
		return nil, fmt.Errorf("datatype definition is not StructureDefinition: %T", eo.Value)
	}

	return def, nil
}

func DecodeDynamicStructure(def *ua.StructureDefinition, body []byte) (map[string]any, error) {
	r := bytes.NewReader(body)
	result := make(map[string]any, len(def.Fields))

	for _, f := range def.Fields {
		switch f.DataType.IntID() {

		// DateTime
		case 13:
			var v int64
			if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
				return nil, fmt.Errorf("field %s: %w", f.Name, err)
			}
			if v == 0 {
				result[f.Name] = time.Time{}
			} else {
				const ticksPerSecond = int64(10000000)
				const unixToFiletimeOffset = int64(11644473600)
				sec := v/ticksPerSecond - unixToFiletimeOffset
				nsec := (v % ticksPerSecond) * 100
				result[f.Name] = time.Unix(sec, nsec).UTC()
			}

		// Int32
		case 6:
			var v int32
			if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
				return nil, fmt.Errorf("field %s: %w", f.Name, err)
			}
			result[f.Name] = v

		// UInt32
		case 7:
			var v uint32
			if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
				return nil, fmt.Errorf("field %s: %w", f.Name, err)
			}
			result[f.Name] = v

		default:
			return nil, fmt.Errorf("field %s: unsupported datatype %v", f.Name, f.DataType)
		}
	}

	if r.Len() != 0 {
		result["_remaining_bytes"] = r.Len()
	}

	return result, nil
}