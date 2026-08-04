package pmap4wa0

import (
	"context"
	"errors"

	pm "github.com/takanoriyanagitani/go-cbor-kvpairs2mapd4wasm"
	"github.com/tetratelabs/wazero"
	wa "github.com/tetratelabs/wazero/api"
)

var (
	ErrNoFunc    error = errors.New("no such function")
	ErrInvlState error = errors.New("invalid state")
)

type State struct {
	wa.Function

	InputRef  *pm.KvPairRaw
	OutputRef *pm.KvPairRaw
}

//nolint:funlen,cyclop,dupl // less trivial to fix the warning
func (s *State) RegisterAll(
	mbld wazero.HostModuleBuilder,
) wazero.HostModuleBuilder {
	return mbld.
		NewFunctionBuilder().
		WithGoModuleFunction(
			wa.GoModuleFunc(func(
				ctx context.Context,
				amod wa.Module,
				stack []uint64,
			) {
				if nil == s.InputRef {
					var ret int32 = -1
					stack[0] = wa.EncodeI32(ret)
					return
				}

				var wmem wa.Memory = amod.Memory()
				if nil == wmem {
					var ret int32 = -2
					stack[0] = wa.EncodeI32(ret)
					return
				}

				var ulkeyptr uint64 = stack[0]
				var keyptr uint32 = wa.DecodeU32(ulkeyptr)
				var wpage *pm.WasmPage = &s.InputRef.Key
				var keydata []byte = wpage.AsSlice()
				var ok bool = wmem.Write(keyptr, keydata)
				if !ok {
					var ret int32 = -3
					stack[0] = wa.EncodeI32(ret)
					return
				}
				var keylen int32 = int32(wpage.Size)
				stack[0] = wa.EncodeI32(keylen)
			}),
			[]wa.ValueType{wa.ValueTypeI32},
			[]wa.ValueType{wa.ValueTypeI32},
		).
		Export("key2wasm").
		NewFunctionBuilder().
		WithGoModuleFunction(
			wa.GoModuleFunc(func(
				ctx context.Context,
				amod wa.Module,
				stack []uint64,
			) {
				if nil == s.InputRef {
					var ret int32 = -1
					stack[0] = wa.EncodeI32(ret)
					return
				}

				var wmem wa.Memory = amod.Memory()
				if nil == wmem {
					var ret int32 = -2
					stack[0] = wa.EncodeI32(ret)
					return
				}

				var ulvalptr uint64 = stack[0]
				var valptr uint32 = wa.DecodeU32(ulvalptr)
				var wpage *pm.WasmPage = &s.InputRef.Val
				var valdata []byte = wpage.AsSlice()
				var ok bool = wmem.Write(valptr, valdata)
				if !ok {
					var ret int32 = -3
					stack[0] = wa.EncodeI32(ret)
					return
				}
				var vallen int32 = int32(wpage.Size)
				stack[0] = wa.EncodeI32(vallen)
			}),
			[]wa.ValueType{wa.ValueTypeI32},
			[]wa.ValueType{wa.ValueTypeI32},
		).
		Export("val2wasm").
		NewFunctionBuilder().
		WithGoModuleFunction(
			wa.GoModuleFunc(func(
				ctx context.Context,
				amod wa.Module,
				stack []uint64,
			) {
				if nil == s.OutputRef {
					var ret int32 = -1
					stack[0] = wa.EncodeI32(ret)
					return
				}

				var wmem wa.Memory = amod.Memory()
				if nil == wmem {
					var ret int32 = -2
					stack[0] = wa.EncodeI32(ret)
					return
				}

				var ulkeyptr uint64 = stack[0]
				var ulkeylen uint64 = stack[1]
				var keyptr uint32 = wa.DecodeU32(ulkeyptr)
				var keylen uint16 = uint16(wa.DecodeU32(ulkeylen))
				okey, ok := wmem.Read(keyptr, uint32(keylen))
				if !ok {
					var ret int32 = -3
					stack[0] = wa.EncodeI32(ret)
					return
				}

				err := s.OutputRef.TrySetKey(okey)
				if nil != err {
					var ret int32 = -4
					stack[0] = wa.EncodeI32(ret)
					return
				}

				stack[0] = 0
			}),
			[]wa.ValueType{wa.ValueTypeI32, wa.ValueTypeI32},
			[]wa.ValueType{wa.ValueTypeI32},
		).
		Export("wasm2key").
		NewFunctionBuilder().
		WithGoModuleFunction(
			wa.GoModuleFunc(func(
				ctx context.Context,
				amod wa.Module,
				stack []uint64,
			) {
				if nil == s.OutputRef {
					var ret int32 = -1
					stack[0] = wa.EncodeI32(ret)
					return
				}

				var wmem wa.Memory = amod.Memory()
				if nil == wmem {
					var ret int32 = -2
					stack[0] = wa.EncodeI32(ret)
					return
				}

				var ulvalptr uint64 = stack[0]
				var ulvallen uint64 = stack[1]
				var valptr uint32 = wa.DecodeU32(ulvalptr)
				var vallen uint16 = uint16(wa.DecodeU32(ulvallen))
				oval, ok := wmem.Read(valptr, uint32(vallen))
				if !ok {
					var ret int32 = -3
					stack[0] = wa.EncodeI32(ret)
					return
				}

				err := s.OutputRef.TrySetVal(oval)
				if nil != err {
					var ret int32 = -4
					stack[0] = wa.EncodeI32(ret)
					return
				}

				stack[0] = 0
			}),
			[]wa.ValueType{wa.ValueTypeI32, wa.ValueTypeI32},
			[]wa.ValueType{wa.ValueTypeI32},
		).
		Export("wasm2val")
}

func (s *State) ToPairMapper() pm.PairMapper {
	return func(ctx context.Context, ipage, opage *pm.KvPairRaw) error {
		if nil == s.Function {
			return ErrInvlState
		}
		s.InputRef = ipage
		s.OutputRef = opage
		_, err := s.Function.Call(ctx)
		return err
	}
}

type WasmModule struct{ wa.Module }

func (m WasmModule) GetFunc(name string) (wa.Function, error) {
	var wfunc wa.Function = m.Module.ExportedFunction(name)
	if nil == wfunc {
		return nil, ErrNoFunc
	}
	return wfunc, nil
}

func (m WasmModule) ToState(mapName string) (State, error) {
	wfunc, err := m.GetFunc(mapName)
	return State{
		Function:  wfunc,
		InputRef:  nil,
		OutputRef: nil,
	}, err
}

type Compiled struct{ wazero.CompiledModule }

func (c Compiled) Instantiate(
	ctx context.Context,
	rtm wazero.Runtime,
	cfg wazero.ModuleConfig,
) (wa.Module, error) {
	return rtm.InstantiateModule(ctx, c.CompiledModule, cfg)
}

func (c Compiled) ToModule(
	ctx context.Context,
	rtm wazero.Runtime,
	cfg wazero.ModuleConfig,
) (WasmModule, error) {
	wmod, err := c.Instantiate(ctx, rtm, cfg)
	return WasmModule{Module: wmod}, err
}

type WasmBytes []byte

func (w WasmBytes) Compile(
	ctx context.Context,
	rtm wazero.Runtime,
) (wazero.CompiledModule, error) {
	return rtm.CompileModule(ctx, w)
}

func (w WasmBytes) ToCompiled(
	ctx context.Context,
	rtm wazero.Runtime,
) (Compiled, error) {
	cmod, err := w.Compile(ctx, rtm)
	return Compiled{CompiledModule: cmod}, err
}
