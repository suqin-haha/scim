package scim

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

var metaSchema schema

func init() {
	json.Unmarshal([]byte(rawMetaSchema), &metaSchema)
}

// NewSchemaFromFile reads the file from given filepath and returns a validated schema if no errors take place.
func NewSchemaFromFile(filepath string) (Schema, error) {
	raw, err := ioutil.ReadFile(filepath)
	if err != nil {
		return Schema{}, err
	}

	return NewSchemaFromBytes(raw)
}

// NewSchemaFromString returns a validated schema if no errors take place.
func NewSchemaFromString(s string) (Schema, error) {
	return NewSchemaFromBytes([]byte(s))
}

// NewSchemaFromBytes returns a validated schema if no errors take place.
func NewSchemaFromBytes(raw []byte) (Schema, error) {
	err := metaSchema.validate(raw)
	if err != nil {
		return Schema{}, err
	}

	var schema schema
	json.Unmarshal(raw, &schema)

	return Schema{schema: schema}, nil
}

type Schema struct {
	schema schema
}

// Schema specifies the defined attribute(s) and their characteristics (mutability, returnability, etc). For every
// schema URI used in a resource object, there is a corresponding "Schema" resource.
// INFO: RFC7643 - 7.  Schema Definition
type schema struct {
	// ID is the unique URI of the schema. REQUIRED.
	ID string
	// Name is the schema's human-readable name. OPTIONAL.
	Name string
	// Description is the schema's human-readable description.  OPTIONAL.
	Description string
	// Attributes is a collection of a complex type that defines service provider attributes and their qualities.
	Attributes attributes
}

// validate unmarshals the given bytes and validates it based on the schema.
func (s schema) validate(raw []byte) error {
	var m interface{}
	err := json.Unmarshal(raw, &m)
	if err != nil {
		return err
	}
	return s.Attributes.validate(m)
}

// attribute is a complex type that defines service provider attributes and their qualities via the following set of
// sub-attributes.
// INFO: RFC7643 - 7.  Schema Definition
type attribute struct {
	// Name is the attribute's name.
	Name string
	// Type is the attribute's data type. Valid values are "string", "boolean", "decimal", "integer", "dateTime",
	// "reference", and "complex".  When an attribute is of type "complex", there SHOULD be a corresponding schema
	// attribute "subAttributes" defined, listing the sub-attributes of the attribute.
	Type attributeType
	// SubAttributes defines a set of sub-attributes when an attribute is of type "complex". "subAttributes" has the
	// same schema sub-attributes as "attributes".
	SubAttributes attributes
	// MultiValued is a boolean value indicating the attribute's plurality.
	MultiValued bool
	// Description is the attribute's human-readable description. When applicable, service providers MUST specify the
	// description.
	Description string
	// Required is a boolean value that specifies whether or not the attribute is required.
	Required bool
	// CanonicalValues is a collection of suggested canonical values that MAY be used (e.g., "work" and "home").
	// OPTIONAL.
	CanonicalValues []string
	// CaseExact is a boolean value that specifies whether or not a string attribute is case sensitive.
	CaseExact bool
	// Mutability is a single keyword indicating the circumstances under which the value of the attribute can be
	// (re)defined.
	Mutability attributeMutability
	// Returned is a single keyword that indicates when an attribute and associated values are returned in response to
	// a GET request or in response to a PUT, POST, or PATCH request.
	Returned attributeReturned
	// Uniqueness is a single keyword value that specifies how the service provider enforces uniqueness of attribute
	// values.
	Uniqueness attributeUniqueness
	// ReferenceTypes is a multi-valued array of JSON strings that indicate the SCIM resource types that may be
	// referenced.
	ReferenceTypes []string
}

func (a attribute) validate(i interface{}) error {
	// validate required
	if i == nil {
		if a.Required {
			return fmt.Errorf("cannot find required value %s", strings.ToLower(a.Name))
		}
		return nil
	}

	if a.MultiValued {
		arr, ok := i.([]interface{})
		if !ok {
			return fmt.Errorf("cannot convert %v to a slice", i)
		}

		// empty array = omitted/nil
		if len(arr) == 0 && a.Required {
			return fmt.Errorf("required array is empty")
		}

		for _, sub := range arr {
			if err := a.validateSingular(sub); err != nil {
				return err
			}
		}
		return nil
	}

	return a.validateSingular(i)
}

func (a attribute) validateSingular(i interface{}) error {
	switch a.Type {
	case attributeTypeBoolean:
		_, ok := i.(bool)
		if !ok {
			return fmt.Errorf("cannot convert %v to type %s", i, a.Type)
		}
	case attributeTypeComplex:
		if err := a.SubAttributes.validate(i); err != nil {
			return err
		}
	case attributeTypeString:
		_, ok := i.(string)
		if !ok {
			return fmt.Errorf("cannot convert %v to type %s", i, a.Type)
		}
	default:
		return fmt.Errorf("not implemented/invalid type: %v", a.Type)
	}
	return nil
}

type attributes []attribute

func (as attributes) validate(i interface{}) error {
	c, ok := i.(map[string]interface{})
	if !ok {
		return fmt.Errorf("cannot convert %v to type complex", i)
	}

	for _, attribute := range as {
		// validate duplicate
		var hit interface{}
		var found bool
		for k, v := range c {
			if strings.EqualFold(attribute.Name, k) {
				if found {
					return fmt.Errorf("duplicate key: %s", strings.ToLower(k))
				}
				found = true
				hit = v
			}
		}

		if err := attribute.validate(hit); err != nil {
			return err
		}
	}
	return nil
}

type attributeType string

const (
	attributeTypeBinary    attributeType = "binary"
	attributeTypeBoolean                 = "boolean"
	attributeTypeComplex                 = "complex"
	attributeTypeDateTime                = "dateTime"
	attributeTypeDecimal                 = "decimal"
	attributeTypeInteger                 = "integer"
	attributeTypeReference               = "reference"
	attributeTypeString                  = "string"
)

type attributeMutability string

const (
	attributeMutabilityImmutable attributeMutability = "immutable"
	attributeMutabilityReadOnly                      = "readOnly"
	attributeMutabilityReadWrite                     = "readWrite"
	attributeMutabilityWriteOnly                     = "writeOnly"
)

type attributeReturned string

const (
	attributeReturnedAlways  attributeReturned = "always"
	attributeReturnedDefault                   = "default"
	attributeReturnedNever                     = "never"
	attributeReturnedRequest                   = "request"
)

type attributeUniqueness string

const (
	attributeUniquenessGlobal attributeUniqueness = "global"
	attributeUniquenessNone                       = "none"
	attributeUniquenessServer                     = "server"
)