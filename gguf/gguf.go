package gguf

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	MagicGGUF = 0x46554747 // "GGUF" in little-endian

	TypeUint8   = 0
	TypeInt8    = 1
	TypeUint16  = 2
	TypeInt16   = 3
	TypeUint32  = 4
	TypeInt32   = 5
	TypeFloat32 = 6
	TypeBool    = 7
	TypeString  = 8
	TypeArray   = 9
	TypeUint64  = 10
	TypeInt64   = 11
	TypeFloat64 = 12
)

// Header holds the parsed GGUF header info.
type Header struct {
	Version       uint32
	TensorCount   uint64
	MetadataCount uint64
}

// Metadata holds key-value pairs parsed from the GGUF file.
type Metadata map[string]interface{}

// Info holds all parsed information from a GGUF file.
type Info struct {
	Header   Header
	Metadata Metadata
}

// ReadInfo reads the header and metadata from a GGUF file.
// It does not read tensor data.
func ReadInfo(path string) (*Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ReadInfoFromReader(f)
}

// ReadInfoFromReader reads the header and metadata from a reader.
func ReadInfoFromReader(r io.Reader) (*Info, error) {
	// Read magic
	var magic uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, fmt.Errorf("failed to read magic: %w", err)
	}
	if magic != MagicGGUF {
		return nil, fmt.Errorf("not a GGUF file (magic: 0x%08X)", magic)
	}

	// Read version
	var version uint32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}
	if version < 2 || version > 3 {
		return nil, fmt.Errorf("unsupported GGUF version: %d", version)
	}

	// Read tensor count and metadata count
	var tensorCount, metadataCount uint64
	if version >= 3 {
		if err := binary.Read(r, binary.LittleEndian, &tensorCount); err != nil {
			return nil, fmt.Errorf("failed to read tensor count: %w", err)
		}
		if err := binary.Read(r, binary.LittleEndian, &metadataCount); err != nil {
			return nil, fmt.Errorf("failed to read metadata count: %w", err)
		}
	} else {
		// v2 also uses uint64
		if err := binary.Read(r, binary.LittleEndian, &tensorCount); err != nil {
			return nil, fmt.Errorf("failed to read tensor count: %w", err)
		}
		if err := binary.Read(r, binary.LittleEndian, &metadataCount); err != nil {
			return nil, fmt.Errorf("failed to read metadata count: %w", err)
		}
	}

	// Read metadata key-value pairs
	metadata := make(Metadata)
	for i := uint64(0); i < metadataCount; i++ {
		key, err := readString(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata key %d: %w", i, err)
		}

		value, err := readValue(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata value for key %q: %w", key, err)
		}

		metadata[key] = value
	}

	return &Info{
		Header: Header{
			Version:       version,
			TensorCount:   tensorCount,
			MetadataCount: metadataCount,
		},
		Metadata: metadata,
	}, nil
}

// GetString returns a string metadata value.
func (m Metadata) GetString(key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUint32 returns a uint32 metadata value.
func (m Metadata) GetUint32(key string) uint32 {
	if v, ok := m[key]; ok {
		if u, ok := v.(uint32); ok {
			return u
		}
	}
	return 0
}

// GetUint64 returns a uint64 metadata value.
func (m Metadata) GetUint64(key string) uint64 {
	if v, ok := m[key]; ok {
		if u, ok := v.(uint64); ok {
			return u
		}
	}
	return 0
}

// GetFloat32 returns a float32 metadata value.
func (m Metadata) GetFloat32(key string) float32 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float32); ok {
			return f
		}
	}
	return 0
}

// GetBool returns a bool metadata value.
func (m Metadata) GetBool(key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// Has returns true if the key exists in metadata.
func (m Metadata) Has(key string) bool {
	_, ok := m[key]
	return ok
}

func readString(r io.Reader) (string, error) {
	var length uint64
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return "", err
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

func readValue(r io.Reader) (interface{}, error) {
	var valueType uint32
	if err := binary.Read(r, binary.LittleEndian, &valueType); err != nil {
		return nil, err
	}

	return readTypedValue(r, valueType)
}

func readTypedValue(r io.Reader, valueType uint32) (interface{}, error) {
	switch valueType {
	case TypeUint8:
		var v uint8
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case TypeInt8:
		var v int8
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case TypeUint16:
		var v uint16
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case TypeInt16:
		var v int16
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case TypeUint32:
		var v uint32
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case TypeInt32:
		var v int32
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case TypeFloat32:
		var v float32
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case TypeBool:
		var v uint8
		err := binary.Read(r, binary.LittleEndian, &v)
		return v != 0, err
	case TypeString:
		return readString(r)
	case TypeArray:
		var elemType uint32
		if err := binary.Read(r, binary.LittleEndian, &elemType); err != nil {
			return nil, err
		}
		var length uint64
		if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
			return nil, err
		}
		arr := make([]interface{}, length)
		for i := uint64(0); i < length; i++ {
			v, err := readTypedValue(r, elemType)
			if err != nil {
				return nil, err
			}
			arr[i] = v
		}
		return arr, nil
	case TypeUint64:
		var v uint64
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case TypeInt64:
		var v int64
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case TypeFloat64:
		var v float64
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	default:
		return nil, fmt.Errorf("unknown metadata type: %d", valueType)
	}
}

// FileTypeString returns a human-readable name for the GGUF file_type enum.
func FileTypeString(ft uint32) string {
	switch ft {
	case 0:
		return "F32"
	case 1:
		return "F16"
	case 2:
		return "Q4_0"
	case 3:
		return "Q4_1"
	case 7:
		return "Q8_0"
	case 8:
		return "Q5_0"
	case 9:
		return "Q5_1"
	case 10:
		return "Q2_K"
	case 11:
		return "Q3_K_S"
	case 12:
		return "Q3_K_M"
	case 13:
		return "Q3_K_L"
	case 14:
		return "Q4_K_S"
	case 15:
		return "Q4_K_M"
	case 16:
		return "Q5_K_S"
	case 17:
		return "Q5_K_M"
	case 18:
		return "Q6_K"
	default:
		return fmt.Sprintf("unknown(%d)", ft)
	}
}
