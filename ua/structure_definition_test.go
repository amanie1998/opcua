package ua_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

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
}