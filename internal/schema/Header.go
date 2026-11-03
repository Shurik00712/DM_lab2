package schema

import (
	"fmt"
	"encoding/binary"
    "bytes"
    "errors"
)


var headerMagic = [4]byte{'M', 'Y', 'D', 'B'}

func EncodeHeader(schema *Schema, recordCount uint64, freeHead int64) ([]byte, error) {
    if err := Validate(schema); err != nil {
        return nil, fmt.Errorf("invalid schema: %w", err)
    }

    fieldCount := len(schema.Fields)
    buf := make([]byte, fixedHeaderSize+fieldCount*fieldDescSize)
    pos := 0

    copy(buf[pos:pos+4], headerMagic[:]);                              pos += 4
    binary.LittleEndian.PutUint32(buf[pos:pos+4], headerVersion);      pos += 4
    binary.LittleEndian.PutUint32(buf[pos:pos+4], uint32(schema.RecordSize)); pos += 4
    binary.LittleEndian.PutUint64(buf[pos:pos+8], recordCount);        pos += 8
    binary.LittleEndian.PutUint64(buf[pos:pos+8], uint64(freeHead));   pos += 8
    binary.LittleEndian.PutUint32(buf[pos:pos+4], uint32(fieldCount)); pos += 4

    for _, f := range schema.Fields {
        copy(buf[pos:pos+32], f.Name);                                 pos += 32   
        buf[pos] = byte(f.Type);                                       pos += 1
        binary.LittleEndian.PutUint32(buf[pos:pos+4], uint32(f.Size)); pos += 4
        binary.LittleEndian.PutUint16(buf[pos:pos+2], uint16(f.Flags));pos += 2
    }

    return buf, nil
}

func DecodeHeader(data []byte) (*Schema, uint64, int64, error) {
	if len(data) < fixedHeaderSize {
		return nil, 0, 0, fmt.Errorf("truncated header: %d < %d", len(data), fixedHeaderSize)
	}
	pos := 0

	if !bytes.Equal(data[0:4], headerMagic[:]) {
		return nil, 0, 0, fmt.Errorf("not a database file")
	}
	pos+=4
	version := binary.LittleEndian.Uint32(data[pos:pos+4])
	if version != headerVersion {
		return nil, 0, 0, fmt.Errorf("unsupported version %d", version)
	}
	pos+=4
	recordSize := int(binary.LittleEndian.Uint32(data[pos:pos+4]))
	pos += 4
    recordCount := binary.LittleEndian.Uint64(data[pos:pos+8])
    pos += 8
    freeHead := int64(binary.LittleEndian.Uint64(data[pos:pos+8]))
    pos += 8
    fieldCount := int(binary.LittleEndian.Uint32(data[pos:pos+4]))
    pos += 4
	if fieldCount <= 0 || fieldCount > maxFields {
        return nil, 0, 0, fmt.Errorf("corrupted: absurd fieldCount %d", fieldCount)
    }
	need := fixedHeaderSize+fieldCount*fieldDescSize
	if len(data) < need {
		return nil, 0, 0, fmt.Errorf("truncated header: %d < %d", len(data), need)
	}
	fields := make([]Field, fieldCount)
	for i:=0; i < fieldCount; i++ {
		val := data[pos:pos+32]
		valLen := 32
		for j := 0; j < 32; j++ {
            if val[j] == 0 {
                valLen = j
                break
            }
        }
		fields[i].Name = string(val[:valLen])
        pos += 32
		typ := FieldType(data[pos])
        if typ < TypeInt64 || typ > TypeDate {
            return nil, 0, 0, fmt.Errorf("field %d: unknown type %d", i, typ)
        }
        fields[i].Type = typ
        pos += 1
		fields[i].Size = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
        pos += 4

        fields[i].Flags = FieldFlags(binary.LittleEndian.Uint16(data[pos : pos+2]))
        pos += 2
	}
	sum := 0
    for _, f := range fields {
        sum += f.Size
    }
    if recordSize != sum {
        return nil, 0, 0, fmt.Errorf("corrupted: recordSize %d != sum(Size) %d", recordSize, sum)
    }
	schema := &Schema{Fields: fields, RecordSize: recordSize}
	return schema, recordCount, freeHead, nil
}

func HeaderSizeFromPrefix(prefix []byte) (int, error) {
    if len(prefix) < fixedHeaderSize {
        return 0, fmt.Errorf("prefix too short: %d", len(prefix))
    }
    if !bytes.Equal(prefix[0:4], headerMagic[:]) {
        return 0, errors.New("not a database file")
    }
    fieldCount := int(binary.LittleEndian.Uint32(prefix[fixedHeaderSize-4:]))
    if fieldCount <= 0 || fieldCount > 1024 {
        return 0, fmt.Errorf("corrupted: absurd fieldCount %d", fieldCount)
    }
    return fixedHeaderSize + fieldCount*fieldDescSize, nil
}