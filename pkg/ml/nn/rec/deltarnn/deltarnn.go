// Copyright 2019 spaGO Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package deltarnn

import (
	"github.com/nlpodyssey/spago/pkg/mat"
	"github.com/nlpodyssey/spago/pkg/ml/ag"
	"github.com/nlpodyssey/spago/pkg/ml/nn"
	"log"
)

var (
	_ nn.Model     = &Model{}
	_ nn.Processor = &Processor{}
)

// Model contains the serializable parameters.
type Model struct {
	W     nn.Param `type:"weights"`
	WRec  nn.Param `type:"weights"`
	B     nn.Param `type:"biases"`
	BPart nn.Param `type:"biases"`
	Alpha nn.Param `type:"weights"`
	Beta1 nn.Param `type:"weights"`
	Beta2 nn.Param `type:"weights"`
}

// New returns a new model with parameters initialized to zeros.
func New(in, out int) *Model {
	return &Model{
		W:     nn.NewParam(mat.NewEmptyDense(out, in)),
		WRec:  nn.NewParam(mat.NewEmptyDense(out, out)),
		B:     nn.NewParam(mat.NewEmptyVecDense(out)),
		BPart: nn.NewParam(mat.NewEmptyVecDense(out)),
		Alpha: nn.NewParam(mat.NewEmptyVecDense(out)),
		Beta1: nn.NewParam(mat.NewEmptyVecDense(out)),
		Beta2: nn.NewParam(mat.NewEmptyVecDense(out)),
	}
}

// State represent a state of the DeltaRNN recurrent network.
type State struct {
	D1 ag.Node
	D2 ag.Node
	C  ag.Node
	P  ag.Node
	Y  ag.Node
}

// Processor implements the nn.Processor interface for a DeltaRNN Model.
type Processor struct {
	nn.BaseProcessor
	States []*State
}

// NewProc returns a new processor to execute the forward step.
func (m *Model) NewProc(ctx nn.Context) nn.Processor {
	return &Processor{
		BaseProcessor: nn.NewBaseProcessor(m, ctx, false),
		States:        nil,
	}
}

// SetInitialState sets the initial state of the recurrent network.
// It panics if one or more states are already present.
func (p *Processor) SetInitialState(state *State) {
	if len(p.States) > 0 {
		log.Fatal("deltarnn: the initial state must be set before any input")
	}
	p.States = append(p.States, state)
}

// Forward performs the forward step for each input and returns the result.
func (p *Processor) Forward(xs ...ag.Node) []ag.Node {
	ys := make([]ag.Node, len(xs))
	for i, x := range xs {
		s := p.forward(x)
		p.States = append(p.States, s)
		ys[i] = s.Y
	}
	return ys
}

// LastState returns the last state of the recurrent network.
// It returns nil if there are no states.
func (p *Processor) LastState() *State {
	n := len(p.States)
	if n == 0 {
		return nil
	}
	return p.States[n-1]
}

// d1 = beta1 * w (dot) x + beta2 * wRec (dot) yPrev
// d2 = alpha * w (dot) x * wRec (dot) yPrev
// c = tanh(d1 + d2 + bc)
// p = sigmoid(w (dot) x + bp)
// y = f(p * c + (1 - p) * yPrev)
func (p *Processor) forward(x ag.Node) (s *State) {
	m := p.Model.(*Model)
	g := p.Graph
	s = new(State)
	yPrev := p.prev()
	wx := g.Mul(m.W, x)
	if yPrev == nil {
		s.D1 = g.Prod(m.Beta1, wx)
		s.C = g.Tanh(g.Add(s.D1, m.B))
		s.P = g.Sigmoid(g.Add(wx, m.BPart))
		s.Y = g.Tanh(g.Prod(s.P, s.C))
	} else {
		wyRec := g.Mul(m.WRec, yPrev)
		s.D1 = g.Add(g.Prod(m.Beta1, wx), g.Prod(m.Beta2, wyRec))
		s.D2 = g.Prod(g.Prod(m.Alpha, wx), wyRec)
		s.C = g.Tanh(g.Add(g.Add(s.D1, s.D2), m.B))
		s.P = g.Sigmoid(g.Add(wx, m.BPart))
		s.Y = g.Tanh(g.Add(g.Prod(s.P, s.C), g.Prod(g.ReverseSub(s.P, g.NewScalar(1.0)), yPrev)))
	}
	return
}

func (p *Processor) prev() (yPrev ag.Node) {
	s := p.LastState()
	if s != nil {
		yPrev = s.Y
	}
	return
}
