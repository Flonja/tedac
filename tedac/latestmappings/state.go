package latestmappings

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"sort"
	"strings"
	"unsafe"
)

// State holds a combination of a name and properties, together with a version.
type State struct {
	// Name is the name of the block.
	Name string `nbt:"name"`
	// Properties is a map of properties that define the block's state.
	Properties map[string]interface{} `nbt:"states"`
	// Version is the version of the block state.
	Version int32 `nbt:"version"`
}

var (
	//go:embed block_states.nbt
	blockStateData []byte
	// stateRuntimeIDs holds a map for looking up the runtime ID of a block by the stateHash it produces.
	stateRuntimeIDs = map[stateHash]uint32{}
	// runtimeIDToState holds a map for looking up the blockState of a block by its runtime ID.
	runtimeIDToState = map[uint32]State{}
)

var (
	//go:embed item_runtime_ids.nbt
	itemRuntimeIDData []byte
	// itemRuntimeIDsToNames holds a map to translate item runtime IDs to string IDs.
	itemRuntimeIDsToNames = map[int32]string{}
	// itemNamesToRuntimeIDs holds a map to translate item string IDs to runtime IDs.
	itemNamesToRuntimeIDs = map[string]int32{}
)

// init initializes the item and state mappings.
func init() {
	var items map[string]int32
	if err := nbt.Unmarshal(itemRuntimeIDData, &items); err != nil {
		panic(err)
	}
	for name, rid := range items {
		itemNamesToRuntimeIDs[name] = rid
		itemRuntimeIDsToNames[rid] = name
	}

	dec := nbt.NewDecoder(bytes.NewBuffer(blockStateData))

	// Register all block states present in the block_states.nbt file. These are all possible options registered
	// blocks may encode to.
	var s State
	for {
		if err := dec.Decode(&s); err != nil {
			break
		}
		rid := uint32(len(stateRuntimeIDs))
		stateRuntimeIDs[stateHash{name: s.Name, properties: hashProperties(s.Properties)}] = rid
		runtimeIDToState[rid] = s
	}
}

// StateToRuntimeID converts a name and its state properties to a runtime ID.
func StateToRuntimeID(name string, properties map[string]any) (runtimeID uint32, found bool) {
	rid, ok := stateRuntimeIDs[stateHash{name: name, properties: hashProperties(properties)}]
	return rid, ok
}

// RuntimeIDToState converts a runtime ID to a name and its state properties.
func RuntimeIDToState(runtimeID uint32) (name string, properties map[string]any, found bool) {
	s := runtimeIDToState[runtimeID]
	return s.Name, s.Properties, true
}

// ItemRuntimeIDToName converts an item runtime ID to a string ID.
func ItemRuntimeIDToName(runtimeID int32) (name string, found bool) {
	name, ok := itemRuntimeIDsToNames[runtimeID]
	return name, ok
}

// ItemNameToRuntimeID converts a string ID to an item runtime ID.
func ItemNameToRuntimeID(name string) (runtimeID int32, found bool) {
	rid, ok := itemNamesToRuntimeIDs[name]
	return rid, ok
}

// stateHash is a struct that may be used as a map key for block states. It contains the name of the block state
// and an encoded version of the properties.
type stateHash struct {
	name, properties string
}

// hashProperties produces a hash for the block properties held by the blockState.
func hashProperties(properties map[string]any) string {
	if properties == nil {
		return ""
	}
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var b strings.Builder
	for _, k := range keys {
		switch v := properties[k].(type) {
		case bool:
			if v {
				b.WriteByte(1)
			} else {
				b.WriteByte(0)
			}
		case uint8:
			b.WriteByte(v)
		case int32:
			a := *(*[4]byte)(unsafe.Pointer(&v))
			b.Write(a[:])
		case string:
			b.WriteString(v)
		default:
			// If block encoding is broken, we want to find out as soon as possible. This saves a lot of time
			// debugging in-game.
			panic(fmt.Sprintf("invalid block property type %T for property %v", v, k))
		}
	}

	return b.String()
}
