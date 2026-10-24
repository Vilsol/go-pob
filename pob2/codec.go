package pob2

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Unparsed keeps what a struct does not name, so a round trip loses nothing: attributes, child elements, and the
// order the children came in (an element name, "#text" or "#other" per kid).
type Unparsed struct {
	ExtraAttrib map[string]string
	ExtraKids   []*Node
	Order       []string
}

// Nil is a value PoB saves as tostring(v) where v can be nil, so the file may say "nil".
type Nil[T int | float64 | bool] struct {
	V     T
	IsNil bool
}

type attrValue interface {
	decodeAttr(s string) error
	encodeAttr() string
}

func (n *Nil[T]) decodeAttr(s string) error {
	if s == "nil" {
		*n = Nil[T]{IsNil: true}
		return nil
	}
	n.IsNil = false
	return decodeScalar(reflect.ValueOf(&n.V).Elem(), s)
}

func (n *Nil[T]) encodeAttr() string {
	if n.IsNil {
		return "nil"
	}
	return encodeScalar(reflect.ValueOf(n.V))
}

const (
	orderText  = "#text"
	orderOther = "#other"
)

var (
	unparsedType  = reflect.TypeFor[Unparsed]()
	attrValueType = reflect.TypeFor[attrValue]()
)

type fieldSpec struct {
	index  int
	name   string
	kind   string
	always bool
}

var specCache sync.Map

func specsOf(t reflect.Type) []fieldSpec {
	if s, ok := specCache.Load(t); ok {
		return s.([]fieldSpec)
	}
	var out []fieldSpec
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Type == unparsedType {
			out = append(out, fieldSpec{index: i, kind: "unparsed"})
			continue
		}
		tag, ok := f.Tag.Lookup("pob")
		if !ok {
			continue
		}
		parts := strings.Split(tag, ",")
		spec := fieldSpec{index: i, name: parts[0], kind: "elem"}
		for _, p := range parts[1:] {
			switch p {
			case "attr", "text":
				spec.kind = p
			case "always":
				spec.always = true
			}
		}
		out = append(out, spec)
	}
	specCache.Store(t, out)
	return out
}

func luaNumber(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	case math.IsNaN(v) && math.Signbit(v):
		return "-nan"
	case math.IsNaN(v):
		return "nan"
	}
	return strconv.FormatFloat(v, 'g', 14, 64)
}

func decodeScalar(v reflect.Value, s string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int:
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("not an integer: %w", err)
		}
		v.SetInt(int64(n))
	case reflect.Float64:
		if s == "-nan" {
			v.SetFloat(math.Copysign(math.NaN(), -1))
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("not a number: %w", err)
		}
		v.SetFloat(f)
	case reflect.Bool:
		switch s {
		case "true":
			v.SetBool(true)
		case "false":
			v.SetBool(false)
		default:
			return fmt.Errorf("%q is not true or false", s)
		}
	default:
		return fmt.Errorf("unsupported attribute type %s", v.Type())
	}
	return nil
}

func encodeScalar(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Float64:
		return luaNumber(v.Float())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	}
	panic("unsupported attribute type " + v.Type().String())
}

func decodeAttr(v reflect.Value, s string) error {
	if v.Kind() == reflect.Pointer {
		v.Set(reflect.New(v.Type().Elem()))
		v = v.Elem()
	}
	if v.Addr().Type().Implements(attrValueType) {
		return v.Addr().Interface().(attrValue).decodeAttr(s)
	}
	return decodeScalar(v, s)
}

func encodeAttr(v reflect.Value) (string, bool) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	if v.CanAddr() && v.Addr().Type().Implements(attrValueType) {
		return v.Addr().Interface().(attrValue).encodeAttr(), true
	}
	if reflect.PointerTo(v.Type()).Implements(attrValueType) {
		c := reflect.New(v.Type())
		c.Elem().Set(v)
		return c.Interface().(attrValue).encodeAttr(), true
	}
	return encodeScalar(v), true
}

func decodeNode(n *Node, out any) error {
	return decodeInto(n, reflect.ValueOf(out).Elem())
}

func decodeInto(n *Node, v reflect.Value) error {
	specs := specsOf(v.Type())
	var un *Unparsed
	used := map[string]bool{}
	byElem := map[string]fieldSpec{}
	var text *fieldSpec
	for i, s := range specs {
		switch s.kind {
		case "unparsed":
			un = v.Field(s.index).Addr().Interface().(*Unparsed)
		case "attr":
			used[s.name] = true
			if val, ok := n.Attrib[s.name]; ok {
				if err := decodeAttr(v.Field(s.index), val); err != nil {
					return fmt.Errorf("<%s %s=%q>: %w", n.Elem, s.name, val, err)
				}
			}
		case "text":
			text = &specs[i]
		case "elem":
			byElem[s.name] = s
		}
	}
	if un == nil {
		return fmt.Errorf("%s has no Unparsed field", v.Type())
	}
	for k, val := range n.Attrib {
		if !used[k] {
			if un.ExtraAttrib == nil {
				un.ExtraAttrib = map[string]string{}
			}
			un.ExtraAttrib[k] = val
		}
	}
	seenText := false
	for _, kid := range n.Kids {
		if kid.Elem == "" {
			if text != nil && !seenText {
				seenText = true
				v.Field(text.index).SetString(kid.Text)
				un.Order = append(un.Order, orderText)
				continue
			}
			un.ExtraKids = append(un.ExtraKids, kid)
			un.Order = append(un.Order, orderOther)
			continue
		}
		s, ok := byElem[kid.Elem]
		if ok && v.Field(s.index).Kind() != reflect.Slice && seenElem(un.Order, kid.Elem) {
			ok = false
		}
		if !ok {
			un.ExtraKids = append(un.ExtraKids, kid)
			un.Order = append(un.Order, orderOther)
			continue
		}
		f := v.Field(s.index)
		var target reflect.Value
		switch f.Kind() {
		case reflect.Slice:
			f.Set(reflect.Append(f, reflect.Zero(f.Type().Elem())))
			target = f.Index(f.Len() - 1)
		case reflect.Pointer:
			f.Set(reflect.New(f.Type().Elem()))
			target = f.Elem()
		default:
			target = f
		}
		if err := decodeInto(kid, target); err != nil {
			return err
		}
		un.Order = append(un.Order, kid.Elem)
	}
	return nil
}

func seenElem(order []string, elem string) bool {
	for _, o := range order {
		if o == elem {
			return true
		}
	}
	return false
}

func encodeNode(elem string, in any) (*Node, error) {
	return encodeFrom(elem, reflect.ValueOf(in))
}

func encodeFrom(elem string, v reflect.Value) (*Node, error) {
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	n := &Node{Elem: elem, Attrib: map[string]string{}}
	specs := specsOf(v.Type())
	var un Unparsed
	queues := map[string][]reflect.Value{}
	var fieldOrder []string
	var text *fieldSpec
	for i, s := range specs {
		f := v.Field(s.index)
		switch s.kind {
		case "unparsed":
			un = f.Interface().(Unparsed)
		case "attr":
			if val, ok := encodeAttr(f); ok {
				n.Attrib[s.name] = val
			}
		case "text":
			text = &specs[i]
		case "elem":
			fieldOrder = append(fieldOrder, s.name)
			switch f.Kind() {
			case reflect.Slice:
				for j := 0; j < f.Len(); j++ {
					queues[s.name] = append(queues[s.name], f.Index(j))
				}
			case reflect.Pointer:
				if !f.IsNil() {
					queues[s.name] = append(queues[s.name], f)
				}
			default:
				queues[s.name] = append(queues[s.name], f)
			}
		}
	}
	for k, val := range un.ExtraAttrib {
		n.Attrib[k] = val
	}
	textDone := false
	if text != nil && !seenElem(un.Order, orderText) {
		textDone = true
		if s := v.Field(text.index).String(); s != "" || text.always {
			n.Kids = append(n.Kids, &Node{Text: s})
		}
	}
	others := un.ExtraKids
	emit := func(name string) error {
		q := queues[name]
		if len(q) == 0 {
			return nil
		}
		queues[name] = q[1:]
		kid, err := encodeFrom(name, q[0])
		if err != nil {
			return err
		}
		n.Kids = append(n.Kids, kid)
		return nil
	}
	for _, o := range un.Order {
		switch o {
		case orderText:
			if text != nil && !textDone {
				textDone = true
				if s := v.Field(text.index).String(); s != "" || text.always {
					n.Kids = append(n.Kids, &Node{Text: s})
				}
			}
		case orderOther:
			if len(others) > 0 {
				n.Kids = append(n.Kids, others[0])
				others = others[1:]
			}
		default:
			if err := emit(o); err != nil {
				return nil, err
			}
		}
	}
	for _, name := range fieldOrder {
		for len(queues[name]) > 0 {
			if err := emit(name); err != nil {
				return nil, err
			}
		}
	}
	n.Kids = append(n.Kids, others...)
	return n, nil
}
