package ruby_types

import (
	"fmt"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/thematthopkins/elm-protobuf/pkg/elm"
	"google.golang.org/protobuf/types/descriptorpb"
	"log"
	"strings"

	pgs "github.com/lyft/protoc-gen-star"
)

type methodType int

const (
	methodTypeGetter methodType = iota
	methodTypeSetter
	methodTypeInitializer
)

// intersection between pgs.FieldType and pgs.FieldTypeElem
type FieldType interface {
	ProtoType() pgs.ProtoType
	IsEmbed() bool
	IsEnum() bool
	Imports() []pgs.File
	Enum() pgs.Enum
	Embed() pgs.Message
}

// intersection between pgs.Message and pgs.Enum
type EntityWithParent interface {
	pgs.Entity
	Parent() pgs.ParentEntity
}

func RubyPackage(file pgs.File) string {
	pkg := file.Descriptor().GetOptions().GetRubyPackage()
	if pkg == "" {
		pkg = file.Descriptor().GetPackage()
	}
	pkg = strings.Replace(pkg, ".", "::", -1)
	// right now the ruby_out doesn't camelcase the ruby_package, but this results in invalid classes, so do it:
	return upperCamelCase(pkg)
}

func Validators(field pgs.Field) []string {
	results := []string{}
	validators := elm.Validators(
		field.Type().Field().Message().File().Descriptor(),
		field.Type().Field().Message().Descriptor(),
		field.Type().Field().Descriptor(),
		field.Type().Field().Descriptor().GetProto3Optional(),
	)

	for _, v := range validators {
		results = append(results, validatorToString(v))
	}
	return results
}

func validatorToString(v elm.Validator) string {
	validatorName := "Validation::" + ((pgs.Name)(v.Name)).UpperCamelCase().String()
	if v.FieldArg != "" {
		return validatorName + ".new( ->(m) { m." + ((pgs.Name)(v.FieldArg)).LowerSnakeCase().String() + "})"
	} else if v.ValidatorArg != "" {
		return validatorName + ".new(" + ((pgs.Name)(v.ValidatorArg)).UpperCamelCase().String() + "Validators.new.all_model_validators)"
	} else {
		return validatorName + ".new"
	}
}

func Translation(field pgs.Field) *pgs.Name {
	return (*pgs.Name)(elm.GetTranslation(field.Type().Field().Descriptor()))
}

func OneOfTranslation(oneof pgs.OneOf) *pgs.Name {
	return Translation(oneof.Fields()[0])
}

func OneOfValidators(oneOf pgs.OneOf) []string {
	results := []string{}
	fields := []*descriptorpb.FieldDescriptorProto{}
	for _, v := range oneOf.Fields() {
		fields = append(fields, v.Descriptor())
	}

	validators := elm.OneOfValidatorsForFields(
		oneOf.Message().File().Descriptor(),
		oneOf.Message().Descriptor(),
		fields,
	)

	for _, v := range validators {
		results = append(results, validatorToString(v))
	}
	return results
}

func RubyMessageType(entity EntityWithParent) string {
	names := make([]string, 0)
	outer := entity
	ok := true
	for ok {
		name := outer.Name().UpperCamelCase()
		names = append([]string{name.String()}, names...)
		outer, ok = outer.Parent().(pgs.Message)
	}
	return fmt.Sprintf("%s::%s", RubyPackage(entity.File()), strings.Join(names, "::"))
}

func RubyGetterFieldType(field pgs.Field) string {
	return rubyFieldType(field, methodTypeGetter)
}

func isStringEncodedInt(fieldType pgs.ProtoType) bool {
	return fieldType == pgs.Int64T ||
		fieldType == pgs.UInt64T ||
		fieldType == pgs.SInt64 ||
		fieldType == pgs.Fixed64T ||
		fieldType == pgs.SFixed64
}

func FieldEncoder(field string, fieldType pgs.FieldType) string {
	operation := ""
	if isIdType(fieldType.Field().Descriptor()) {
		operation = "value"
	} else if fieldType.IsEmbed() && RubyMessageType(fieldType.Embed()) == "Google::Protobuf::Timestamp" ||
		fieldType.IsRepeated() && fieldType.Element().IsEmbed() && RubyMessageType(fieldType.Element().Embed()) == "Google::Protobuf::Timestamp" {

		operation = "iso8601"
	} else if fieldType.IsEmbed() || (fieldType.IsRepeated() && fieldType.Element().IsEmbed()) {
		operation = "serialize"
	} else if fieldType.IsEnum() || (fieldType.IsRepeated() && fieldType.Element().IsEnum()) {
		operation = "serialize"
	} else if isStringEncodedInt(fieldType.ProtoType()) || (fieldType.IsRepeated() && isStringEncodedInt(fieldType.Element().ProtoType())) {
		operation = "to_s"
	} else {
		return field
	}

	nullHandler := ""
	if fieldType.Field().Descriptor().GetProto3Optional() {
		nullHandler = "&"
	}

	if fieldType.IsRepeated() {
		return fmt.Sprintf("%s%s.map(&:%s)", field, nullHandler, operation)
	} else {
		return fmt.Sprintf("%s%s.%s", field, nullHandler, operation)
	}
}

func FieldDecoder(key string, fieldType pgs.FieldType) string {
	operation := ""
	if isIdType(fieldType.Field().Descriptor()) {
		operation = *idTypeName(fieldType) + ".new"
	} else if fieldType.IsEmbed() && RubyMessageType(fieldType.Embed()) == "Google::Protobuf::Timestamp" ||
		fieldType.IsRepeated() && fieldType.Element().IsEmbed() && RubyMessageType(fieldType.Element().Embed()) == "Google::Protobuf::Timestamp" {

		operation = "Time.iso8601"
	} else if fieldType.IsEmbed() || (fieldType.IsRepeated() && fieldType.Element().IsEmbed()) {
		typeName := ""
		if fieldType.IsRepeated() {
			typeName = RubyMessageType(fieldType.Element().Embed())
		} else {
			typeName = RubyMessageType(fieldType.Embed())
		}
		operation = typeName + ".from_hash"
	} else if fieldType.IsEnum() || (fieldType.IsRepeated() && fieldType.Element().IsEnum()) {
		typeName := ""
		if fieldType.IsRepeated() {
			typeName = RubyMessageType(fieldType.Element().Enum())
		} else {
			typeName = RubyMessageType(fieldType.Enum())
		}
		operation = typeName + ".deserialize"
	} else if isStringEncodedInt(fieldType.ProtoType()) || (fieldType.IsRepeated() && isStringEncodedInt(fieldType.Element().ProtoType())) {
		operation = "Integer"
	}

	key = fmt.Sprintf("hash[\"%s\"]", key)

	isOptional := fieldType.Field().Descriptor().GetProto3Optional()
	defaultVal := rubyProtoTypeValue(fieldType.Field(), fieldType)
	if defaultVal != nil && !isOptional && !fieldType.IsRepeated() {
		key = fmt.Sprintf("PbHelper::withDefault(%s, %s)", key, *defaultVal)
	} else if fieldType.IsRepeated() {
		key = fmt.Sprintf("PbHelper::withDefault(%s, [])", key)
	}

	extractor := ""
	if fieldType.IsRepeated() {
		extractor = fmt.Sprintf("%s.map{%s(_1)}", key, operation)
	} else {
		extractor = fmt.Sprintf("%s(%s)", operation, key)
	}

	if isOptional {
		return fmt.Sprintf("PbHelper::mapNil(%s) { %s }", key, extractor)
	}
	return extractor
}

func RubySetterFieldType(field pgs.Field) string {
	return rubyFieldType(field, methodTypeSetter)
}

func RubyInitializerFieldType(field pgs.Field) string {
	return rubyFieldType(field, methodTypeInitializer)
}

func maybeNillable(descriptorProto *descriptor.FieldDescriptorProto, rubyType string) string {
	if descriptorProto.GetProto3Optional() {
		return fmt.Sprintf("T.nilable(%s)", rubyType)
	}
	return rubyType
}

func isIdType(fieldDescriptor *descriptor.FieldDescriptorProto) bool {
	parentName := "placeholder"
	idType := elm.GetIdType(&parentName, fieldDescriptor)
	return idType != nil
}

func idTypeName(field pgs.FieldType) *string {
	parentName := field.Field().Message().Name().String()
	idType := elm.GetIdType(&parentName, field.Field().Descriptor())
	if idType != nil {
		typeName := strings.TrimPrefix(((string)(*idType)), "Ids.")
		val := typeName + "Id"
		return &val
	}
	return nil
}

func rubyFieldType(field pgs.Field, mt methodType) string {
	var rubyType string

	t := field.Type()

	if t.IsMap() {
		rubyType = rubyFieldMapType(field, t, mt)
	} else if t.IsRepeated() {
		rubyType = rubyFieldRepeatedType(field, t, mt)
	} else {
		idType := idTypeName(t)
		if idType != nil {
			rubyType = *idType
		} else {
			rubyType = rubyProtoTypeElem(field, t, mt)
		}
	}

	// initializer fields can be passed a `nil` value for all field types
	// messages are already wrapped so we skip those
	// if mt == methodTypeInitializer && (t.IsMap() || t.IsRepeated() || t.ProtoType() != pgs.MessageT) {
	//	return fmt.Sprintf("T.nilable(%s)", rubyType)
	//}

	// override the default behavior to be stricter, since we don't have old messages laying around

	return maybeNillable(field.Descriptor(), rubyType)
}

func rubyFieldMapType(field pgs.Field, ft pgs.FieldType, mt methodType) string {
	if mt == methodTypeSetter {
		return "::Google::Protobuf::Map"
	}
	key := rubyProtoTypeElem(field, ft.Key(), mt)
	value := rubyProtoTypeElem(field, ft.Element(), mt)
	return fmt.Sprintf("T::Hash[%s, %s]", key, value)
}

func rubyFieldRepeatedType(field pgs.Field, ft pgs.FieldType, mt methodType) string {
	// An enumerable/array is not accepted at the setter
	// See: https://github.com/protocolbuffers/protobuf/issues/4969
	// See: https://developers.google.com/protocol-buffers/docs/reference/ruby-generated#repeated-fields
	if mt == methodTypeSetter {
		return "::Google::Protobuf::RepeatedField"
	}
	value := rubyProtoTypeElem(field, ft.Element(), mt)
	return fmt.Sprintf("T::Array[%s]", value)
}

func RubyFieldValue(field pgs.Field) string {
	t := field.Type()
	if t.IsMap() {
		key := rubyMapType(t.Key())
		if t.Element().ProtoType() == pgs.MessageT {
			value := RubyMessageType(t.Element().Embed())
			return fmt.Sprintf("::Google::Protobuf::Map.new(%s, :message, %s)", key, value)
		}
		value := rubyMapType(t.Element())
		return fmt.Sprintf("::Google::Protobuf::Map.new(%s, %s)", key, value)
	} else if t.IsRepeated() {
		return "[]"
	}
	return *rubyProtoTypeValue(field, t)
}

func rubyProtoTypeElem(field pgs.Field, ft FieldType, mt methodType) string {
	pt := ft.ProtoType()
	idType := idTypeName(field.Type())
	if idType != nil {
		return *idType
	}

	if pt.IsInt() {
		return "Integer"
	}
	if pt.IsNumeric() {
		return "Float"
	}
	if pt == pgs.StringT || pt == pgs.BytesT {
		return "String"
	}
	if pt == pgs.BoolT {
		return "T::Boolean"
	}
	if pt == pgs.EnumT {
		return RubyMessageType(ft.Enum())
	}
	if pt == pgs.MessageT {
		inner := RubyMessageType(ft.Embed())
		if inner == "Google::Protobuf::Timestamp" {
			inner = "Time"
		}
		return inner
	}
	log.Panicf("Unsupported field type for field: %v\n", field.Name().String())
	return ""
}

func sPtr(s string) *string {
	return &s
}

func rubyProtoTypeValue(field pgs.Field, ft FieldType) *string {
	pt := ft.ProtoType()
	if pt.IsInt() {
		return sPtr("0")
	}
	if pt.IsNumeric() {
		return sPtr("0.0")
	}
	if pt == pgs.StringT || pt == pgs.BytesT {
		return sPtr("\"\"")
	}
	if pt == pgs.BoolT {
		return sPtr("false")
	}
	if pt == pgs.EnumT && ft.Enum() != nil {
		return sPtr(fmt.Sprintf("\"%s\"", ft.Enum().Values()[0].Name().String()))
	}
	if pt == pgs.MessageT && ft.IsEmbed() {
		inner := RubyMessageType(ft.Embed())
		if inner == "Google::Protobuf::Timestamp" {
			return sPtr("\"1970-01-01T00:00:00Z\"")
		}
	}
	if pt == pgs.MessageT {
		return sPtr("nil")
	}

	return nil
}

func rubyMapType(ft FieldType) string {
	switch ft.ProtoType() {
	case pgs.DoubleT:
		return ":double"
	case pgs.FloatT:
		return ":float"
	case pgs.Int64T:
		return ":int64"
	case pgs.UInt64T:
		return ":uint64"
	case pgs.Int32T:
		return ":int32"
	case pgs.Fixed64T:
		return ":fixed64"
	case pgs.Fixed32T:
		return ":fixed32"
	case pgs.BoolT:
		return ":bool"
	case pgs.StringT:
		return ":string"
	case pgs.BytesT:
		return ":bytes"
	case pgs.UInt32T:
		return ":uint32"
	case pgs.EnumT:
		return ":enum"
	case pgs.SFixed32:
		return ":sfixed32"
	case pgs.SFixed64:
		return ":sfixed64"
	case pgs.SInt32:
		return ":sint32"
	case pgs.SInt64:
		return ":sint64"
	}
	log.Panicf("Unsupported map field type\n")
	return ""
}

func RubyMethodParamType(method pgs.Method) string {
	return rubyMethodType(method.Input(), method.ClientStreaming())
}

func RubyMethodReturnType(method pgs.Method) string {
	return rubyMethodType(method.Output(), method.ServerStreaming())
}

func ShortEnumValName(enumVal pgs.EnumValue) string {
	lower_enum := enumVal.Enum().Name().String()
	lower_val := ((pgs.Name)(strings.ToLower(enumVal.Name().String()))).UpperCamelCase().String()
	return strings.TrimPrefix(lower_val, lower_enum)
}

func rubyMethodType(message pgs.Message, streaming bool) string {
	t := RubyMessageType(message)
	if streaming {
		return fmt.Sprintf("T::Enumerable[%s]", t)
	}
	return t
}
