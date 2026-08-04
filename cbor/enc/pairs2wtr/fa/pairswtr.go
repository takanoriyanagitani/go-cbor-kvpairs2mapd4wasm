package pairswtr

import (
	"context"
	"io"

	fa "github.com/fxamacker/cbor/v2"
	pm "github.com/takanoriyanagitani/go-cbor-kvpairs2mapd4wasm"
)

type Encoder struct{ *fa.Encoder }

func (d Encoder) ToPairsWriter() pm.PairsWriter {
	return func(ctx context.Context, pairs pm.CBORPairs) error {
		for pair, err := range pairs {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if nil != err {
				return err
			}

			werr := d.Encoder.Encode(pair)
			if nil != werr {
				return werr
			}
		}

		return nil
	}
}

func WtrToPairsWtr(_ context.Context, wtr io.Writer) pm.PairsWriter {
	return Encoder{Encoder: fa.NewEncoder(wtr)}.ToPairsWriter()
}

var WtrToPairsWriter pm.WtrToPairsWriter = WtrToPairsWtr
