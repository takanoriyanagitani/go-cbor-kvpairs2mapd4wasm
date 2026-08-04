package kvpairs2mapd4wasm

import (
	"context"
	"errors"
	"io"
	"iter"
)

var (
	ErrOOM error = errors.New("too many bytes")
)

type WasmPage struct {
	Storage [65536]byte
	Size    uint16
}

func (p *WasmPage) AsSlice() []byte { return p.Storage[:p.Size] }

func (p *WasmPage) TrySet(data []byte) error {
	var sz int = len(data)
	if sz > 65536 {
		return ErrOOM
	}

	p.Size = uint16(sz)
	copy(p.Storage[:], data)

	return nil
}

func (p *WasmPage) Set(other *WasmPage) {
	p.Size = other.Size
	copy(p.Storage[:], other.AsSlice())
}

func (p *WasmPage) Reset() {
	p.Size = 0
	var empty [65536]byte
	p.Storage = empty
}

type KvPairRaw struct {
	Key WasmPage
	Val WasmPage
}

type KvPairs iter.Seq2[*KvPairRaw, error]

type CBORPair [2][]byte

type PairsWriter func(context.Context, CBORPairs) error

func (p *KvPairRaw) TrySetKey(key []byte) error { return p.Key.TrySet(key) }
func (p *KvPairRaw) TrySetVal(val []byte) error { return p.Val.TrySet(val) }

func (p *KvPairRaw) Set(other *KvPairRaw) {
	p.Key.Set(&other.Key)
	p.Val.Set(&other.Val)
}

func (p *KvPairRaw) Reset() {
	p.Key.Reset()
	p.Val.Reset()
}

func (p *KvPairRaw) WriteToPair(pair *CBORPair) {
	pair[0] = pair[0][:0]
	pair[1] = pair[1][:0]

	var key []byte = p.Key.AsSlice()
	var val []byte = p.Val.AsSlice()

	pair[0] = append(pair[0], key...)
	pair[1] = append(pair[1], val...)
}

type PairMapper func(ctx context.Context, ipage, opage *KvPairRaw) error

func (m PairMapper) MapPairs(ctx context.Context, pairs KvPairs) KvPairs {
	return func(yield func(*KvPairRaw, error) bool) {
		var opage KvPairRaw

		for ipage, err := range pairs {
			if err != nil {
				yield(nil, err)
				return
			}

			merr := m(ctx, ipage, &opage)
			if !yield(&opage, merr) {
				return
			}
		}
	}
}

func MapperIdent(_ context.Context, ipage, opage *KvPairRaw) error {
	opage.Set(ipage)
	return nil
}

func MapperEmpty(_ context.Context, ipage, opage *KvPairRaw) error {
	opage.Reset()
	return nil
}

var PairMapperNop PairMapper = MapperIdent
var PairMapperReset PairMapper = MapperEmpty

func (p *KvPairRaw) TrySetPair(pair *CBORPair) error {
	return errors.Join(
		p.TrySetKey(pair[0]),
		p.TrySetVal(pair[1]),
	)
}

type CBORPairs iter.Seq2[*CBORPair, error]

func (p KvPairs) ToCBORPairs() CBORPairs {
	return func(yield func(*CBORPair, error) bool) {
		var cpair CBORPair
		cpair[0] = []byte{}
		cpair[1] = []byte{}
		for ipage, err := range p {
			if err != nil {
				yield(nil, err)
				return
			}

			ipage.WriteToPair(&cpair)
			if !yield(&cpair, nil) {
				return
			}
		}
	}
}

func (p KvPairs) ToMapd(ctx context.Context, mapper PairMapper) KvPairs {
	return mapper.MapPairs(ctx, p)
}

func (p CBORPairs) ToKvPairs() KvPairs {
	return func(yield func(*KvPairRaw, error) bool) {
		var kv KvPairRaw
		for pair, err := range p {
			if nil != err {
				yield(nil, err)
				return
			}

			serr := kv.TrySetPair(pair)
			if !yield(&kv, serr) {
				return
			}
		}
	}
}

func (p CBORPairs) ToWriter(ctx context.Context, wtr PairsWriter) error {
	return wtr(ctx, p)
}

type PairsSource func(context.Context) CBORPairs

type ReaderToSource func(context.Context, io.Reader) PairsSource

type WtrToPairsWriter func(context.Context, io.Writer) PairsWriter

type ReaderToSourceToPairsToMapdToWriter struct {
	ReaderToSource
	PairMapper
	WtrToPairsWriter
}

func (r ReaderToSourceToPairsToMapdToWriter) ReaderToWriter(
	ctx context.Context,
	rdr io.Reader,
	wtr io.Writer,
) error {
	var psrc PairsSource = r.ReaderToSource(ctx, rdr)
	var pwtr PairsWriter = r.WtrToPairsWriter(ctx, wtr)
	var cpairs CBORPairs = psrc(ctx)
	var kpairs KvPairs = cpairs.ToKvPairs()
	var kvmapd KvPairs = kpairs.ToMapd(ctx, r.PairMapper)
	var opairs CBORPairs = kvmapd.ToCBORPairs()
	return opairs.ToWriter(ctx, pwtr)
}
