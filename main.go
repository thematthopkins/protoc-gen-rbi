package main

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
	"log"
	"regexp"
	"strings"
	"text/template"

	"github.com/coinbase/protoc-gen-rbi/ruby_types"
	"github.com/thematthopkins/elm-protobuf/pkg/forwardextensions"

	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"
)

var (
	validRubyField = regexp.MustCompile(`\A[a-z][A-Za-z0-9_]*\z`)
)

type rbiModule struct {
	*pgs.ModuleBase
	ctx                pgsgo.Context
	tpl                *template.Template
	serviceTpl         *template.Template
	httpServiceTpl     *template.Template
	validatorTpl       *template.Template
	hideCommonMethods  bool
	useAbstractMessage bool
}

func (m *rbiModule) HideCommonMethods() bool {
	return m.hideCommonMethods
}

func (m *rbiModule) UseAbstractMessage() bool {
	return m.useAbstractMessage
}

func RBI() *rbiModule { return &rbiModule{ModuleBase: &pgs.ModuleBase{}} }

func (m *rbiModule) InitContext(c pgs.BuildContext) {
	m.ModuleBase.InitContext(c)
	m.ctx = pgsgo.InitContext(c.Parameters())

	hideCommonMethods, err := m.ctx.Params().BoolDefault("hide_common_methods", false)
	if err != nil {
		log.Panicf("Bad parameter: hide_common_methods\n")
	}
	m.hideCommonMethods = hideCommonMethods

	useAbstractMessage, err := m.ctx.Params().BoolDefault("use_abstract_message", false)
	if err != nil {
		log.Panicf("Bad parameter: use_abstract_message\n")
	}
	m.useAbstractMessage = useAbstractMessage

	funcs := map[string]interface{}{
		"increment":                    m.increment,
		"optional":                     m.optional,
		"optionalOneOf":                m.optionalOneOf,
		"willGenerateInvalidRuby":      m.willGenerateInvalidRuby,
		"validators":                   ruby_types.Validators,
		"oneOfValidators":              ruby_types.OneOfValidators,
		"readableLabel":                ruby_types.ReadableLabel,
		"rubyPackage":                  ruby_types.RubyPackage,
		"rubyMessageType":              ruby_types.RubyMessageType,
		"fieldEncoder":                 ruby_types.FieldEncoder,
		"fieldDecoder":                 ruby_types.FieldDecoder,
		"rubyGetterFieldType":          ruby_types.RubyGetterFieldType,
		"rubySetterFieldType":          ruby_types.RubySetterFieldType,
		"rubyInitializerFieldType":     ruby_types.RubyInitializerFieldType,
		"rubyFieldValue":               ruby_types.RubyFieldValue,
		"rubyMethodParamType":          ruby_types.RubyMethodParamType,
		"rubyMethodReturnType":         ruby_types.RubyMethodReturnType,
		"shortEnumValName":             ruby_types.ShortEnumValName,
		"hideCommonMethods":            m.HideCommonMethods,
		"useAbstractMessage":           m.UseAbstractMessage,
		"isAuthenticatedServiceMethod": m.IsAuthenticatedServiceMethod,
	}

	m.tpl = template.Must(template.New("rbi").Funcs(funcs).Parse(tpl))
	m.serviceTpl = template.Must(template.New("rbiService").Funcs(funcs).Parse(serviceTpl))
	m.httpServiceTpl = template.Must(template.New("rbiHttpService").Funcs(funcs).Parse(httpServiceTpl))
	m.validatorTpl = template.Must(template.New("validator").Funcs(funcs).Parse(validatorTpl))
}

func (m *rbiModule) IsAuthenticatedServiceMethod(method pgs.Method) bool {
	return !proto.HasExtension(method.Descriptor().Options, forwardextensions.E_Unauthenticated)
}

func (m *rbiModule) Name() string { return "rbi" }

func (m *rbiModule) Execute(targets map[string]pgs.File, pkgs map[string]pgs.Package) []pgs.Artifact {
	for _, t := range targets {
		m.generate(t)

		grpc, err := m.ctx.Params().BoolDefault("grpc", true)
		if err != nil {
			log.Panicf("Bad parameter: grpc\n")
		}

		if len(t.Services()) > 0 && grpc {
			m.generateServices(t)
		}
		m.generateHttpServices(t)
		m.generateValidator(t)
	}
	return m.Artifacts()
}

func (m *rbiModule) generate(f pgs.File) {
	op := "lib/" + strings.TrimSuffix(f.InputPath().String(), ".proto") + "_pb.rb"
	m.AddGeneratorTemplateFile(op, m.tpl, f)
}

func (m *rbiModule) generateServices(f pgs.File) {
	op := strings.TrimSuffix(f.InputPath().String(), ".proto") + "_services_pb.rbi"
	m.AddGeneratorTemplateFile(op, m.serviceTpl, f)
}

func (m *rbiModule) generateHttpServices(f pgs.File) {
	op := "lib/" + strings.TrimSuffix(f.InputPath().String(), ".proto") + "_pb_service.rb"
	m.AddGeneratorTemplateFile(op, m.httpServiceTpl, f)
}

func (m *rbiModule) generateValidator(f pgs.File) {
	op := "lib/" + strings.TrimSuffix(f.InputPath().String(), ".proto") + "_pb_validator.rb"
	m.AddGeneratorTemplateFile(op, m.validatorTpl, f)
}

func (m *rbiModule) increment(i int) int {
	return i + 1
}

func (m *rbiModule) optional(field pgs.Field) bool {
	return field.Descriptor().GetProto3Optional()
}

func (m *rbiModule) optionalOneOf(oneOf pgs.OneOf) bool {
	return len(oneOf.Fields()) == 1 && oneOf.Fields()[0].Descriptor().GetProto3Optional()
}

func (m *rbiModule) willGenerateInvalidRuby(fields []pgs.Field) bool {
	for _, field := range fields {
		if !validRubyField.MatchString(string(field.Name())) {
			return true
		}
	}
	return false
}

func main() {
	supportOptionalKeyward := (uint64)(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
	pgs.Init(
		pgs.DebugEnv("DEBUG"),
		pgs.SupportedFeatures(&supportOptionalKeyward),
	).RegisterModule(
		RBI(),
	).RegisterPostProcessor(
		pgsgo.GoFmt(),
	).Render()
}

const tpl = `# Code generated by protoc-gen-rbi. DO NOT EDIT.
# source: {{ .InputPath }}
# typed: strict

module {{ rubyPackage .File }}
end


module PbHelper
  extend T::Sig

  sig {
    params(value: T.untyped, defaultValue: T.untyped).returns(T.untyped)
  }
  def self.withDefault(value, defaultValue)
    value.nil? ? defaultValue : value
  end

  sig {
    params(value: T.untyped, blk: T.proc.params(v: T.untyped).returns(T.untyped)).returns(T.untyped)
  }
  def self.mapNil(value, &blk)
    value.nil? ? nil : blk.call(value)
  end
end

{{ range .Enums }}
class {{ rubyMessageType . }} < T::Enum
  extend T::Sig

  enums do{{ range .Values }}
    {{ shortEnumValName . }} = new("{{ .Name }}"){{ end }}
  end

  class Attr < EnumAttrBase
    sig { override.params(value: T.untyped).returns(T.nilable({{ rubyMessageType . }})) }
    def deserialize(value)
      {{ rubyMessageType . }}.deserialize(value) unless value.nil?
    end
  end

  sig { returns(String) }
  def to_s
    serialize
  end
end
{{ end }}

# T::Structs require forward declarations of reference classes
# dynamically declaring them like this hides the duplicate 
# declaration from sorbet, and keeps ruby happy about 
# classes existing before they're referenced
{{ range .AllMessages }}{{ rubyMessageType . }} = Class.new(T::Struct)
{{ end }}
{{ range .AllMessages }}
{{ range .Enums }}
class {{ rubyMessageType . }} < T::Enum
  extend T::Sig

  enums do{{ range .Values }}
    {{ shortEnumValName . }} = new("{{ .Name }}"){{ end }}
  end

  class Attr < EnumAttrBase
    sig { override.params(value: T.untyped).returns(T.nilable({{ rubyMessageType . }})) }
    def deserialize(value)
      {{ rubyMessageType . }}.deserialize(value) unless value.nil?
    end
  end

  sig { returns(String) }
  def to_s
    serialize
  end
end
{{ end }}
class {{ rubyMessageType . }} < T::Struct
  extend T::Sig
  include T::Props::Serializable
  include T::Struct::ActsAsComparable
{{ range .Fields }}{{ if not (.InRealOneOf) }}
  const :{{ .Name }}, {{ rubyGetterFieldType . }}
{{ end }}{{ end }}{{ range .OneOfs }}{{ if not (optionalOneOf .) }}
  module {{.Name.UpperCamelCase}}; end
  const :{{ .Name.LowerSnakeCase }}, {{ .Name.UpperCamelCase }}
{{ end }}{{ end }}{{ range .OneOfs }}{{ if not (optionalOneOf .) }}
  module {{ .Name.UpperCamelCase }}{{ $oneOfName := .Name.UpperCamelCase }}
    extend T::Sig
    extend T::Helpers
    include T::Props::Serializable
    sealed!

    sig { params(_strict: T::Boolean).returns(T::Hash[String, T.untyped]) }
    def serialize(_strict = true)
        case self
        {{ range .Fields }}when {{ .Name.UpperCamelCase }}
            { "{{ .Name.LowerCamelCase }}": {{ fieldEncoder "value" (.Type) }} }
        {{ end }}  else
            T.absurd(self)
        end
    end

    sig { params(hash: T::Hash[String, T.untyped]).returns({{ .Name.UpperCamelCase }}) }
    def self.from_hash(hash)
      {{ range .Fields }}if hash["{{ .Name.LowerCamelCase }}"] then
        return {{ .Name.UpperCamelCase }}.new({{ fieldDecoder (.Name.LowerCamelCase.String) .Type }})
      end
      {{ end }}
      raise "Expected one of, but none were set"
    end

    {{ range .Fields }}
    class {{ .Name.UpperCamelCase }}
      include {{ $oneOfName }}
      extend T::Sig

      sig { params(value: {{ rubyGetterFieldType . }}).void }
      def initialize(value)
        @value = T.let(value, {{ rubyGetterFieldType . }})
      end

      sig { returns(String) }
      def to_s
        value.to_s
      end

      sig { params(other: T.untyped).returns(T::Boolean) }
      def ==(other)
        self.class == other.class &&
          @value == other.value
      end

      sig { returns({{ rubyGetterFieldType . }}) }
      attr_reader :value
    end
    {{ end }}
  end
{{ end }}{{ end }}
  sig { returns(String) }
  def to_json
    serialize.to_json
  end

  sig { params(_strict: T::Boolean).returns(T::Hash[String, T.untyped]) }
  def serialize(_strict = true)
    serialize_result = { {{ range .Fields }}{{ if not (.InRealOneOf) }}
      "{{ .Name.LowerCamelCase }}": {{ fieldEncoder (.Name.LowerSnakeCase.String) (.Type) }},{{ end }}{{ end }}
    }
    {{ range .OneOfs }}{{ if not (optionalOneOf .) }}serialize_result = serialize_result.merge({{ .Name.LowerSnakeCase }}.serialize )
    {{ end }}{{ end }}serialize_result
  end

  sig { params(contents: String).returns({{ rubyMessageType . }}) }
  def self.decode_json(contents)
    json_obj = JSON.parse(contents)
    from_hash(json_obj)
  end

  sig { params(hash: T::Hash[String, T.untyped]).returns({{ rubyMessageType . }}) }
  def self.from_hash(hash)
    new({{ range .Fields }}{{ if not (.InRealOneOf) }}
      {{ .Name }}: {{ fieldDecoder (.Name.LowerCamelCase.String) .Type }},{{end}}{{ end }}{{ range .OneOfs }}{{ if not (optionalOneOf .) }}
      {{ .Name.LowerCamelCase }}: {{ .Name.UpperCamelCase }}.from_hash(hash){{ end }}{{ end }}
    )
  end
end
{{ end }}
`

const serviceTpl = `# Code generated by protoc-gen-rbi. DO NOT EDIT.
# source: {{ .InputPath }}
# typed: strict
{{ range .Services }}
module {{ rubyPackage .File }}::{{ .Name }}
  class Service
    include ::GRPC::GenericService
  end

  class Stub < ::GRPC::ClientStub
    sig do
      params(
        host: String,
        creds: T.any(::GRPC::Core::ChannelCredentials, Symbol),
        kw: T.untyped,
      ).void
    end
    def initialize(host, creds, **kw)
    end{{ range .Methods }}

    sig do
      params(
        request: {{ rubyMethodParamType . }}
      ).returns({{ rubyMethodReturnType . }})
    end
    def {{ .Name.LowerSnakeCase }}(request)
    end{{ end }}
  end
end
{{ end }}`

const httpServiceTpl = `# typed: strict
# frozen_string_literal: true

# Code generated by protoc-gen-rbi. DO NOT EDIT.

module {{ rubyPackage .File }}Service
  extend T::Sig

  sig do
    params(mapper: ActionDispatch::Routing::Mapper).void
  end
  def self.add_routes(mapper)
{{ range $Service := .Services }}    # {{.Name}} routes{{ range .Methods }}
    mapper.post "/{{$Service.Name.LowerSnakeCase}}/{{.Name.LowerSnakeCase}}", to: "{{$Service.Name.LowerSnakeCase}}#{{.Name.LowerSnakeCase}}_stub"{{ end }}

{{ end }}
  end

{{ range .Services }}
  class {{.Name}}Base < ApiController
    extend T::Sig

    abstract!

    # rails routing doesn't like to route to super classes
    # automatically define our stubs in the sub classes, so it can 
    # find them
    sig do
      params(subclass: T::Class[T.untyped]).void
    end
    def self.inherited(subclass)
{{ range .Methods }}      subclass.define_method("{{ .Name.LowerSnakeCase }}_stub") do super() end
{{ end }}
    end

{{ range .Methods }}
    sig { void }
    def {{ .Name.LowerSnakeCase }}_stub
      params = {{ rubyMethodParamType . }}.decode_json(request.body.read){{ if isAuthenticatedServiceMethod . }}
      user = UserRm.find(UserSession.new(self).user_id)
      result = {{ .Name.LowerSnakeCase }}(user, params)
{{ else }}
      result = {{ .Name.LowerSnakeCase }}(params)
{{ end }}      render json: result.to_json
    end

    sig do
      abstract.params(
{{ if isAuthenticatedServiceMethod . }}        user: UserRm,
{{ end }}        p: {{ rubyMethodParamType . }}
      ).returns({{ rubyMethodReturnType . }})
    end
    def {{ .Name.LowerSnakeCase }}({{ if isAuthenticatedServiceMethod . }}user, {{end}}p)
    end
{{ end }}
  end
{{ end }}
end
`

const validatorTpl = `# typed: strict
# frozen_string_literal: true

# Code generated by protoc-gen-rbi. DO NOT EDIT.

{{ range .AllMessages }}

{{ $message := . }}
class {{ rubyMessageType . }}Validators
  extend T::Sig
  include Validation


  include T::Props::Serializable
  include T::Struct::ActsAsComparable
{{ range .Fields }}{{ if not (.InRealOneOf) }}
  sig {
    returns ValidatableField[{{ rubyMessageType $message }}, T.nilable(String)]
  }
  def {{ .Name.LowerSnakeCase }}
    ValidatableField.new(
      label: "{{ readableLabel .Name }}",
      getter: ->(message) { message.{{ .Name }} },
      setter: ->(message, field) { message.{{ .Name }} = field },
      validators: [
        {{ range validators . }}{{ . }}.new,
        {{ end }}
      ]
    )
  end
{{ end }}{{ end }}{{ range .OneOfs }}{{ if not (optionalOneOf .) }}
  sig {
    returns ValidatableField[{{ rubyMessageType $message }}, T.nilable(String)]
  }
  def {{ .Name.LowerSnakeCase }}
    ValidatableField.new(
      label: "{{ readableLabel .Name }}",
      getter: ->(message) { message.{{ .Name.LowerSnakeCase }} },
      setter: ->(message, field) { message.{{ .Name.LowerSnakeCase }} = field },
      validators: [
        {{ range oneOfValidators . }}{{ . }}.new,
        {{ end }}]
    )
  end
{{ end }}{{ end }}

  sig {
      returns(T::Array[[String, MessageValidator[{{ rubyMessageType $message }}]]])
  }
  def all_model_validators
    [{{ range .Fields }}{{ if not (.InRealOneOf) }}
        ["{{ readableLabel .Name }}", {{ .Name.LowerSnakeCase }}.message_validator],{{ end }}{{ end }}
{{ range .OneOfs }}{{ if not (optionalOneOf .) }}
        ["{{ readableLabel .Name }}", {{ .Name.LowerSnakeCase }}.message_validator],{{ end }}{{ end }}
    ]
  end
end
{{end}}
`
