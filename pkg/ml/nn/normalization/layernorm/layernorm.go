// Copyright 2019 spaGO Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package layernorm

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

// Reference: "Layer normalization" by Jimmy Lei Ba, Jamie Ryan Kiros, and Geoffrey E Hinton (2016).
// (https://arxiv.org/pdf/1607.06450.pdf)
type Model struct {
	W *nn.Param `type:"weights"`
	B *nn.Param `type:"biases"`
}

func New(size int) *Model {
	return &Model{
		W: nn.NewParam(mat.NewEmptyVecDense(size)),
		B: nn.NewParam(mat.NewEmptyVecDense(size)),
	}
}

type Processor struct {
	nn.BaseProcessor
	w   ag.Node
	b   ag.Node
	eps ag.Node
}

func (m *Model) NewProc(g *ag.Graph, opt ...interface{}) nn.Processor {
	p := &Processor{
		BaseProcessor: nn.BaseProcessor{
			Model:             m,
			Mode:              nn.Training,
			Graph:             g,
			FullSeqProcessing: false,
		},
		w:   g.NewWrap(m.W),
		b:   g.NewWrap(m.B),
		eps: g.NewScalar(1e-5), // avoid underflow errors
	}
	p.init(opt)
	return p
}

func (p *Processor) init(opt []interface{}) {
	if len(opt) > 0 {
		log.Fatal("layernorm: invalid init options")
	}
}

// y = (x - E\[x\]) / sqrt(VAR\[x\] + [EPS]) * g + b
func (p *Processor) Forward(xs ...ag.Node) []ag.Node {
	g := p.Graph
	ys := make([]ag.Node, len(xs))
	for i, x := range xs {
		mean := g.ReduceMean(x)
		dev := g.SubScalar(x, mean)
		stdDev := g.Sqrt(g.Add(g.ReduceMean(g.Square(dev)), p.eps))
		ys[i] = g.Add(g.Prod(g.DivScalar(dev, stdDev), p.w), p.b)
	}
	return ys
}
