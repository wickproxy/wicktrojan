package main

import (
	"io"
	"math/rand"
	"time"
)

const (
	shapeSizeMax = 16384

	probSkip         = 0.5
	probNotShapeBase = 0.4
)

var (
	simpleExpSimple = []float64{32, 256, 758, 1024, 2048, 4096, 16328, 65536}
	simpleVarSimple = []float64{4, 16, 32, 64, 128, 192, 1024, 2048}

	shapeExpSimple = []float64{}
	shapeVarSimple = []float64{}
	shapeNotShape  float64
)

type shapeWriter struct {
	r      *rand.Rand
	writer io.Writer
}

func initShape() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range simpleExpSimple {
		if r.Float64() > probSkip {
			shapeExpSimple = append(shapeExpSimple, float64(simpleExpSimple[i])*(0.7+0.6*rand.Float64()))
			shapeVarSimple = append(shapeVarSimple, float64(simpleVarSimple[i])*(0.7+0.6*rand.Float64()))
		}
	}
	shapeNotShape = probNotShapeBase + (1-probNotShapeBase)*rand.Float64()
	debug("[shape] shaping flow into following normally distribution:", shapeExpSimple, shapeVarSimple)
}

func (sw *shapeWriter) init(writer io.Writer) {
	sw.writer = writer
	sw.r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (sw shapeWriter) Write(payload []byte) (int, error) {
	nr := len(payload)
	onr := nr

	ns := 0
	for ns < len(payload) {
		if ns >= onr {
			break
		}
		needShape := sw.r.Float64() >= shapeNotShape
		if !needShape {
			nw, err := sw.writer.Write(payload[ns:])
			ns += nw
			return ns, err
		}

		nr = len(payload[ns:])
		idx := 0
		for idx < len(shapeExpSimple) {
			if nr >= int(0.5*shapeExpSimple[idx]) && nr <= int(1.5*shapeExpSimple[idx]) {
				break
			}
			idx++
		}
		idx = idx - 1
		if idx < 0 || idx >= len(shapeVarSimple) {
			nw, err := sw.writer.Write(payload[ns:])
			ns += nw
			return ns, err
		}

		cut := int(sw.r.NormFloat64()*float64(shapeVarSimple[idx]) + shapeExpSimple[idx])
		if cut >= nr {
			nw, err := sw.writer.Write(payload[ns:])
			ns += nw
			return ns, err
		}
		nw, err := sw.writer.Write(payload[ns : ns+cut])
		ns += nw
		if err != nil {
			return ns, err
		}
	}
	return ns, nil
}
