package schema

import "encoding/binary"


func PutU64(buf []byte, v uint64) {
    binary.LittleEndian.PutUint64(buf, v)
}

func U64(buf []byte) uint64 {
    return binary.LittleEndian.Uint64(buf)
}