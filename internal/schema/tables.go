package schema

const (
	sizeInt64 = 8
	sizeFloat = 8
	sizeDate  = 8
	sizeBool  = 1
)

func mkSchema(fields ...Field) (*Schema, error) {
	s := &Schema{Fields: fields}
	if err := Validate(s); err != nil {
		return nil, err
	}
	return s, nil
}

func SeekerSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",            Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "email",         Type: TypeStr,   Size: 255, Flags: FlagUnique | FlagNotNull | FlagHash},
		Field{Name: "password_hash", Type: TypeStr,   Size: 128, Flags: FlagNotNull},
		Field{Name: "phone",         Type: TypeStr,   Size: 20},
		Field{Name: "full_name",     Type: TypeStr,   Size: 255, Flags: FlagNotNull},
		Field{Name: "birth_date",    Type: TypeDate,  Size: sizeDate},
		Field{Name: "city",          Type: TypeStr,   Size: 100, Flags: FlagHash},
		Field{Name: "status",        Type: TypeStr,   Size: 20,  Flags: FlagHash},
	)
}

func EmployerSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",                  Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "email",               Type: TypeStr,   Size: 255, Flags: FlagUnique | FlagNotNull | FlagHash},
		Field{Name: "password_hash",       Type: TypeStr,   Size: 128, Flags: FlagNotNull},
		Field{Name: "phone",               Type: TypeStr,   Size: 20},
		Field{Name: "company_name",        Type: TypeStr,   Size: 255, Flags: FlagNotNull},
		Field{Name: "inn",                 Type: TypeStr,   Size: 12,  Flags: FlagUnique | FlagHash},
		Field{Name: "website",             Type: TypeStr,   Size: 255},
		Field{Name: "verification_status", Type: TypeStr,   Size: 20,  Flags: FlagHash},
		Field{Name: "city",                Type: TypeStr,   Size: 100, Flags: FlagHash},
		Field{Name: "moderator_id",        Type: TypeInt64, Size: sizeInt64, Flags: FlagHash},
	)
}


func ModeratorSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",        Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "email",     Type: TypeStr,   Size: 255, Flags: FlagUnique | FlagNotNull | FlagHash},
		Field{Name: "full_name", Type: TypeStr,   Size: 255, Flags: FlagNotNull},
	)
}

func ResumeSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",             Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "seeker_id",      Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "position",       Type: TypeStr,   Size: 255, Flags: FlagNotNull | FlagHash},
		Field{Name: "desired_salary", Type: TypeInt64, Size: sizeInt64, Flags: FlagBTree},
		Field{Name: "category",       Type: TypeStr,   Size: 100, Flags: FlagHash},
		Field{Name: "employment_type",Type: TypeStr,   Size: 30,  Flags: FlagHash},
		Field{Name: "schedule",       Type: TypeStr,   Size: 30,  Flags: FlagHash},
		Field{Name: "status",         Type: TypeStr,   Size: 20,  Flags: FlagHash},
		Field{Name: "work_experience",Type: TypeStr,   Size: 2048},
		Field{Name: "education",      Type: TypeStr,   Size: 2048},
	)
}

func VacancySchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",                 Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "employer_id",        Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "title",              Type: TypeStr,   Size: 255, Flags: FlagNotNull | FlagHash},
		Field{Name: "city",               Type: TypeStr,   Size: 100, Flags: FlagHash},
		Field{Name: "description",        Type: TypeStr,   Size: 4096},
		Field{Name: "salary_from",        Type: TypeInt64, Size: sizeInt64, Flags: FlagBTree},
		Field{Name: "salary_to",          Type: TypeInt64, Size: sizeInt64},
		Field{Name: "required_experience",Type: TypeStr,   Size: 2048},
		Field{Name: "category",           Type: TypeStr,   Size: 100, Flags: FlagHash},
		Field{Name: "employment_type",    Type: TypeStr,   Size: 30,  Flags: FlagHash},
		Field{Name: "schedule",           Type: TypeStr,   Size: 30,  Flags: FlagHash},
		Field{Name: "status",             Type: TypeStr,   Size: 20,  Flags: FlagHash},
		Field{Name: "moderator_id",       Type: TypeInt64, Size: sizeInt64, Flags: FlagHash},  
	)
}

func SkillSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",   Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "name", Type: TypeStr,   Size: 100, Flags: FlagUnique | FlagNotNull | FlagHash},
	)
}

func ResumeSkillSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",        Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "resume_id", Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "skill_id",  Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
	)
}

func VacancySkillSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",         Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "vacancy_id", Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "skill_id",   Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
	)
}

func ApplicationSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",           Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "seeker_id",    Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "resume_id",    Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "vacancy_id",   Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "initiator",    Type: TypeStr,   Size: 10,  Flags: FlagNotNull},
		Field{Name: "cover_letter", Type: TypeStr,   Size: 2048},
		Field{Name: "status",       Type: TypeStr,   Size: 20,  Flags: FlagHash},
		Field{Name: "created_at",   Type: TypeDate,  Size: sizeDate, Flags: FlagNotNull | FlagBTree},
	)
}

func MessageSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",          Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "author_type", Type: TypeStr,   Size: 10,  Flags: FlagNotNull | FlagHash},
		Field{Name: "author_id",   Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "text",        Type: TypeStr,   Size: 2048, Flags: FlagNotNull},
		Field{Name: "sent_at",     Type: TypeDate,  Size: sizeDate, Flags: FlagNotNull | FlagBTree},
		Field{Name: "is_read",     Type: TypeBool,  Size: sizeBool},
	)
}

func ReviewSchema() (*Schema, error) {
	return mkSchema(
		Field{Name: "id",           Type: TypeInt64, Size: sizeInt64, Flags: FlagPK | FlagAutoInc | FlagNotNull},
		Field{Name: "seeker_id",    Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "employer_id",  Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagHash},
		Field{Name: "rating",       Type: TypeInt64, Size: sizeInt64, Flags: FlagNotNull | FlagBTree},
		Field{Name: "text",         Type: TypeStr,   Size: 2048},
		Field{Name: "is_published", Type: TypeBool,  Size: sizeBool},
		Field{Name: "created_at",   Type: TypeDate,  Size: sizeDate, Flags: FlagNotNull},
		Field{Name: "moderator_id", Type: TypeInt64, Size: sizeInt64, Flags: FlagHash},
	)
}