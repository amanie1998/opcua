# Dynamic Extension Object Decoding

> **Status:** Proof of Concept
> **File:** `dynamic_extension_object.go`

This document explains the dynamic extension object decoding feature, which allows you to decode custom OPC UA data types at runtime without pre-generated code.

## Overview

OPC UA servers can define custom data structures (Extension Objects). When your client receives data from such a custom type, it arrives as raw binary bytes. This feature provides tools to:

1. **Ask the server for the structure definition** (what fields does the data type have?)
2. **Decode the raw bytes into readable data** using that structure information

## Quick Start

```go
// 1. Get the structure definition (blueprint) from the server
def, err := opcua.ResolveStructureDefinition(ctx, client, encodingID)
if err != nil {
    log.Fatal(err)
}

// 2. Decode raw bytes using the blueprint
decoded, err := opcua.DecodeDynamicStructure(def, rawBytes)
if err != nil {
    log.Fatal(err)
}

// 3. Use the decoded data
fmt.Printf("Timestamp: %v\n", decoded["TsEquip"])
fmt.Printf("Value: %v\n", decoded["ActualBirds"])
```

## Functions

### `ResolveStructureDefinition`

```go
func ResolveStructureDefinition(
    ctx context.Context,
    c *Client,
    encodingID *ua.NodeID,
) (*ua.StructureDefinition, error)
```

**Purpose:** Retrieves the structure definition (field names and types) for a custom data type from the server.

**Parameters:**
- `ctx` - Context for timeout/cancellation
- `c` - Connected OPC UA client
- `encodingID` - The NodeID of the encoding (typically from `ExtensionObject.TypeID`)

**Returns:**
- `*ua.StructureDefinition` - The blueprint containing field definitions
- `error` - Any error encountered

**How it works:**
1. Takes the encoding NodeID (e.g., `ns=4;i=5002`)
2. Follows the `HasEncoding` reference **backwards** to find the DataType node
3. Reads the `DataTypeDefinition` attribute from the DataType node
4. Returns the `StructureDefinition` containing all field information

### `DecodeDynamicStructure`

```go
func DecodeDynamicStructure(
    def *ua.StructureDefinition,
    body []byte,
) (map[string]any, error)
```

**Purpose:** Decodes raw binary data into a key-value map using the structure definition.

**Parameters:**
- `def` - The structure definition from `ResolveStructureDefinition`
- `body` - Raw bytes from the Extension Object body

**Returns:**
- `map[string]any` - Decoded data with field names as keys
- `error` - Any error encountered

## Supported Data Types

| Type ID | OPC UA Type | Go Type    | Size    |
|---------|-------------|------------|---------|
| 6       | Int32       | `int32`    | 4 bytes |
| 7       | UInt32      | `uint32`   | 4 bytes |
| 13      | DateTime    | `time.Time`| 8 bytes |

## Example: Bird Counter

Real-world example from a poultry processing line:

```go
// The server has a custom "BirdCounterData" type with encoding ns=4;i=5002
encodingID := ua.NewNumericNodeID(4, 5002)

// Get the blueprint
def, err := opcua.ResolveStructureDefinition(ctx, client, encodingID)
// def.Fields contains:
//   [0] TsEquip         - DateTime (i=13)
//   [1] ActualBirds     - UInt32   (i=7)
//   [2] ShacklesCounted - UInt32   (i=7)
//   [3] EmptyShackles   - UInt32   (i=7)
//   [4] FeetHigh        - UInt32   (i=7)
//   [5] OneLegged       - UInt32   (i=7)
//   [6] LooseFeet       - UInt32   (i=7)
//   [7] State           - Int32    (i=6)
//   [8] ErrorCode       - Int32    (i=6)

// Decode raw bytes (40 bytes for this structure)
decoded, err := opcua.DecodeDynamicStructure(def, rawBytes)

// Result:
// {
//     "TsEquip":          2026-03-11 10:55:45 UTC,
//     "ActualBirds":      3507021,
//     "ShacklesCounted":  3578327,
//     "EmptyShackles":    71306,
//     "FeetHigh":         692787,
//     "OneLegged":        0,
//     "LooseFeet":        7177,
//     "State":            6,
//     "ErrorCode":        0,
// }
```

## How It Works

### Step 1: Resolving the Structure Definition

```
   Encoding Node                              DataType Node
   (ns=4;i=5002)                              (ns=4;i=5001)
   ┌─────────────┐         HasEncoding        ┌─────────────┐
   │  "Default   │  ◄─────────────────────    │ BirdCounter │
   │   Binary"   │     (browse inverse)       │  DataType   │
   └─────────────┘                            └─────────────┘
        ▲                                           │
        │                                           │
   We start here                            Read DataTypeDefinition
   (from ExtensionObject.TypeID)                    │
                                                    ▼
                                            StructureDefinition
                                            with Fields array
```

The code uses `BrowseDirectionInverse` because:
- The `HasEncoding` reference points FROM DataType TO Encoding
- We have the Encoding and need to find the DataType
- So we follow the reference **backwards**

### Step 2: Decoding the Bytes

```
Raw bytes (40 bytes):
┌────────┬───────┬───────┬───────┬───────┬───────┬───────┬───────┬───────┐
│TsEquip │Actual │Shackle│Empty  │Feet   │One    │Loose  │State  │Error  │
│8 bytes │4 bytes│4 bytes│4 bytes│4 bytes│4 bytes│4 bytes│4 bytes│4 bytes│
└────────┴───────┴───────┴───────┴───────┴───────┴───────┴───────┴───────┘

The decoder reads sequentially using the field types from the blueprint.
All values are little-endian (least significant byte first).
```

## Limitations

This is a proof-of-concept with the following limitations:

1. **Limited type support** - Only DateTime, Int32, and UInt32 are implemented
2. **No array support** - Cannot decode array fields
3. **No nested structures** - Cannot decode structures containing other structures
4. **No optional fields** - All fields are assumed to be present
5. **No string support** - String fields will cause an error

## TODO

- [ ] Add support for more primitive types (Boolean, Float, Double, String, etc.)
- [ ] Add array support (check `ValueRank` field)
- [ ] Add nested structure support
- [ ] Add optional field handling
- [ ] Add unit tests
- [ ] Consider caching structure definitions

## Technical Details

### DateTime Conversion

OPC UA DateTime is stored as Windows FILETIME (100-nanosecond intervals since January 1, 1601). The code converts this to Go's `time.Time`:

```go
const ticksPerSecond = int64(10000000)         // 10 million ticks = 1 second
const unixToFiletimeOffset = int64(11644473600) // Seconds between 1601 and 1970

sec := v/ticksPerSecond - unixToFiletimeOffset
nsec := (v % ticksPerSecond) * 100
result[f.Name] = time.Unix(sec, nsec).UTC()
```

### Little-Endian Byte Order

OPC UA binary encoding uses little-endian byte order:

```
Number 1000 (0x000003E8) in memory:
┌────┬────┬────┬────┐
│ E8 │ 03 │ 00 │ 00 │
└────┴────┴────┴────┘
 LSB            MSB
```

## Related Files

- `dynamic_extension_object.go` - Main implementation
- `client_aware_extension_object_dynamic.go` - Client-aware variant
- `ua/extobjs_gen.go` - StructureDefinition and StructureField types
- `ua/node_id.go` - NodeID type
- `ua/extension_object.go` - ExtensionObject type
