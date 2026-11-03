package schema

type FieldType uint8

const (
    headerVersion   = 1
    fixedHeaderSize = 32   
    fieldDescSize   = 39   
)

func FixedHeaderSize() int {
    return fixedHeaderSize
}

const (
    maxFields     = 1024
    maxStrSize    = 8192
    maxRecordSize = 64 * 1024
    maxFieldName  = 32
)

const (
	TypeInt64 FieldType = 1
	TypeFloat64 FieldType = 2 
    TypeStr     FieldType = 3 
    TypeBool    FieldType = 4
    TypeDate    FieldType = 5 
)

type FieldFlags uint16

const (
    FlagPK       FieldFlags = 1 << 0
    FlagUnique   FieldFlags = 1 << 1
    FlagNotNull  FieldFlags = 1 << 2
    FlagHash     FieldFlags = 1 << 3
    FlagBTree    FieldFlags = 1 << 4
    FlagAutoInc  FieldFlags = 1 << 5
    FlagNullable FieldFlags = 1 << 6
)

func (f FieldFlags) Has(x FieldFlags) bool { return f&x != 0 }

type Field struct {
	Name string
	Type FieldType
	Size int
	Flags FieldFlags
}

type Schema struct {
	Fields []Field
	RecordSize int
}

func HeaderSize(s *Schema) int {
    return fixedHeaderSize + len(s.Fields)*fieldDescSize
}
