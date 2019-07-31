package schema

var IntType = &ScalarType{
	Name: "Int",
}

var FloatType = &ScalarType{
	Name: "Float",
}

var StringType = &ScalarType{
	Name: "String",
}

var BooleanType = &ScalarType{
	Name: "Boolean",
}

var IDType = &ScalarType{
	Name: "ID",
}

var builtins = map[string]*ScalarType{
	"Int":     IntType,
	"Float":   FloatType,
	"String":  StringType,
	"Boolean": BooleanType,
	"ID":      IDType,
}
