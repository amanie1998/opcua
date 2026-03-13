package ua_test

import (
	"context"
	"fmt"
	"testing"
	"bytes"
	"encoding/binary"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func DecodeDynamicStructure(def *ua.StructureDefinition, body []byte) (map[string]any, error) {

	r := bytes.NewReader(body)

	result := make(map[string]any)

	for _, f := range def.Fields {

		switch f.DataType.IntID() {

		// DateTime
		case 13:
			var v int64
			if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
				return nil, err
			}

			if v == 0 {
				result[f.Name] = time.Time{}
			} else {
				const ticksPerSecond = int64(10000000)
				const unixToFiletimeOffset = int64(11644473600) // seconds
				sec := v/ticksPerSecond - unixToFiletimeOffset
				nsec := (v % ticksPerSecond) * 100
				result[f.Name] = time.Unix(sec, nsec).UTC()
			}

		// UInt32
		case 7:
			var v uint32
			if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
				return nil, err
			}
			result[f.Name] = v

		// Int32
		case 6:
			var v int32
			if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
				return nil, err
			}
			result[f.Name] = v

		default:
			return nil, fmt.Errorf("unsupported datatype %v", f.DataType)
		}
	}

	return result, nil
}

func ResolveStructureDefinition(
	ctx context.Context,
	c *opcua.Client,
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
	if v == nil {
		return nil, fmt.Errorf("datatype definition attribute is nil")
	}

	eo, ok := v.Value().(*ua.ExtensionObject)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", v.Value())
	}

	def, ok := eo.Value.(*ua.StructureDefinition)
	if !ok {
		return nil, fmt.Errorf("not a structure definition: %T", eo.Value)
	}

	return def, nil
}

func TestResolveStructureDefinition(t *testing.T) {
	ctx := context.Background()

	endpoint := "opc.tcp://localhost:14840"

	c, err := opcua.NewClient(endpoint)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)

	encID := ua.NewNumericNodeID(4, 5002)

	def, err := ResolveStructureDefinition(ctx, c, encID)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("STRUCTURE: %+v\n", def)
	for _, f := range def.Fields {
		fmt.Printf("FIELD: %s TYPE: %v\n", f.Name, f.DataType)
	}

	fmt.Println("FIELDS FOUND:")

	for _, f := range def.Fields {
		fmt.Println("FIELD:", f.Name, "TYPE:", f.DataType)
	}

	// example raw payload (replace later with real EO body)
	body := []byte{
		0xb0, 0xab, 0xdf, 0x9c, 0x45, 0xb1, 0xdc, 0x1,
		0x4d, 0x83, 0x35, 0x0,
		0xd7, 0x99, 0x36, 0x0,
		0x8a, 0x16, 0x1, 0x0,
		0x33, 0x92, 0xa, 0x0,
		0x0, 0x0, 0x0, 0x0,
		0x9, 0x1c, 0x0, 0x0,
		0x6, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0,
	}

	decoded, err := DecodeDynamicStructure(def, body)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("DECODED STRUCTURE:")

	for k, v := range decoded {
		fmt.Println(k, "=", v)
	}
}



