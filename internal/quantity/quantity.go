// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package quantity

import "fmt"

type Format string

const (
	Decimal Format = "Decimal"
	Binary  Format = "Binary"
)

type Quantity struct {
	Format Format
	value  int64
}

func New(format Format, value int64) *Quantity {
	return &Quantity{
		Format: format,
		value:  value,
	}
}

var stringByScale = map[int32]string{
	3:  "K",
	6:  "M",
	9:  "G",
	15: "P",
	18: "E",
}

func (q *Quantity) String() string {
	var divisor int64
	switch q.Format {
	case Decimal:
		divisor = 1000
	case Binary:
		divisor = 1024
	}

	var (
		scale int32
		value = q.value
	)
	scaleHasString := func() bool {
		_, ok := stringByScale[scale]
		return ok
	}

	for (scale == 0 || scaleHasString()) && value > divisor*divisor {
		value /= divisor
		scale += 3
	}

	scaleString, ok := stringByScale[scale]
	if ok && q.Format == Decimal {
		scaleString += "i"
	}

	return fmt.Sprintf("%d%s", value, scaleString)
}

func (q *Quantity) Value() int64 {
	return q.value
}
