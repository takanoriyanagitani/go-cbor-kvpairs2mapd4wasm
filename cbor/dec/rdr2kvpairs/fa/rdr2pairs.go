package rdr2pairs

import (
	"context"
	"errors"
	"io"

	fa "github.com/fxamacker/cbor/v2"
	pm "github.com/takanoriyanagitani/go-cbor-kvpairs2mapd4wasm"
)

type Decoder struct{ *fa.Decoder }

func (d Decoder) ToPairsSource() pm.PairsSource {
	return func(ctx context.Context) pm.CBORPairs {
		return func(yield func(*pm.CBORPair, error) bool) {
			var pair pm.CBORPair
			for {
				err := d.Decoder.Decode(&pair)
				if errors.Is(err, io.EOF) {
					return
				}

				if !yield(&pair, err) {
					return
				}
			}
		}
	}
}

func RdrToSrc(_ context.Context, rdr io.Reader) pm.PairsSource {
	return Decoder{Decoder: fa.NewDecoder(rdr)}.ToPairsSource()
}

var ReaderToSource pm.ReaderToSource = RdrToSrc
