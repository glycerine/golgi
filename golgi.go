package golgi

import (
	"github.com/chewxy/hm"
	"github.com/pkg/errors"
	G "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

type Term interface {
	Type() hm.Type
}

// Layer represents a neural network layer.
// λ
type Layer interface {
	// σ - The weights are the "free variables" of a function
	Model() G.Nodes

	// Fwd represents the forward application of inputs
	// x.t
	Fwd(x *G.Node) (*G.Node, error)

	// meta stuff. This stuff is just placholder for more advanced things coming

	Term

	// TODO
	Shape() tensor.Shape

	// Name gives the layer's name
	Name() string

	// Serialization stuff

	// Describe returns the protobuf definition of a Layer that conforms to the ONNX standard
	Describe() // some protobuf things TODO
}

func Apply(a, b Term) (Term, error) {
	panic("STUBBED")
}

var (
	_ Layer = (*Composition)(nil)
)

// Composition represents a composition of functions
type Composition struct {
	a, b Term // can be thunk, Layer or *G.Node

	// store returns
	retVal   *G.Node
	retType  hm.Type
	retShape tensor.Shape
}

func Compose(a, b Term) (retVal *Composition, err error) {
	return &Composition{
		a: a,
		b: b,
	}, nil
}

// ComposeSeq creates a composition with the inputs written in left to right order
//
//
// The equivalent in F# is |>. The equivalent in Haskell is (flip (.))
func ComposeSeq(layers ...Term) (retVal *Composition, err error) {
	inputs := len(layers)
	switch inputs {
	case 0:
		return nil, errors.Errorf("Expected more than 1 input")
	case 1:
		// ?????
		return nil, errors.Errorf("Expected more than 1 input")
	}
	l := layers[0]
	for _, next := range layers[1:] {
		if l, err = Compose(l, next); err != nil {
			return nil, err
		}
	}
	return l.(*Composition), nil
}

func (l *Composition) Fwd(input *G.Node) (output *G.Node, err error) {
	if l.retVal != nil {
		return l.retVal, nil
	}
	var x *G.Node
	var layer Layer
	switch at := l.a.(type) {
	case *G.Node:
		x = at
	case consThunk:
		if layer, err = at.LayerCons(input, at.Opts...); err != nil {
			goto next
		}
		l.a = layer
		x, err = layer.Fwd(input)
	case Layer:
		x, err = at.Fwd(input)
	default:
		return nil, errors.Errorf("Fwd of Composition not handled for a of %T", l.a)
	}
next:
	if err != nil {
		return nil, errors.Wrapf(err, "Happened while doing a of Composition %v", l)
	}

	switch bt := l.b.(type) {
	case *G.Node:
		return nil, errors.New("Cannot Fwd when b is a *Node")
	case consThunk:
		if layer, err = bt.LayerCons(x, bt.Opts...); err != nil {
			return nil, errors.Wrapf(err, "Happened while doing b of Composition %v", l)
		}
		l.b = layer
		output, err = layer.Fwd(x)
	case Layer:
		output, err = bt.Fwd(x)
	default:
		return nil, errors.Errorf("Fwd of Composition not handled for b of %T", l.b)
	}
	return
}

func (l *Composition) Model() (retVal G.Nodes) {
	if a, ok := l.a.(Layer); ok {
		return append(a.Model(), l.b.(Layer).Model()...)
	}
	return l.b.(Layer).Model()
}

func (l *Composition) Type() hm.Type { return l.retType }

func (l *Composition) Shape() tensor.Shape { return l.retShape }

func (l *Composition) Name() string { panic("STUB") }

func (l *Composition) Describe() { panic("STUB") }
