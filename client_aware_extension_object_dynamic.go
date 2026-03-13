package opcua

import (
	"context"
	"fmt"

	"github.com/gopcua/opcua/ua"
)

func DecodeExtensionObjectDynamically(
	ctx context.Context,
	c *Client,
	eo *ua.ExtensionObject,
) (any, error) {
	if eo == nil || eo.TypeID == nil || eo.TypeID.NodeID == nil {
		return nil, fmt.Errorf("invalid extension object")
	}

	raw, ok := eo.Value.([]byte)
	if !ok {
		return nil, fmt.Errorf("extension object value is %T, not []byte", eo.Value)
	}

	def, err := ResolveStructureDefinition(ctx, c, eo.TypeID.NodeID)
	if err != nil {
		return nil, err
	}

	return DecodeDynamicStructure(def, raw)
}